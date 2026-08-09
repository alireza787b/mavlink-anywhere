# First board setup

Use this checklist for a new Raspberry Pi, Jetson, or Debian/Ubuntu companion computer.

## 1. Connect and update

```bash
ssh pi@PI_IP
sudo apt update
```

## 2. Get MAVLink Anywhere

```bash
git clone https://github.com/alireza787b/mavlink-anywhere.git
cd mavlink-anywhere
sudo ./install_mavlink_router.sh
```

The installer builds the upstream `mavlink-routerd` service. This can take several minutes on a small Pi.

## 3. Connect the flight controller

For a Pixhawk UART connection, wire TX to RX, RX to TX, and ground to ground. Confirm that the voltage level is safe for both boards. Do not power a flight controller from an unknown companion-computer rail.

For a USB adapter, connect it before configuration and look for `/dev/ttyUSB0` or `/dev/ttyACM0`.

More detail: [UART setup](UART-SETUP.md).

## 4. Configure

```bash
sudo ./configure_mavlink_router.sh
```

Follow the prompts. On a Raspberry Pi, setup can enable UART and disable the Linux serial console. That boot change requires one reboot:

```bash
sudo reboot
```

Reconnect, return to the checkout, and run configure again.

## 5. Check the router

```bash
mla status
mla logs
```

Press `Ctrl+C` to leave logs.

Connect QGroundControl to the board's IP on UDP `14550`. QGroundControl must send first because the default `gcs_listen` route learns the most recent UDP sender.

## 6. Dashboard access

The dashboard is local-only by default. From your computer:

```bash
ssh -L 9070:127.0.0.1:9070 pi@PI_IP
```

Then open `http://127.0.0.1:9070`.

For a trusted LAN or VPN:

```bash
sudo mla dashboard expose
```

Undo exposure:

```bash
sudo mla dashboard hide
```

Reset its password:

```bash
sudo mla dashboard password reset
```

## 7. Add destinations

Examples:

```bash
sudo mla endpoint add mavsdk 127.0.0.1 14540
sudo mla endpoint add qgc_vpn 100.80.10.20 24550
mla endpoint list
```

Use addresses that belong to your real device or VPN. Documentation examples are placeholders.

## Update this board later

```bash
cd ~/mavlink-anywhere
git fetch --tags origin
git switch main
git pull --ff-only
sudo ./configure_mavlink_router.sh --install-dashboard
mla status
```

This leaves `/etc/mavlink-router/main.conf`, dashboard credentials, tokens, and the saved local/exposed state in place.

## Final checklist

- `mla status` says the router is running.
- The input device and baud match the Pixhawk telemetry port.
- QGroundControl receives a heartbeat.
- Only required routes are enabled.
- `mla dashboard status` says `local only` unless you intentionally exposed it.
- A remotely exposed dashboard has a browser login.
