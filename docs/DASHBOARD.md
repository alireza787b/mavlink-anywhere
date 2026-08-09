# Dashboard and authentication

The dashboard shows router health and manages MAVLink routes from a browser. It is optional; `mavlink-router` keeps working when the dashboard is off.

## Safest access: local or SSH tunnel

The default address is:

```text
http://127.0.0.1:9070
```

From another computer, create an SSH tunnel:

```bash
ssh -L 9070:127.0.0.1:9070 pi@PI_IP
```

Keep that terminal open and visit `http://127.0.0.1:9070` on your computer.

## Expose on a trusted network

```bash
sudo mla dashboard expose
```

This binds the dashboard to all network interfaces on port `9070`. If there is no browser password, MLA creates one and prints it once.

Check the saved state:

```bash
mla dashboard status
```

Do not expose port `9070` directly to the public internet. Use a VPN, SSH tunnel, and firewall.

## Revert exposure

Return to localhost-only access:

```bash
sudo mla dashboard hide
```

This is the normal way to undo `dashboard expose`. It does not stop MAVLink routing.

To stop the UI completely:

```bash
sudo mla dashboard off
```

Start it again:

```bash
sudo mla dashboard on
```

Important: `on` uses the last saved address. Run `hide` before `off` when you want the next start to remain local-only.

## Reset the browser password

Choose your own password:

```bash
sudo mla dashboard password reset
```

The default username is `admin`. To change it while resetting:

```bash
sudo mla dashboard password reset operator
```

Generate a random password instead:

```bash
sudo mla dashboard password generate
```

Only the bcrypt password hash is stored in `/etc/mavlink-anywhere/dashboard.env`.

## Machine API token

Browser users sign in with the username and password. Programs such as MDS Fleet Ops use a bearer token for remote changes.

```bash
sudo mla dashboard token create
sudo mla dashboard token rotate
sudo mla dashboard token remove
```

Save the value printed by `create` or `rotate`. MLA status does not reveal it later.

Example API call:

```bash
curl -H "Authorization: Bearer $MAVLINK_ANYWHERE_API_TOKEN" \
  http://PI_IP:9070/api/v1/status
```

Browser changes send a CSRF header automatically. A direct mutating API call also needs it when using browser Basic Auth:

```bash
curl -u operator \
  -H 'X-Sidecar-CSRF: 1' \
  -X POST http://PI_IP:9070/api/v1/service/restart
```

## What the UI shows

The first screen is intentionally short:

- router state and restart control;
- current flight-data input;
- simple MAVLink health;
- listeners and output routes;
- warnings that need attention.

Profiles, raw config, logs, and board details are collapsed as advanced tools.

Every dashboard route/input change creates a backup. If the router was running, the dashboard restarts it. A failed restart restores the previous files.

## Install or update the dashboard

```bash
cd ~/mavlink-anywhere
sudo ./configure_mavlink_router.sh --install-dashboard
```

The update preserves the saved listen address, browser credentials, API token, and router config.

Prebuilt binaries are published for Linux `arm6`, `arm64`, and `amd64`. If a matching download is unavailable and Go is installed, setup tries a local build. The router still works if the optional dashboard cannot be installed.

## Firewall

If UFW is active and you intentionally expose the dashboard:

```bash
sudo ufw allow 9070/tcp
```

To remove that firewall permission later:

```bash
sudo ufw delete allow 9070/tcp
```

`dashboard hide` makes the service local-only even if the old firewall rule still exists, but removing unused rules is clearer.

## API summary

Read-only:

```text
GET /api/v1/status
GET /api/v1/diagnostics
GET /api/v1/config
GET /api/v1/endpoints
GET /api/v1/input
GET /api/v1/logs/recent
GET /api/v1/profiles/export
GET /api/v1/profiles/summary
```

Changes:

```text
PUT    /api/v1/config
POST   /api/v1/endpoints
PUT    /api/v1/endpoints/{name}
PATCH  /api/v1/endpoints/{name}
DELETE /api/v1/endpoints/{name}
PUT    /api/v1/input
POST   /api/v1/service/start
POST   /api/v1/service/stop
POST   /api/v1/service/restart
```

Fleet profile endpoints remain available for MDS. They use dry-run plus confirmation for managed fleet changes. See the API responses and source tests when integrating a machine client.

## Troubleshooting

Dashboard status and logs:

```bash
mla dashboard status
sudo journalctl -u mavlink-anywhere-dashboard -n 100 --no-pager
```

If the browser repeatedly asks for credentials, reset the password and try a private browser window:

```bash
sudo mla dashboard password reset
```

If the page is unreachable, confirm whether it is deliberately local-only:

```bash
mla dashboard status
```

See [Troubleshooting](TROUBLESHOOTING.md) for router and serial checks.
