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

- Set or rotate the browser password with `sudo mla dashboard password reset`.
- Return an exposed dashboard to loopback with `sudo mla dashboard hide`, or
  stop the UI with `sudo mla dashboard off`.
- Create or rotate the machine token with `sudo mla dashboard token create` or
  `sudo mla dashboard token rotate`.
- Headless installs should use `--dashboard-auth-password-file PATH` with a
  root-readable file or `--dashboard-auth-password-stdin`, not a command-line
  password. `--dashboard-auth-password PASSWORD` is available only as a
  non-recommended lab/automation escape hatch.
- If an operator is locked out, SSH to the node and run the password reset.
- Store machine tokens in the fleet orchestrator secret store, not in git.
- The longer configure-script credential flags remain available for headless
  automation; see `sudo ./configure_mavlink_router.sh --help`.

## Deferred Hardening

Future work should add:

- CIDR allowlists for GCS, NetBird, admin LAN, and field laptop subnets
- Caddy/reverse-proxy guidance for serving MAVLink Anywhere beside MDS

Report security issues privately to `p30planets@gmail.com`.
