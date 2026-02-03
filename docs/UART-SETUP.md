# UART Setup Guide for Raspberry Pi

This guide explains how to configure the serial/UART port on Raspberry Pi for MAVLink communication with flight controllers.

## Overview

When connecting a flight controller (Pixhawk, etc.) to a Raspberry Pi via UART, you need to:

1. Enable UART hardware
2. Disable the serial console (which normally uses the UART)
3. Ensure correct permissions

## Quick Setup

### Using raspi-config (Recommended)

```bash
sudo raspi-config
```

1. Navigate to: **Interface Options** → **Serial Port**
2. "Would you like a login shell to be accessible over serial?" → **No**
3. "Would you like the serial port hardware to be enabled?" → **Yes**
4. Finish and **Reboot**

After reboot, the UART will be available at `/dev/serial0` or `/dev/ttyS0`.

---

## Raspberry Pi Models and UART Devices

Different Pi models have different default UART configurations:

### Raspberry Pi 5

| Device | Description | Notes |
|--------|-------------|-------|
| `/dev/ttyAMA0` | Primary UART (PL011) | GPIO 14/15, recommended |
| `/dev/serial0` | Symlink to primary | Use this for portability |

### Raspberry Pi 4 / Zero 2 W

| Device | Description | Notes |
|--------|-------------|-------|
| `/dev/ttyAMA0` | Primary UART (PL011) | Used by Bluetooth by default |
| `/dev/ttyS0` | Mini UART | GPIO 14/15 when BT enabled |
| `/dev/serial0` | Symlink | Points to mini UART by default |

**To use the better PL011 UART:**
```bash
# Add to /boot/config.txt (or /boot/firmware/config.txt on newer OS)
dtoverlay=disable-bt
```

### Raspberry Pi 3 / Zero / Older

| Device | Description | Notes |
|--------|-------------|-------|
| `/dev/ttyS0` | Mini UART | Default on GPIO 14/15 |
| `/dev/serial0` | Symlink | Points to ttyS0 |

---

## Manual Configuration

### Step 1: Enable UART

Edit the boot configuration:

```bash
# For Raspberry Pi OS Bullseye and earlier:
sudo nano /boot/config.txt

# For Raspberry Pi OS Bookworm and newer:
sudo nano /boot/firmware/config.txt
```

Add or ensure these lines exist:
```ini
# Enable UART
enable_uart=1

# Optional: Disable Bluetooth to free up PL011 UART
# dtoverlay=disable-bt
```

### Step 2: Disable Serial Console

Edit the kernel command line:

```bash
# For Raspberry Pi OS Bullseye and earlier:
sudo nano /boot/cmdline.txt

# For Raspberry Pi OS Bookworm and newer:
sudo nano /boot/firmware/cmdline.txt
```

Remove any references to the serial console. Remove:
- `console=serial0,115200`
- `console=ttyAMA0,115200`
- `console=ttyS0,115200`

**Before:**
```
console=serial0,115200 console=tty1 root=PARTUUID=xxx ...
```

**After:**
```
console=tty1 root=PARTUUID=xxx ...
```

### Step 3: Disable Serial Getty Service

```bash
sudo systemctl stop serial-getty@ttyS0.service
sudo systemctl disable serial-getty@ttyS0.service
sudo systemctl stop serial-getty@ttyAMA0.service
sudo systemctl disable serial-getty@ttyAMA0.service
```

### Step 4: Add User to dialout Group

```bash
sudo usermod -aG dialout $USER
```

Log out and back in for this to take effect.

### Step 5: Reboot

```bash
sudo reboot
```

---

## Verification

After reboot, verify the configuration:

### Check Device Exists

```bash
ls -la /dev/serial* /dev/ttyS* /dev/ttyAMA*
```

Expected output:
```
lrwxrwxrwx 1 root root 5 Jan 15 10:00 /dev/serial0 -> ttyS0
crw-rw---- 1 root dialout 4, 64 Jan 15 10:00 /dev/ttyS0
```

### Check User Permissions

```bash
groups
```

Should include `dialout`.

### Check Serial Console is Disabled

```bash
cat /boot/cmdline.txt  # or /boot/firmware/cmdline.txt
```

Should NOT contain `console=serial0` or similar.

### Test with mavlink-anywhere

```bash
./mavlink-anywhere test
```

---

## Wiring

Connect your flight controller's TELEM port to the Raspberry Pi GPIO:

| Flight Controller | Raspberry Pi |
|-------------------|--------------|
| TX | GPIO 15 (RXD) - Pin 10 |
| RX | GPIO 14 (TXD) - Pin 8 |
| GND | GND - Pin 6 |

**Important:**
- Cross TX/RX (FC TX → Pi RX, FC RX → Pi TX)
- Ensure voltage levels match (most Pixhawks use 3.3V logic)
- Share a common ground

```
Flight Controller          Raspberry Pi
     TELEM Port               GPIO Header
    ┌─────────┐              ┌─────────────┐
    │   TX    │──────────────│ Pin 10 (RX) │
    │   RX    │──────────────│ Pin 8  (TX) │
    │   GND   │──────────────│ Pin 6 (GND) │
    └─────────┘              └─────────────┘
```

---

## Baud Rate Configuration

Common baud rates for MAVLink:

| Baud Rate | Usage |
|-----------|-------|
| 57600 | Default for many configurations |
| 115200 | Higher bandwidth (recommended if stable) |
| 921600 | High-speed (may require short cables) |

The baud rate must match:
1. Flight controller TELEM port setting
2. mavlink-router configuration

### Setting Baud Rate on Flight Controller

**PX4:**
```
param set SER_TEL1_BAUD 57600
```

**ArduPilot:**
```
param set SERIAL1_BAUD 57
```

---

## Troubleshooting

### Device Not Found

```
ls: cannot access '/dev/serial0': No such file or directory
```

**Solution:** Enable UART in `/boot/config.txt` and reboot.

### Permission Denied

```
cannot open /dev/ttyS0: Permission denied
```

**Solution:** Add user to dialout group and log out/in:
```bash
sudo usermod -aG dialout $USER
logout
```

### No Data Received

Possible causes:
1. Wrong baud rate
2. TX/RX wires swapped
3. MAVLink not enabled on flight controller
4. Serial console still enabled (blocking UART)

### Intermittent Data Loss

With mini UART (`/dev/ttyS0`):
- Mini UART clock is tied to CPU clock
- May be unstable under CPU frequency changes

**Solution:** Use PL011 UART by disabling Bluetooth:
```bash
# Add to /boot/config.txt
dtoverlay=disable-bt
```

---

## Using USB Serial Instead

If UART setup is problematic, you can use a USB-to-serial adapter:

```bash
# Connect USB adapter, then check
ls /dev/ttyUSB* /dev/ttyACM*

# Configure mavlink-anywhere
sudo ./configure_mavlink_router.sh --auto --uart /dev/ttyUSB0
```

USB serial doesn't require boot configuration changes and is often more reliable.

---

## See Also

- [Raspberry Pi UART Documentation](https://www.raspberrypi.com/documentation/computers/configuration.html#configuring-uarts)
- [CLI-REFERENCE.md](CLI-REFERENCE.md) - mavlink-anywhere commands
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues
