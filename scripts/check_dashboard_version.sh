#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXPECTED="${1:-$(. "$ROOT_DIR/lib/common.sh"; printf '%s' "$MAVLINK_ANYWHERE_VERSION")}"
EXPECTED="${EXPECTED#v}"
URL="${MAVLINK_ANYWHERE_DASHBOARD_URL:-http://127.0.0.1:9070/api/v1/status}"
BIN="${MAVLINK_ANYWHERE_DASHBOARD_BINARY:-/opt/mavlink-anywhere/mavlink-anywhere}"

version_from_api() {
    curl -fsS --max-time 3 "$URL" 2>/dev/null | python3 -c 'import json,sys
try:
    print(str(json.load(sys.stdin).get("version","")).lstrip("v"))
except Exception:
    raise SystemExit(1)'
}

version_from_binary() {
    "$BIN" --version 2>/dev/null | awk '{print $3}' | sed 's/^v//'
}

actual=""
if command -v curl >/dev/null 2>&1 && command -v python3 >/dev/null 2>&1; then
    actual="$(version_from_api || true)"
fi
if [[ -z "$actual" && -x "$BIN" ]]; then
    actual="$(version_from_binary || true)"
fi
if [[ -z "$actual" ]]; then
    echo "MAVLink Anywhere dashboard version: unknown" >&2
    exit 1
fi

if [[ "$actual" != "$EXPECTED" ]]; then
    echo "MAVLink Anywhere dashboard version drift: expected ${EXPECTED}, got ${actual}" >&2
    exit 2
fi

echo "MAVLink Anywhere dashboard version ok: ${actual}"
