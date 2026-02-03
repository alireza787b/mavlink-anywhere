# MAVLink-Anywhere Troubleshooting Guide

This guide covers common issues and solutions for mavlink-anywhere and mavlink-router.

## Quick Diagnostics

Run these commands to gather diagnostic information:

```bash
# Check overall status
./mavlink-anywhere status

# Check service logs
./mavlink-anywhere logs -n 50

# Test serial connection
./mavlink-anywhere test

# View configuration
cat /etc/mavlink-router/main.conf
```

---

## Installation Issues

### Build Fails with "Dependency systemd not found"

**Symptom:**
```
meson.build:XX: ERROR: Dependency "systemd" not found
```

**Cause:** pkg-config cannot find systemd on some Debian versions (Trixie, Bookworm).

**Solution:** The install script automatically handles this by explicitly setting the systemd directory. If you're running an older version, update mavlink-anywhere:
```bash
cd mavlink-anywhere
git pull
sudo ./install_mavlink_router.sh
```

---

### Build Fails with Out of Memory

**Symptom:**
```
c++: fatal error: Killed signal terminated program cc1plus
```

**Cause:** Not enough RAM for compilation (common on Pi Zero).

**Solution:** The install script automatically increases swap to 2GB. If issues persist:
```bash
# Manually increase swap
sudo dphys-swapfile swapoff
sudo sed -i 's/CONF_SWAPSIZE=.*/CONF_SWAPSIZE=2048/' /etc/dphys-swapfile
sudo dphys-swapfile setup
sudo dphys-swapfile swapon

# Retry installation
sudo ./install_mavlink_router.sh
```

---

### "dphys-swapfile: command not found"

**Cause:** Newer Raspberry Pi OS versions use standard swap instead of dphys-swapfile.

**Solution:** This is normal. The install script automatically detects this and uses the standard swap method instead.

---

## Serial/UART Issues

### Device Not Found

**Symptom:**
```
UART device not found: /dev/ttyS0
```

**Solutions:**

1. **Check if UART is enabled:**
   ```bash
   ls -la /dev/serial* /dev/tty*
   ```

2. **Enable UART via raspi-config:**
   ```bash
   sudo raspi-config
   # Interface Options → Serial Port
   # Login shell: No
   # Hardware: Yes
   # Reboot
   ```

3. **Check boot config:**
   ```bash
   grep enable_uart /boot/config.txt  # or /boot/firmware/config.txt
   ```
   Should show: `enable_uart=1`

---

### Permission Denied

**Symptom:**
```
cannot open /dev/ttyS0: Permission denied
```

**Solution:**
```bash
# Add user to dialout group
sudo usermod -aG dialout $USER

# Log out and back in, or:
newgrp dialout

# Verify
groups
# Should include: dialout
```

---

### Serial Console Blocking UART

**Symptom:** Device exists but mavlink-router fails to get data, or data is corrupted.

**Diagnosis:**
```bash
cat /boot/cmdline.txt  # or /boot/firmware/cmdline.txt
```
If you see `console=serial0` or `console=ttyS0`, the console is blocking the UART.

**Solution:**
1. Edit cmdline.txt and remove console=serial0
2. Or use raspi-config to disable login shell over serial

---

### No MAVLink Data Received

**Symptoms:**
- mavlink-router starts but no data
- `./mavlink-anywhere test` shows no data

**Checklist:**

1. **Check wiring:**
   - TX and RX must be crossed (FC TX → Pi RX)
   - Common ground connected
   - Check voltage levels (3.3V vs 5V)

2. **Check baud rate:**
   - Must match flight controller setting
   - Try common rates: 57600, 115200

3. **Check flight controller:**
   - MAVLink enabled on TELEM port
   - Correct baud rate set
   - TELEM port not in GPS mode

4. **Check for serial console:**
   ```bash
   ps aux | grep getty
   ```
   If you see getty on ttyS0 or ttyAMA0, disable it:
   ```bash
   sudo systemctl stop serial-getty@ttyS0.service
   sudo systemctl disable serial-getty@ttyS0.service
   ```

---

### Wrong UART Device

**Pi 4 with Bluetooth:**
- `/dev/ttyS0` is the mini UART (less reliable)
- `/dev/ttyAMA0` is the better PL011 UART (used by Bluetooth)

**Solution:** Disable Bluetooth to free PL011:
```bash
# Add to /boot/config.txt
dtoverlay=disable-bt

# Reboot
sudo reboot
```

---

## Service Issues

### Service Won't Start

**Symptom:**
```
Failed to start MAVLink Router Service.
```

**Diagnosis:**
```bash
sudo systemctl status mavlink-router
sudo journalctl -u mavlink-router -n 100
```

**Common causes:**

1. **Invalid configuration:**
   ```bash
   cat /etc/mavlink-router/main.conf
   # Check for syntax errors
   ```

2. **Device doesn't exist:**
   ```bash
   ls -la /dev/ttyS0  # or your configured device
   ```

3. **Port already in use:**
   ```bash
   sudo lsof -i :5760  # Check TCP port
   sudo lsof -i :14550 # Check UDP port
   ```

---

### Service Keeps Restarting

**Symptom:** Service starts then stops repeatedly.

**Diagnosis:**
```bash
sudo journalctl -u mavlink-router -f
```

**Common causes:**

1. **Serial device disconnected or not ready:**
   - Add delay before starting
   - Check physical connection

2. **Permission issues:**
   - Check device permissions
   - Ensure service runs as correct user

---

### Service Running But No Data Routing

**Diagnosis:**
```bash
# Check if mavlink-router is receiving data
sudo journalctl -u mavlink-router -f

# Check UDP ports are listening
ss -ulpn | grep mavlink
```

**Solutions:**

1. **Check endpoint configuration:**
   ```bash
   cat /etc/mavlink-router/main.conf
   ```
   Ensure IP addresses and ports are correct.

2. **Check firewall:**
   ```bash
   sudo ufw status
   # If active, allow required ports
   sudo ufw allow 14550/udp
   sudo ufw allow 14540/udp
   ```

---

## Network Issues

### Can't Connect from QGroundControl

**Checklist:**

1. **Check IP address:**
   ```bash
   ip addr show
   ```

2. **Check mavlink-router is running:**
   ```bash
   ./mavlink-anywhere status
   ```

3. **Check endpoint configuration:**
   ```bash
   grep -A3 "UdpEndpoint" /etc/mavlink-router/main.conf
   ```

4. **Check firewall:**
   ```bash
   sudo ufw status
   sudo iptables -L -n
   ```

5. **Test connectivity:**
   ```bash
   # From QGC machine
   nc -vzu <PI_IP> 14550
   ```

---

### Data Visible Locally But Not Remotely

**Cause:** Endpoint configured with wrong IP or network issue.

**Solutions:**

1. **Check endpoint IP in config:**
   ```bash
   cat /etc/mavlink-router/main.conf
   ```
   For remote access, use the Pi's IP (not 127.0.0.1).

2. **Check routing:**
   ```bash
   ip route
   ping <GCS_IP>
   ```

3. **For VPN connections:**
   - Verify VPN is connected
   - Use VPN IP address in endpoint

---

## Configuration Issues

### Changes Not Taking Effect

**Solution:** Restart the service after configuration changes:
```bash
sudo systemctl restart mavlink-router
```

---

### Lost Configuration After Reboot

**Cause:** Configuration wasn't saved properly.

**Verification:**
```bash
ls -la /etc/mavlink-router/
cat /etc/mavlink-router/main.conf
```

If files are missing, run configuration again:
```bash
sudo ./configure_mavlink_router.sh
```

---

## Performance Issues

### High Latency

**Possible causes:**

1. **Network congestion**
2. **CPU overload on Pi**
3. **USB serial adapter issues**

**Solutions:**

1. Check CPU usage:
   ```bash
   top
   ```

2. Use native UART instead of USB:
   ```bash
   sudo ./configure_mavlink_router.sh --uart /dev/ttyS0
   ```

3. Reduce number of endpoints

---

### Data Loss or Corruption

**Possible causes:**

1. **Baud rate mismatch**
2. **Mini UART instability**
3. **Poor wiring**

**Solutions:**

1. Use PL011 UART (disable Bluetooth)
2. Check physical connections
3. Use shielded cables for long runs
4. Lower baud rate if needed

---

## Getting Help

If issues persist:

1. **Gather diagnostics:**
   ```bash
   ./mavlink-anywhere status
   ./mavlink-anywhere logs -n 100 > mavlink_logs.txt
   cat /etc/mavlink-router/main.conf
   uname -a
   cat /etc/os-release
   ```

2. **Check existing issues:**
   https://github.com/alireza787b/mavlink-anywhere/issues

3. **Open new issue** with:
   - Raspberry Pi model
   - OS version
   - mavlink-anywhere version
   - Configuration used
   - Error messages
   - Steps to reproduce

---

## See Also

- [UART-SETUP.md](UART-SETUP.md) - Serial port configuration
- [CLI-REFERENCE.md](CLI-REFERENCE.md) - Command reference
- [Main README](../README.md) - Project overview
