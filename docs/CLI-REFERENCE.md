# MAVLink Anywhere CLI Reference

MAVLink Anywhere uses shell entry points plus a standalone dashboard binary. It
does not ship a unified `mavlink-anywhere <command>` operator CLI in this
release.

Main entry points:

- `install_mavlink_router.sh` installs upstream `mavlink-routerd` from source.
- `configure_mavlink_router.sh` configures routing, systemd, and the optional dashboard.
- `mavlink-router-cli.sh` provides small local service helpers.
- `/opt/mavlink-anywhere/mavlink-anywhere` is the Go dashboard server binary,
  not the fleet/operator command wrapper.

## `install_mavlink_router.sh`

```bash
sudo ./install_mavlink_router.sh [OPTIONS]
```

Installs upstream `mavlink-router` from source. It may run `apt`, clone a build
tree under the current user's home directory, compile with Meson/Ninja, and stop
service components during installation.

Options:

| Option | Description |
|--------|-------------|
| `--skip-swap` | Do not create or resize swap while compiling |
| `--force`, `-f` | Rebuild/reinstall even when `mavlink-routerd` already exists |
| `--help`, `-h` | Show help and exit without side effects |

Examples:

```bash
sudo ./install_mavlink_router.sh
sudo ./install_mavlink_router.sh --force
sudo ./install_mavlink_router.sh --skip-swap
```

## `configure_mavlink_router.sh`

```bash
sudo ./configure_mavlink_router.sh [OPTIONS]
```

Modes:

| Mode | Description |
|------|-------------|
| default | Interactive mode with prompts |
| `--auto` | Auto-detect settings, minimal prompts |
| `--headless` | No prompts; all settings via CLI |

Routing options:

| Option | Description | Default |
|--------|-------------|---------|
| `--uart DEVICE` | UART device path | auto-detected |
| `--baud RATE` | UART baud rate | `57600` |
| `--endpoints LIST` | Comma-separated UDP endpoints | standard local endpoints |
| `--gcs-ip IP` | Add GCS endpoint on port `24550` | unset |
| `--input-type TYPE` | `uart` or `udp` | `uart` |
| `--input-address ADDR` | UDP input bind address | `0.0.0.0` |
| `--input-port PORT` | UDP input port | `14550` |
| `--skip-serial-check` | Skip serial port prerequisite check | unset |
| `--debug` | Enable debug output | unset |

Dashboard options:

| Option | Description | Default |
|--------|-------------|---------|
| `--skip-dashboard` | Skip dashboard installation | unset |
| `--install-dashboard` | Install/update dashboard only | unset |
| `--dashboard-listen HOST:PORT` | Dashboard listen address | `127.0.0.1:9070` |
| `--dashboard-auth-user USER` | Browser login username | `admin` when generated |
| `--dashboard-auth-password PASSWORD` | Read browser login password from this argument; not recommended | unset |
| `--dashboard-auth-password-file PATH` | Read browser login password from a root-readable file | unset |
| `--dashboard-auth-password-stdin` | Read browser login password from stdin | unset |
| `--dashboard-auth-hash HASH` | Use an existing bcrypt browser password hash | unset |
| `--dashboard-auth-prompt` | Prompt twice for browser login password | unset |
| `--dashboard-generate-password` | Generate browser login password and print once | unset |
| `--dashboard-disable-auth` | Remove browser login config; use only on trusted isolated networks | unset |
| `--dashboard-open-lab-mode` | No browser login and no API token for remote dashboard mutations | unset |
| `--dashboard-api-token TOKEN` | Configure machine API bearer token; prefer file/generate options | unset |
| `--dashboard-api-token-file PATH` | Read machine API bearer token from a root-readable file | unset |
| `--dashboard-generate-api-token` | Generate machine API bearer token and print once | unset |
| `--dashboard-disable-api-token` | Remove machine API bearer token from dashboard env | unset |
| `--dashboard-ufw-rule`, `--ufw-rule` | If UFW is active and dashboard is remote, allow the dashboard TCP port | unset |

Examples:

```bash
sudo ./configure_mavlink_router.sh

sudo ./configure_mavlink_router.sh --auto --gcs-ip 192.168.1.100

sudo ./configure_mavlink_router.sh --headless \
  --uart /dev/ttyS0 \
  --baud 57600 \
  --endpoints "127.0.0.1:14540,127.0.0.1:14569,192.168.1.100:24550"

sudo ./configure_mavlink_router.sh --headless \
  --input-type udp \
  --input-port 14550 \
  --endpoints "127.0.0.1:14540"

sudo ./configure_mavlink_router.sh --install-dashboard \
  --dashboard-listen 0.0.0.0:9070 \
  --dashboard-auth-user operator \
  --dashboard-auth-prompt
```

Dashboard install behavior:

- downloads the matching release binary (`arm6`, `arm64`, `amd64`) when available
- smoke-tests the release asset's password-hash path on the target host before
  using it for dashboard auth
- falls back to a local Go build if Go is installed
- continues with router-only setup if the dashboard is unavailable
- generates browser Basic Auth automatically when remote exposure is requested
  and no dashboard auth exists
- preserves existing browser auth and machine API token values in
  `/etc/mavlink-anywhere/dashboard.env` unless explicitly replaced or removed
- blocks `--dashboard-open-lab-mode` from downgrading an already protected env
- opens UFW only when explicitly requested with `--dashboard-ufw-rule`

## `mavlink-router-cli.sh`

```bash
./mavlink-router-cli.sh [command]
```

Commands:

| Command | Description |
|---------|-------------|
| `status` | Show systemd status and current config |
| `logs`, `log` | Follow live `mavlink-router` logs |
| `restart` | Restart `mavlink-router` |
| `stop` | Stop `mavlink-router` |
| `start` | Start `mavlink-router` |
| `config`, `show` | Show `/etc/mavlink-router/main.conf` and env file |
| `edit` | Edit the config file and optionally restart |
| `endpoints`, `ep` | Quick edit UDP endpoints |
| `reconfigure`, `reconfig` | Run `configure_mavlink_router.sh` again |
| `help`, `--help`, `-h` | Show helper help |

Examples:

```bash
./mavlink-router-cli.sh status
./mavlink-router-cli.sh logs
sudo ./mavlink-router-cli.sh restart
sudo ./mavlink-router-cli.sh reconfigure
```

## Fleet Profile Automation

Fleet profile reconciliation is exposed through the dashboard API rather than a
shell subcommand. This keeps node hardware-source settings local while allowing
MDS Fleet Ops to manage shared endpoint policy.

Loopback example:

```bash
curl -s http://127.0.0.1:9070/api/v1/profiles/summary

curl -s -X POST http://127.0.0.1:9070/api/v1/profiles/import \
  -H 'Content-Type: application/json' \
  -d '{"mode":"fleet-merge","dry_run":true,"baseline":{"kind":"mavlink-anywhere-profile","schemaVersion":"1","endpoints":[]}}'

curl -s -X POST http://127.0.0.1:9070/api/v1/profiles/apply \
  -H 'Content-Type: application/json' \
  -d '{"dry_run_id":"mla-example","confirmation":{"acknowledged_risks":true,"confirmation_token":"dry-run-token"}}'
```

Remote machine clients use `MAVLINK_ANYWHERE_API_TOKEN` with
`Authorization: Bearer ...` or `X-Mavlink-Anywhere-Token`. Remote browser
mutations use dashboard Basic Auth and the bundled dashboard JavaScript adds
`X-Sidecar-CSRF`.

Modes:

| Mode | Behavior |
|------|----------|
| `observe` | Validate and report only; apply is rejected |
| `local` | Node-local dashboard/API remains authoritative; apply is rejected |
| `fleet-merge` | Apply named baseline endpoints while preserving local extra endpoints and hardware input |
| `fleet-strict` | Apply baseline endpoints and prune local extra outputs only after advanced confirmation |

## Endpoint Format

Endpoints are specified as `IP:PORT` pairs, comma-separated:

```text
127.0.0.1:14540,127.0.0.1:14569,192.168.1.100:24550
```

Standard endpoints:

| Port | Service | Mode | Description |
|------|---------|------|-------------|
| `14550` | `gcs_listen` | UDP server | Default ad-hoc GCS listener |
| `14540` | MAVSDK | UDP normal | Local MAVSDK connection |
| `14569` | mavlink2rest | UDP normal | Local REST bridge |
| `12550` | Local | UDP normal | Local monitoring/debugging |
| `24550` | GCS VPN | UDP normal | Remote ground station over VPN |
| `5760` | TCP server | TCP | Dynamic multi-client TCP access and dashboard probe |

Notes:

- `gcs_listen` on `14550/udp` is ad-hoc server access and is not a replacement
  for explicit local outputs.
- An explicit outbound UDP endpoint to `remote-ip:14550` can coexist with local
  `gcs_listen`.
- Avoid feeding the same remote GCS from both paths at the same time.

## Configuration Files

| File | Description |
|------|-------------|
| `/etc/mavlink-router/main.conf` | Main mavlink-router configuration |
| `/etc/default/mavlink-router` | Router environment variables |
| `/etc/mavlink-anywhere/dashboard.env` | Dashboard browser auth/API token env |
| `/etc/systemd/system/mavlink-router.service` | Router systemd service |
| `/etc/systemd/system/mavlink-anywhere-dashboard.service` | Dashboard systemd service |

## See Also

- [DASHBOARD.md](DASHBOARD.md)
- [UART-SETUP.md](UART-SETUP.md)
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- [Main README](../README.md)
