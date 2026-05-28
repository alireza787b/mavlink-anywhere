# Security Policy

MAVLink Anywhere is intended to run on trusted companion-computer networks. The
dashboard binds to `127.0.0.1:9070` by default. Expose it to a LAN/VPN only when
the surrounding network is trusted.

## Current Posture

- Endpoint and route configuration can affect real vehicle telemetry paths.
- Do not expose mutation APIs or dashboard controls to public networks.
- Prefer loopback, NetBird/VPN, firewall rules, SSH tunnels, or an authenticated
  reverse proxy for remote access.
- Remote browser access uses HTTP Basic Auth when
  `MAVLINK_ANYWHERE_DASHBOARD_USER` and
  `MAVLINK_ANYWHERE_DASHBOARD_PASSWORD_BCRYPT` are configured. The configure
  script stores only the bcrypt password hash in
  `/etc/mavlink-anywhere/dashboard.env`.
- Browser-authenticated mutating requests must come from the dashboard JavaScript
  and include `X-Sidecar-CSRF`; bearer-token machine clients are exempt.
- Remote machine mutations use `MAVLINK_ANYWHERE_API_TOKEN` with
  `Authorization: Bearer ...` or `X-Mavlink-Anywhere-Token`.
- `MAVLINK_ANYWHERE_ALLOW_UNAUTHENTICATED_MUTATIONS=true` is an explicit
  open-lab override and is ignored when dashboard auth or an API token is also
  configured.

## Credential Operations

- Set or rotate the browser password with `sudo ./configure_mavlink_router.sh
  --install-dashboard --dashboard-auth-user USER --dashboard-auth-prompt`.
- Headless installs should use `--dashboard-auth-password-file PATH` with a
  root-readable file, not a command-line password.
- If an operator is locked out, SSH to the node, rerun the configure script with
  a new password, or use `--dashboard-disable-auth` only on an isolated trusted
  network.
- Set or rotate the machine API token with `--dashboard-generate-api-token` or
  `--dashboard-api-token-file PATH`; store the resulting token in the fleet
  orchestrator secret store, not in git.

## Deferred Hardening

Future work should add:

- CIDR allowlists for GCS, NetBird, admin LAN, and field laptop subnets
- Caddy/reverse-proxy guidance for serving MAVLink Anywhere beside MDS

Report security issues privately to `p30planets@gmail.com`.
