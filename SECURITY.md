# Security Policy

MAVLink Anywhere is intended to run on trusted companion-computer networks. The
dashboard binds to `127.0.0.1:9070` by default. Expose it to a LAN/VPN only when
the surrounding network is trusted.

## Current Posture

- Endpoint and route configuration can affect real vehicle telemetry paths.
- Do not expose mutation APIs or dashboard controls to public networks.
- Prefer loopback, NetBird/VPN, firewall rules, SSH tunnels, or an authenticated
  reverse proxy for remote access.

## Deferred Hardening

Future work should add:

- optional dashboard login similar to MDS
- bearer-token protection for mutation APIs
- CIDR allowlists for GCS, NetBird, admin LAN, and field laptop subnets
- Caddy/reverse-proxy guidance for serving MAVLink Anywhere beside MDS
- SSH/CLI recovery steps if auth configuration locks out an operator

Report security issues privately to `p30planets@gmail.com`.

