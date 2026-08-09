# MAVLink Anywhere

Route MAVLink between a Pixhawk or simulator, a Linux companion computer, and the tools that need flight data.

![MAVLink Anywhere logo](assets/brand/mavlink-anywhere-logo.svg)

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-3.1.0-blue.svg)](configure_mavlink_router.sh)

## Install on a Raspberry Pi or Linux computer

```bash
git clone https://github.com/alireza787b/mavlink-anywhere.git
cd mavlink-anywhere
sudo ./install_mavlink_router.sh
sudo ./configure_mavlink_router.sh
```

Follow the prompts. On a Raspberry Pi, the serial setup may ask for one reboot. After rebooting, run the last command again.

Check the result:

```bash
mla status
```

By default:

- Pixhawk data comes from the serial device chosen during setup.
- QGroundControl can connect to the Pi's IP on UDP port `14550`.
- The optional dashboard runs locally at `http://127.0.0.1:9070`.

## The command you need

```bash
mla help
```

It points to short help for every common task:

```bash
mla help endpoint
mla help input
mla help dashboard
mla help config
```

Use `sudo` for commands that change settings.

## Manage MAVLink routes

List routes:

```bash
mla endpoint list
```

Send MAVLink to QGroundControl at `192.168.1.50`:

```bash
sudo mla endpoint add qgc 192.168.1.50 14550
```

Change, temporarily disable, or remove it:

```bash
sudo mla endpoint edit qgc 100.80.10.20 24550
sudo mla endpoint disable qgc
sudo mla endpoint enable qgc
sudo mla endpoint remove qgc
```

Changes are validated and backed up. If the running router cannot restart, MLA restores the previous files.

## Change the flight-data input

Pixhawk on the Raspberry Pi serial port:

```bash
sudo mla input uart /dev/serial0 57600
```

USB serial adapter:

```bash
sudo mla input uart /dev/ttyUSB0 115200
```

UDP input for SITL or another MAVLink router:

```bash
sudo mla input udp 0.0.0.0 14560
```

## Dashboard access

The dashboard starts local-only. An SSH tunnel is the safest remote option:

```bash
ssh -L 9070:127.0.0.1:9070 pi@PI_IP
```

Then open `http://127.0.0.1:9070` on your computer.

To expose it on a trusted LAN or VPN:

```bash
sudo mla dashboard expose
```

If no browser login exists, this creates one and prints the password once.

To reverse exposure and return to local-only:

```bash
sudo mla dashboard hide
```

To stop and disable only the UI:

```bash
sudo mla dashboard off
```

Turn it back on:

```bash
sudo mla dashboard on
```

These commands do not stop MAVLink routing.

### Reset the dashboard password

```bash
sudo mla dashboard password reset
```

For a generated password instead:

```bash
sudo mla dashboard password generate
```

### Manage a machine API token

Use a token for MDS Fleet Ops or another program that changes routes through the API:

```bash
sudo mla dashboard token create
sudo mla dashboard token rotate
sudo mla dashboard token remove
```

`create` and `rotate` print the token once. Save it in your password manager or client configuration.

## Raw terminal editing

For full control of `main.conf`:

```bash
sudo mla config edit
```

MLA opens `$EDITOR` or `nano`, validates the file, creates a backup, syncs `/etc/default/mavlink-router`, and safely applies the change.

Useful checks:

```bash
mla config show
mla config check
mla config path
```

## Update an existing Raspberry Pi

This updates MAVLink Anywhere without replacing your router config, routes, dashboard password, token, or saved dashboard access mode:

```bash
cd ~/mavlink-anywhere
git fetch --tags origin
git switch main
git pull --ff-only
sudo ./configure_mavlink_router.sh --install-dashboard
mla status
```

If your checkout is somewhere else, use that directory in the first command. Rebuilding upstream `mavlink-routerd` is optional and separate:

```bash
sudo ./install_mavlink_router.sh --force
```

## Ports

| Port | Purpose |
|---|---|
| `14550/udp` | Default device-side listener for a GCS |
| `14540/udp` | Common local MAVSDK output |
| `14569/udp` | Common local mavlink2rest output |
| `24550/udp` | Common explicit remote/VPN GCS output |
| `5760/tcp` | mavlink-router TCP server |
| `9070/tcp` | Optional web dashboard |

## More help

- [CLI command reference](docs/CLI-REFERENCE.md)
- [Dashboard and authentication](docs/DASHBOARD.md)
- [First board setup](docs/BOARD_SETUP.md)
- [UART and Pixhawk wiring](docs/UART-SETUP.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)

MAVLink Anywhere works alongside [Smart Wi-Fi Manager](https://github.com/alireza787b/smart-wifi-manager) for connectivity and [MDS](https://github.com/alireza787b/mavsdk_drone_show) for fleet operations.

## Uninstall

```bash
sudo systemctl disable --now mavlink-anywhere-dashboard mavlink-router
sudo rm /etc/systemd/system/mavlink-anywhere-dashboard.service
sudo rm /etc/systemd/system/mavlink-router.service
sudo rm /usr/local/bin/mla
sudo rm -r /usr/local/lib/mavlink-anywhere
sudo systemctl daemon-reload
```

Configuration is intentionally left in `/etc/mavlink-router` and `/etc/mavlink-anywhere` so it can be recovered. Remove those directories only if you no longer need the saved settings.

Licensed under the [MIT License](LICENSE).
