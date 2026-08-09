# Troubleshooting

Start with these two commands:

```bash
mla status
mla logs
```

Press `Ctrl+C` to leave logs.

## Router is stopped

```bash
sudo mla start
systemctl status mavlink-router --no-pager
```

Check the config:

```bash
mla config check
```

If you recently used raw editing, timestamped backups are beside `/etc/mavlink-router/main.conf`.

## No Pixhawk heartbeat

Check the configured input:

```bash
mla input show
ls -l /dev/serial0 /dev/ttyUSB* /dev/ttyACM* 2>/dev/null
```

Common causes:

- TX and RX are not crossed.
- The boards do not share ground.
- Pixhawk telemetry baud and MLA baud differ.
- Linux is still using the UART as a serial console.
- Another process already owns the serial device.
- The selected device changed after reconnecting a USB adapter.

Check who owns the device:

```bash
sudo lsof /dev/serial0
```

Set the correct input, for example:

```bash
sudo mla input uart /dev/serial0 57600
```

For Raspberry Pi boot/UART details, see [UART setup](UART-SETUP.md).

## Router says “permission denied” for UART

```bash
ls -l /dev/serial0
groups
```

Add your normal account to the serial group, then log out and back in:

```bash
sudo usermod -aG dialout "$USER"
```

The systemd router normally runs with sufficient privileges; a permission error can also mean the service unit was customized.

## QGroundControl does not receive data

List routes:

```bash
mla endpoint list
```

For the default `gcs_listen` listener, configure QGroundControl to contact the companion computer's IP on UDP `14550`. QGroundControl must send first; server-mode UDP replies to the most recent sender.

For a fixed outbound route instead:

```bash
sudo mla endpoint add qgc GCS_IP 14550
```

Replace `GCS_IP` with the actual address. Confirm basic reachability:

```bash
ping -c 3 GCS_IP
```

Check firewalls on both computers. Do not add both listener and outbound routes to the same GCS unless you understand the duplicate-telemetry risk.

## Port already in use

```bash
sudo ss -lntup | grep -E ':(14550|5760|9070)\b'
```

Only one server-mode endpoint can bind the same local address and UDP port. `mla config check` detects this in saved config.

To move a custom listener:

```bash
sudo mla endpoint edit field_listener 0.0.0.0 14600 server
```

## A route change failed

CLI and dashboard changes create a backup before applying. If restart fails, the previous files are restored automatically.

See recent errors:

```bash
sudo journalctl -u mavlink-router -n 100 --no-pager
```

Validate the effective file:

```bash
mla config check
```

Use safe raw editing only when the route commands cannot represent your advanced setting:

```bash
sudo mla config edit
```

## Dashboard is unreachable

```bash
mla dashboard status
sudo systemctl status mavlink-anywhere-dashboard --no-pager
```

If status says `local only`, use an SSH tunnel or intentionally expose it:

```bash
ssh -L 9070:127.0.0.1:9070 pi@PI_IP
```

or:

```bash
sudo mla dashboard expose
```

Reverse network exposure with:

```bash
sudo mla dashboard hide
```

## Dashboard password does not work

Reset it:

```bash
sudo mla dashboard password reset
```

Then use a private browser window to avoid cached Basic Auth credentials.

Dashboard logs:

```bash
sudo journalctl -u mavlink-anywhere-dashboard -n 100 --no-pager
```

## API token does not work

```bash
mla dashboard token status
```

Rotate it and update the client:

```bash
sudo mla dashboard token rotate
```

The token goes in either of these request headers:

```text
Authorization: Bearer TOKEN
X-Mavlink-Anywhere-Token: TOKEN
```

Browser passwords and machine tokens are different credentials.

## Dashboard update failed

```bash
cd ~/mavlink-anywhere
git pull --ff-only
sudo ./configure_mavlink_router.sh --install-dashboard --debug
```

The router is independent and should keep running if the optional dashboard download or build fails.

Check architecture:

```bash
uname -m
```

Published Linux assets support arm6, arm64, and amd64.

## Collect useful diagnostics

These commands do not print the dashboard token or password hash:

```bash
mla status
mla config check
systemctl status mavlink-router --no-pager
sudo journalctl -u mavlink-router -n 100 --no-pager
uname -a
```

Before sharing raw files, remove public IPs, VPN addresses, hostnames, and any contents of `/etc/mavlink-anywhere/dashboard.env`.
