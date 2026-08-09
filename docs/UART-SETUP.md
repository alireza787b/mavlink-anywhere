# UART and Pixhawk setup

Use this guide when MAVLink reaches the companion computer through a Pixhawk TELEM port.

## Wiring

| Flight controller | Raspberry Pi GPIO header |
|---|---|
| TX | RX, GPIO 15, pin 10 |
| RX | TX, GPIO 14, pin 8 |
| GND | GND, pin 6 |

TX and RX must cross. Both boards need a common ground. Confirm the logic voltage and pinout for your exact flight controller before connecting it. Do not connect a power pin unless the full power design is known to be safe.

## Enable Raspberry Pi UART

The easiest method is:

```bash
sudo raspi-config
```

Choose `Interface Options` → `Serial Port`:

1. Login shell over serial: **No**
2. Serial port hardware: **Yes**
3. Finish and reboot

```bash
sudo reboot
```

After reconnecting:

```bash
ls -l /dev/serial0
```

Use `/dev/serial0` when available because it follows the Pi's selected primary UART.

## Configure MLA

```bash
sudo mla input uart /dev/serial0 57600
mla input show
mla status
```

The baud rate must match the Pixhawk TELEM port.

Common values are `57600`, `115200`, and `921600`. Higher is not automatically better; wiring length and electrical noise matter.

## USB serial adapter

USB adapters avoid Raspberry Pi boot UART configuration.

```bash
ls -l /dev/ttyUSB* /dev/ttyACM* 2>/dev/null
sudo mla input uart /dev/ttyUSB0 115200
```

The device name can change when multiple USB serial adapters are connected. For permanent systems, use a stable `/dev/serial/by-id/...` path:

```bash
ls -l /dev/serial/by-id/
```

## Manual Raspberry Pi configuration

Use this only when `raspi-config` is unavailable.

On current Raspberry Pi OS, edit `/boot/firmware/config.txt`; older images may use `/boot/config.txt`:

```ini
enable_uart=1
```

Then remove `console=serial0,...`, `console=ttyAMA0,...`, or `console=ttyS0,...` from the single line in `/boot/firmware/cmdline.txt` or `/boot/cmdline.txt`.

Disable serial getty units if present:

```bash
sudo systemctl disable --now serial-getty@ttyS0.service
sudo systemctl disable --now serial-getty@ttyAMA0.service
sudo reboot
```

Make a backup before manually changing boot files. `cmdline.txt` must remain one line.

## Raspberry Pi Bluetooth and PL011

Some Pi models assign the stable PL011 UART to Bluetooth. If `/dev/ttyS0` is unstable at your chosen baud, you can move Bluetooth away from that UART by adding this to the applicable `config.txt`:

```ini
dtoverlay=disable-bt
```

Reboot afterward. This changes Bluetooth behavior, so use it only when needed.

## Checks when no data arrives

```bash
mla input show
ls -l /dev/serial0
sudo lsof /dev/serial0
sudo journalctl -u mavlink-router -n 100 --no-pager
```

Confirm:

- the Pixhawk telemetry port is enabled for MAVLink;
- its baud matches MLA;
- TX/RX are crossed;
- ground is connected;
- Linux serial console is disabled;
- no other program has opened the device.

See [Troubleshooting](TROUBLESHOOTING.md) for routing and dashboard checks.
