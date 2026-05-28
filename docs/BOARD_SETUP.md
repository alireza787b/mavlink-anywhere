# Board Setup And Dashboard Auth

This guide is for a new Linux companion computer that should run MAVLink
Anywhere with the optional dashboard.

## Recommended Field Setup

1. Create or verify the operator SSH user and sudo policy.
2. Install `mavlink-router`, `curl`, `systemd`, and serial/UDP prerequisites.
3. Clone the repo under a persistent path such as `/opt/mavlink-anywhere`.
4. Run the configure script with browser auth:

```bash
cd /opt/mavlink-anywhere
sudo git fetch --tags origin
sudo git checkout -f v3.0.14
sudo ./configure_mavlink_router.sh --install-dashboard \
  --dashboard-listen 0.0.0.0:9070 \
  --dashboard-auth-user admin \
  --dashboard-auth-prompt \
  --dashboard-ufw-rule
```

Use a non-default username for shared or production deployments. Supply the
actual password out of band; do not paste it into chat, Git, shell history, or
reports.

For noninteractive automation, prefer stdin or a root-readable file:

```bash
printf '%s' "$MAVLINK_DASHBOARD_PASSWORD" | sudo ./configure_mavlink_router.sh --install-dashboard \
  --dashboard-listen 0.0.0.0:9070 \
  --dashboard-auth-user admin \
  --dashboard-auth-password-stdin \
  --dashboard-ufw-rule
```

`--dashboard-auth-password PASSWORD` also exists for constrained lab automation,
but it can leak through shell history and process listings. Prefer prompt,
stdin, file, or bcrypt hash.

## Firewall

The configure script does not change firewall policy unless requested. Add
`--dashboard-ufw-rule` or `--ufw-rule` to allow the dashboard TCP port when UFW
is active and the dashboard listens on a non-loopback address.

Without that flag, use:

```bash
sudo ufw allow 9070/tcp
```

## Release Binaries And Go

Published dashboard binaries are static Go binaries and do not require Go on the
board. Password hashing uses Go's pure-Go bcrypt implementation and does not
require CGO at runtime.

The configure script falls back to building from local source only when the
release asset is unavailable, invalid, or fails the on-board password-hash smoke
test. If you rely on that fallback, install Go in a persistent path such as
`/opt/go` or through the OS package manager. Do not depend on a toolchain under
`/tmp`; many boards mount `/tmp` as tmpfs.

Release maintainers should build assets with:

```bash
./scripts/build_release_assets.sh
```

The build script uses `CGO_ENABLED=0` and smoke-tests password hashing on the
host architecture.

## Verify

```bash
systemctl is-active mavlink-router mavlink-anywhere-dashboard
/opt/mavlink-anywhere/mavlink-anywhere --version
curl -u admin http://127.0.0.1:9070/api/v1/status
./scripts/check_dashboard_version.sh
```

Remote mutation from the browser requires dashboard login plus the bundled
CSRF header. Machine clients should use `MAVLINK_ANYWHERE_API_TOKEN`.

## Drift Checks

Run the version check locally, from cron, or from fleet orchestration:

```bash
/opt/mavlink-anywhere/scripts/check_dashboard_version.sh 3.0.14
```

MDS Fleet Ops also reports installed sidecar versions and sidecar profile drift
for enrolled boards.
