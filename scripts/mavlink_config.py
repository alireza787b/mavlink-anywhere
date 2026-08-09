#!/usr/bin/env python3
"""Safe, dependency-free MAVLink router config operations for the MLA CLI."""

from __future__ import annotations

import argparse
import datetime as dt
import ipaddress
import os
import re
import shutil
import sys
import tempfile
from dataclasses import dataclass
from pathlib import Path


SECTION_RE = re.compile(r"^\s*\[([A-Za-z]+Endpoint)\s+([^\]]+)]\s*(?:[#;].*)?$")
ANY_SECTION_RE = re.compile(r"^\s*\[([^\]]+)]\s*(?:[#;].*)?$")
DISABLED_BEGIN_RE = re.compile(r"^\s*#\s*MAVLINK_ANYWHERE_DISABLED_BEGIN\s+(.+?)\s*$")
DISABLED_END_RE = re.compile(r"^\s*#\s*MAVLINK_ANYWHERE_DISABLED_END\s+(.+?)\s*$")
VALID_NAME_RE = re.compile(r"^[A-Za-z0-9_-]{1,32}$")
KNOWN_TYPES = {"UdpEndpoint", "UartEndpoint", "TcpEndpoint"}


class ConfigError(ValueError):
    pass


@dataclass
class EndpointBlock:
    name: str
    kind: str
    values: dict[str, str]
    start: int
    end: int
    enabled: bool = True


def uncomment_disabled_line(line: str) -> str:
    return re.sub(r"^\s*# ?", "", line, count=1)


def parse_config(text: str) -> tuple[list[str], list[EndpointBlock]]:
    lines = text.splitlines(keepends=True)
    blocks: list[EndpointBlock] = []
    i = 0
    while i < len(lines):
        disabled_match = DISABLED_BEGIN_RE.match(lines[i].rstrip("\r\n"))
        if disabled_match:
            marker_name = disabled_match.group(1).strip()
            end = i + 1
            inner: list[str] = []
            while end < len(lines) and not DISABLED_END_RE.match(lines[end].rstrip("\r\n")):
                inner.append(uncomment_disabled_line(lines[end]))
                end += 1
            if end >= len(lines):
                raise ConfigError(f'disabled endpoint "{marker_name}" has no end marker')
            block = parse_endpoint_lines(inner, i, end + 1, enabled=False)
            if block.name != marker_name:
                raise ConfigError(
                    f'disabled marker names "{marker_name}" but contains endpoint "{block.name}"'
                )
            blocks.append(block)
            i = end + 1
            continue

        section_match = SECTION_RE.match(lines[i].rstrip("\r\n"))
        if section_match and section_match.group(1) in KNOWN_TYPES:
            scan = i + 1
            end = i + 1
            while scan < len(lines):
                if ANY_SECTION_RE.match(lines[scan].rstrip("\r\n")):
                    break
                if DISABLED_BEGIN_RE.match(lines[scan].rstrip("\r\n")):
                    break
                candidate = lines[scan].strip()
                if candidate and not candidate.startswith(("#", ";")):
                    end = scan + 1
                scan += 1
            blocks.append(parse_endpoint_lines(lines[i:end], i, end, enabled=True))
            i = scan
            continue
        i += 1
    return lines, blocks


def parse_endpoint_lines(
    lines: list[str], start: int, end: int, *, enabled: bool
) -> EndpointBlock:
    if not lines:
        raise ConfigError("empty endpoint block")
    match = SECTION_RE.match(lines[0].rstrip("\r\n"))
    if not match or match.group(1) not in KNOWN_TYPES:
        raise ConfigError("invalid endpoint section")
    values: dict[str, str] = {}
    for raw in lines[1:]:
        line = raw.strip()
        if not line or line.startswith(("#", ";")):
            continue
        if "=" not in line:
            raise ConfigError(f'invalid line in endpoint "{match.group(2).strip()}": {line}')
        key, value = line.split("=", 1)
        values[key.strip()] = value.strip()
    return EndpointBlock(
        name=match.group(2).strip(),
        kind=match.group(1),
        values=values,
        start=start,
        end=end,
        enabled=enabled,
    )


def validate(text: str) -> list[EndpointBlock]:
    lines, blocks = parse_config(text)
    general_count = 0
    names: set[str] = set()
    active_count = 0
    server_binds: list[tuple[str, str, int]] = []

    for number, raw in enumerate(lines, start=1):
        stripped = raw.strip()
        if stripped.startswith("[") and not ANY_SECTION_RE.match(stripped):
            raise ConfigError(f"line {number}: malformed section header")
        section = ANY_SECTION_RE.match(stripped)
        if section and section.group(1).strip() == "General":
            general_count += 1

    if general_count != 1:
        raise ConfigError("config must contain exactly one [General] section")
    if not blocks:
        raise ConfigError("config must contain at least one endpoint")

    for block in blocks:
        if not VALID_NAME_RE.fullmatch(block.name):
            raise ConfigError(
                f'endpoint "{block.name}" must use only letters, numbers, _ or - (32 characters maximum)'
            )
        if block.name in names:
            raise ConfigError(f'duplicate endpoint name: "{block.name}"')
        names.add(block.name)
        if not block.enabled:
            continue
        active_count += 1

        if block.kind == "UartEndpoint":
            device = block.values.get("Device", "")
            if not device.startswith("/dev/"):
                raise ConfigError(f'UART endpoint "{block.name}" needs Device=/dev/...')
            parse_positive_int(block.values.get("Baud", ""), "baud", block.name)
            continue

        address = block.values.get("Address", "")
        try:
            ipaddress.ip_address(address)
        except ValueError as exc:
            raise ConfigError(f'endpoint "{block.name}" has invalid IP address: {address}') from exc
        port = parse_port(block.values.get("Port", ""), block.name)
        if block.kind == "UdpEndpoint":
            mode = block.values.get("Mode", "normal").lower()
            if mode not in {"normal", "server"}:
                raise ConfigError(f'endpoint "{block.name}" mode must be normal or server')
            if mode == "server":
                for other_name, other_address, other_port in server_binds:
                    if port == other_port and bind_conflicts(address, other_address):
                        raise ConfigError(
                            f'server endpoints "{other_name}" and "{block.name}" both bind {address}:{port}'
                        )
                server_binds.append((block.name, address, port))

    if active_count == 0:
        raise ConfigError("config must contain at least one enabled endpoint")
    return blocks


def parse_positive_int(value: str, label: str, name: str) -> int:
    try:
        parsed = int(value)
    except ValueError as exc:
        raise ConfigError(f'endpoint "{name}" has invalid {label}: {value}') from exc
    if parsed <= 0:
        raise ConfigError(f'endpoint "{name}" has invalid {label}: {value}')
    return parsed


def parse_port(value: str, name: str) -> int:
    port = parse_positive_int(value, "port", name)
    if port > 65535:
        raise ConfigError(f'endpoint "{name}" port must be between 1 and 65535')
    return port


def bind_conflicts(left: str, right: str) -> bool:
    wildcard = {"0.0.0.0", "::"}
    return left == right or left in wildcard or right in wildcard


def udp_block(name: str, address: str, port: int, mode: str) -> str:
    return (
        f"[UdpEndpoint {name}]\n"
        f"Mode={mode}\n"
        f"Address={address}\n"
        f"Port={port}\n\n"
    )


def uart_block(device: str, baud: int) -> str:
    return f"[UartEndpoint uart]\nDevice={device}\nBaud={baud}\n\n"


def write_atomic(path: Path, content: str, *, backup: bool = True) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    if backup and path.exists():
        stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        backup_path = path.with_name(f"{path.name}.backup.{stamp}")
        shutil.copy2(path, backup_path)
    mode = path.stat().st_mode & 0o777 if path.exists() else 0o644
    fd, temp_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as handle:
            handle.write(content)
        os.chmod(temp_name, mode)
        os.replace(temp_name, path)
    finally:
        if os.path.exists(temp_name):
            os.unlink(temp_name)


def replace_range(text: str, block: EndpointBlock, replacement: str) -> str:
    lines = text.splitlines(keepends=True)
    return "".join(lines[: block.start]) + replacement + "".join(lines[block.end :])


def find_block(blocks: list[EndpointBlock], name: str) -> EndpointBlock:
    for block in blocks:
        if block.name == name:
            return block
    raise ConfigError(f'endpoint "{name}" was not found')


def validate_endpoint_args(name: str, address: str, port: int, mode: str) -> None:
    if not VALID_NAME_RE.fullmatch(name):
        raise ConfigError("name must use only letters, numbers, _ or - (32 characters maximum)")
    try:
        ipaddress.ip_address(address)
    except ValueError as exc:
        raise ConfigError(f"invalid IP address: {address}") from exc
    if port < 1 or port > 65535:
        raise ConfigError("port must be between 1 and 65535")
    if mode not in {"normal", "server"}:
        raise ConfigError("mode must be normal or server")


def sync_env(config_text: str, env_path: Path) -> None:
    blocks = validate(config_text)
    values = {
        "INPUT_TYPE": "",
        "UART_DEVICE": "",
        "UART_BAUD": "",
        "INPUT_ADDRESS": "",
        "INPUT_PORT": "",
        "UDP_ENDPOINTS": "",
    }
    outputs: list[str] = []
    for block in blocks:
        if not block.enabled:
            continue
        if block.kind == "UartEndpoint":
            values["INPUT_TYPE"] = "uart"
            values["UART_DEVICE"] = block.values.get("Device", "")
            values["UART_BAUD"] = block.values.get("Baud", "")
        elif block.kind == "UdpEndpoint":
            mode = block.values.get("Mode", "normal").lower()
            if block.name == "input" and mode == "server":
                values["INPUT_TYPE"] = "udp"
                values["INPUT_ADDRESS"] = block.values.get("Address", "")
                values["INPUT_PORT"] = block.values.get("Port", "")
            elif mode == "normal":
                outputs.append(f'{block.values.get("Address", "")}:{block.values.get("Port", "")}')
    values["UDP_ENDPOINTS"] = ",".join(sorted(outputs))
    content = "# Generated by mavlink-anywhere CLI\n" + "".join(
        f'{key}="{value}"\n' for key, value in values.items()
    )
    write_atomic(env_path, content)


def mutate_config(config_path: Path, env_path: Path, operation) -> None:
    text = config_path.read_text(encoding="utf-8")
    updated = operation(text)
    validate(updated)
    write_atomic(config_path, updated)
    try:
        sync_env(updated, env_path)
    except Exception:
        # The timestamped config backup remains available if env sync fails.
        raise


def command_list(config_path: Path) -> None:
    blocks = validate(config_path.read_text(encoding="utf-8"))
    print(f"{'NAME':<22} {'STATE':<9} {'TYPE':<6} {'MODE':<7} ADDRESS / DEVICE")
    for block in blocks:
        state = "enabled" if block.enabled else "disabled"
        if block.kind == "UartEndpoint":
            target = f'{block.values.get("Device", "?")} @ {block.values.get("Baud", "?")}'
            kind, mode = "UART", "input"
        else:
            target = f'{block.values.get("Address", "?")}:{block.values.get("Port", "?")}'
            kind = "UDP" if block.kind == "UdpEndpoint" else "TCP"
            mode = block.values.get("Mode", "normal")
        print(f"{block.name:<22} {state:<9} {kind:<6} {mode:<7} {target}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Manage mavlink-router main.conf safely")
    parser.add_argument("--config", default="/etc/mavlink-router/main.conf")
    parser.add_argument("--env", default="/etc/default/mavlink-router")
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("list")
    sub.add_parser("validate")
    sub.add_parser("sync-env")

    for command in ("add", "edit"):
        item = sub.add_parser(command)
        item.add_argument("name")
        item.add_argument("address")
        item.add_argument("port", type=int)
        item.add_argument("mode", nargs="?", default="normal", choices=("normal", "server"))
    for command in ("remove", "enable", "disable"):
        item = sub.add_parser(command)
        item.add_argument("name")

    uart = sub.add_parser("input-uart")
    uart.add_argument("device")
    uart.add_argument("baud", type=int, nargs="?", default=57600)
    udp = sub.add_parser("input-udp")
    udp.add_argument("address")
    udp.add_argument("port", type=int)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    config_path = Path(args.config)
    env_path = Path(args.env)
    if not config_path.is_file():
        raise ConfigError(f"config file not found: {config_path}")

    if args.command == "list":
        command_list(config_path)
        return 0
    if args.command == "validate":
        validate(config_path.read_text(encoding="utf-8"))
        print(f"OK: {config_path}")
        return 0
    if args.command == "sync-env":
        sync_env(config_path.read_text(encoding="utf-8"), env_path)
        return 0

    if args.command in {"add", "edit"}:
        validate_endpoint_args(args.name, args.address, args.port, args.mode)

        def add_or_edit(text: str) -> str:
            _, blocks = parse_config(text)
            existing = next((item for item in blocks if item.name == args.name), None)
            replacement = udp_block(args.name, args.address, args.port, args.mode)
            if args.command == "add":
                if existing:
                    raise ConfigError(f'endpoint "{args.name}" already exists; use edit')
                return text.rstrip() + "\n\n" + replacement
            if not existing:
                raise ConfigError(f'endpoint "{args.name}" was not found; use add')
            if existing.kind != "UdpEndpoint":
                raise ConfigError("only UDP endpoints can be edited with this command")
            if not existing.enabled:
                replacement = disabled_block(args.name, replacement)
            return replace_range(text, existing, replacement)

        mutate_config(config_path, env_path, add_or_edit)
        return 0

    if args.command == "remove":
        def remove(text: str) -> str:
            _, blocks = parse_config(text)
            block = find_block(blocks, args.name)
            return replace_range(text, block, "")

        mutate_config(config_path, env_path, remove)
        return 0

    if args.command in {"enable", "disable"}:
        def toggle(text: str) -> str:
            _, blocks = parse_config(text)
            block = find_block(blocks, args.name)
            if (args.command == "enable") == block.enabled:
                return text
            raw_lines = text.splitlines(keepends=True)[block.start : block.end]
            raw = "".join(raw_lines)
            if args.command == "disable":
                return replace_range(text, block, disabled_block(block.name, raw))
            inner = raw.splitlines(keepends=True)[1:-1]
            enabled_raw = "".join(uncomment_disabled_line(line) for line in inner)
            return replace_range(text, block, enabled_raw)

        mutate_config(config_path, env_path, toggle)
        return 0

    if args.command in {"input-uart", "input-udp"}:
        if args.command == "input-uart":
            if not args.device.startswith("/dev/"):
                raise ConfigError("UART device must start with /dev/")
            if args.baud <= 0:
                raise ConfigError("baud must be greater than zero")
            replacement = uart_block(args.device, args.baud)
        else:
            validate_endpoint_args("input", args.address, args.port, "server")
            replacement = udp_block("input", args.address, args.port, "server")

        def set_input(text: str) -> str:
            _, blocks = parse_config(text)
            inputs = [
                block
                for block in blocks
                if block.kind == "UartEndpoint" or (block.kind == "UdpEndpoint" and block.name == "input")
            ]
            if not inputs:
                general_lines = text.splitlines(keepends=True)
                insert_at = 0
                for index, line in enumerate(general_lines):
                    if line.strip() == "[General]":
                        insert_at = index + 1
                        while insert_at < len(general_lines) and not ANY_SECTION_RE.match(general_lines[insert_at].strip()):
                            insert_at += 1
                        break
                general_lines.insert(insert_at, "\n" + replacement)
                return "".join(general_lines)
            result = text
            for block in sorted(inputs, key=lambda item: item.start, reverse=True):
                result = replace_range(result, block, replacement if block is inputs[0] else "")
            return result

        mutate_config(config_path, env_path, set_input)
        return 0

    raise ConfigError(f"unsupported command: {args.command}")


def disabled_block(name: str, raw: str) -> str:
    commented = "".join(f"# {line}" if line.strip() else "#\n" for line in raw.splitlines(keepends=True))
    if raw and not raw.endswith("\n"):
        commented += "\n"
    return (
        f"# MAVLINK_ANYWHERE_DISABLED_BEGIN {name}\n"
        f"{commented}"
        f"# MAVLINK_ANYWHERE_DISABLED_END {name}\n\n"
    )


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ConfigError as exc:
        print(f"Error: {exc}", file=sys.stderr)
        raise SystemExit(2)
