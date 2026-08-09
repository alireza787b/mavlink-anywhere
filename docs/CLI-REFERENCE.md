# CLI command reference

`mla` is the day-to-day command for MAVLink Anywhere.

```bash
mla help
```

Reading commands work as your normal user. Commands that change settings tell you to rerun them with `sudo`.

## Status and service

```bash
mla status
mla logs
sudo mla restart
sudo mla stop
sudo mla start
```

`mla logs` follows live router logs. Press `Ctrl+C` to leave.

## Routes and endpoints

Show the complete endpoint help:

```bash
mla help endpoint
```

List the current input, listeners, and outputs:

```bash
mla endpoint list
```

Add an outbound UDP route:

```bash
sudo mla endpoint add NAME IP PORT
```

Example for QGroundControl:

```bash
sudo mla endpoint add qgc 192.168.1.50 14550
```

Example for local MAVSDK:

```bash
sudo mla endpoint add mavsdk 127.0.0.1 14540
```

Edit an existing route:

```bash
sudo mla endpoint edit qgc 100.80.10.20 24550
```

Temporarily disable and later restore a route:

```bash
sudo mla endpoint disable qgc
sudo mla endpoint enable qgc
```

Remove a route:

```bash
sudo mla endpoint remove qgc
```

The optional last argument is the UDP mode:

```bash
sudo mla endpoint add field_listener 0.0.0.0 14600 server
```

- `normal` sends MAVLink to the given address. It is the default.
- `server` listens on this computer and replies to the most recent UDP sender.

Each change creates a timestamped config backup and updates `/etc/default/mavlink-router`. If the router was already running, MLA restarts it. A failed restart restores the previous files.

## Flight-data input

```bash
mla help input
mla input show
```

Set a UART input:

```bash
sudo mla input uart /dev/serial0 57600
sudo mla input uart /dev/ttyUSB0 115200
```

Set a UDP input:

```bash
sudo mla input udp 0.0.0.0 14560
```

The input is the Pixhawk, SITL, or upstream router that supplies the original MAVLink stream. Outputs receive copies of that stream.

## Dashboard and authentication

```bash
mla help dashboard
mla dashboard status
```

Network access:

```bash
sudo mla dashboard expose
sudo mla dashboard expose 9071
sudo mla dashboard hide
```

`hide` is the direct answer to “how do I undo expose?” It keeps the UI running but binds it to localhost only.

Dashboard process control:

```bash
sudo mla dashboard off
sudo mla dashboard on
```

`off` stops and disables the dashboard only. It does not stop `mavlink-router`. `on` restores the last saved listen address, so use `hide` before `off` if the next start must be local-only.

Browser password:

```bash
mla dashboard password status
sudo mla dashboard password reset
sudo mla dashboard password reset operator
sudo mla dashboard password generate
```

`reset` securely asks for a password twice. `generate` prints a random password once.

Machine API token:

```bash
mla dashboard token status
sudo mla dashboard token create
sudo mla dashboard token rotate
sudo mla dashboard token remove
```

- `create` refuses to overwrite an existing token.
- `rotate` replaces the token. Update every connected client afterward.
- `remove` disables bearer-token access for machine clients.
- Status commands never print a saved password hash or token.

## Raw config

```bash
mla help config
mla config show
mla config check
mla config path
sudo mla config edit
```

`config edit` is the safe raw method. It:

1. keeps the previous config in memory;
2. opens `$EDITOR`, or `nano` when `$EDITOR` is unset;
3. validates endpoint names, addresses, ports, modes, and local bind conflicts;
4. creates a timestamped backup;
5. syncs `/etc/default/mavlink-router`;
6. restarts a router that was already running;
7. restores the previous files if validation or restart fails.

The main file is `/etc/mavlink-router/main.conf`.

## First-time configure script

The longer configure script remains available for first installation, automation, and unusual boards:

```bash
sudo ./configure_mavlink_router.sh
sudo ./configure_mavlink_router.sh --help
```

Common noninteractive examples follow.

UART input:

```bash
sudo ./configure_mavlink_router.sh --headless \
  --uart /dev/serial0 \
  --baud 57600 \
  --endpoints "127.0.0.1:14540,127.0.0.1:14569"
```

UDP input:

```bash
sudo ./configure_mavlink_router.sh --headless \
  --input-type udp \
  --input-address 0.0.0.0 \
  --input-port 14560 \
  --endpoints "127.0.0.1:14540"
```

Update only the dashboard and install/refresh the `mla` command:

```bash
sudo ./configure_mavlink_router.sh --install-dashboard
```

Advanced dashboard authentication flags are retained for automation. See `sudo ./configure_mavlink_router.sh --help`; beginners should use `mla dashboard ...`.

## Files and services

| Item | Path or name |
|---|---|
| Router config | `/etc/mavlink-router/main.conf` |
| Router companion env | `/etc/default/mavlink-router` |
| Dashboard settings and auth | `/etc/mavlink-anywhere/dashboard.env` |
| Router service | `mavlink-router` |
| Dashboard service | `mavlink-anywhere-dashboard` |

The installed `mla` script and config helper are root-owned copies under
`/usr/local/lib/mavlink-anywhere`; they are not executed through a user-writable
checkout.

Do not paste `dashboard.env` into support chats. It contains the machine API token and browser password hash.
