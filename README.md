![Build](https://github.com/tarow/dockdns/actions/workflows/ci.yaml/badge.svg)
[![go report](https://goreportcard.com/badge/github.com/Tarow/dockdns)](https://goreportcard.com/report/github.com/Tarow/dockdns)
[![Renovate](https://img.shields.io/badge/renovate-enabled-brightgreen.svg)](https://renovatebot.com)

# DockDNS - (Dynamic) DNS Client based on Docker Labels

DockDNS is a DNS updater, which supports configuring DNS records through Docker labels.
DockDNS supports Cloudflare and Technitium DNS Server as providers.

## Features

- Dynamic DNS updates
- Static DNS entries (e.g. with a static IP address)
- Static DNS record configuration based on a config file
- Dynamic DNS record configuration based on Docker labels
- IPv4 & IPv6 support
- CNAME support
- Supports multiple zones
- Provider-specific overrides for CNAME and Proxied settings
- Automatically trigger DNS updates when labeled containers start & stop

## Configuration

The app configuration as well as the static domain entries are read from a configuration file (see [example configuration](config.example.yaml)).

```yaml
interval: 600 # Optional, the update interval in seconds. Defaults to 600. Negative interval will result in one-shot invocations.
debounceTime: 10 # Optional, delay the DNS update run until no new trigger event has been received for <<debounceTime>> seconds. This is used to avoid multiple DNS update runs when multiple containers are started/stopped in succession, e.g. by Docker Compose. Defaults to 10.
maxDebounceTime: 600 # Optional, if debouncing exceeds <<maxDebounceTime>> seconds, do not delay the DNS update beyond that. This avoids delaying the DNS update forever, e.g. in case of crash-looping containers that generate trigger events indefinitely. Defaults to 600.

webUI: false # Optional, enables a WebUI (port 8080) that lists the scanned domains and current settings. Defaults to false

log:
  level: info # Optional, Log level, one of 'debug', 'info', 'warn' or 'error'. Defaults to 'info'
  format: simple # Optional, output of the log format, 'simple' or 'json'. Defaults to 'simple'

zones: # Zone configuration (multiple zones can be provided)
  - name: somedomain.com # Root name of the zone
    id: cloudflare-prod # Optional: custom ID for override labels (defaults to zone name if not set)
    provider: cloudflare # Name of the provider. Supported: cloudflare, technitium

## Technitium DNS Provider

DockDNS supports Technitium DNS Server through its HTTP API. This allows full management of A, AAAA, and CNAME records.

Example zone configuration:

```yaml
zones:
  - name: internal.example.com
    id: technitium-internal           # Optional: custom ID for override labels
    provider: technitium
    apiURL: http://192.168.1.10:5380  # Technitium DNS Server URL
    # Option 1: Use API token (recommended)
    apiToken: ...                      # Technitium API token
    # Option 2: Use username/password (if no apiToken is set)
    # apiUsername: admin               # Technitium username
    # apiPassword: ...                 # Technitium password
    # skipTLSVerify: true              # Skip TLS certificate verification (for self-signed certs)
```

### Supported Operations
- Create, Update, Delete A, AAAA, and CNAME records
- List all records in a zone
- Get a specific record by name and type
- Automatic authentication and session management

> Note: The Technitium provider uses the HTTP API and requires either an API token or a user account with appropriate permissions.
    apiToken: ... # API Token, needs permission 'Zone.Zone' (read) and Zone.DNS (edit). Can also be passed as environment variable: SOMEDOMAIN_COM_API_TOKEN
    zoneID: ... # Optional: If not set, will be fetched dynamically. ZoneID of this zone. Can also be passed as environment variable: SOMEDOMAIN_COM_ZONE_ID
dns:
  a: true # Update IPv4 addresses
  aaaa: false # Update IPv6 addresses
  defaultTTL: 300 # Optional, default TTL for all records. Defaults to 300
  purgeUnknown: true # Optional, delete unknown records. Defaults to false.

# Static domain configuration (optional)
domains:
  - name: "*.somedomain.com" # IPs for A and AAAA records will be determined dynamically
    comment: "Some comment" # Record comment

  - name: "somedomain.com"
    a: 10.0.0.2 # Static IPv4 address
    aaaa: ::1 # Static IPv6 address

  - name: "alt.somedomain.com" # Name of the CNAME record
    cname: "main.somedomain.com" # Target of the CNAME record

  # Example with provider-specific overrides
  - name: "app.somedomain.com"
    cname: "default-target.somedomain.com"
    cnameOverrides:
      technitium-internal: "internal-target.local"  # Different CNAME for Technitium zone
    proxied: false
    proxiedOverrides:
      cloudflare-prod: true  # Enable Cloudflare proxy for this zone
```

## Dynamic Domains

Domains can also be configured using Docker labels.
Supported labels:
| Label | Example | Description |
|----------------------|-------------------------------------------|-------------|
| dockdns.name | dockdns.name=somedomain.com | Domain name (required) |
| dockdns.a | dockdns.a=127.0.0.1 | Static IPv4 address |
| dockdns.aaaa | dockdns.aaaa=::1 | Static IPv6 address |
| dockdns.cname | dockdns.cname=target.otherdomain.com | CNAME target |
| dockdns.ttl | dockdns.ttl=600 | Record TTL |
| dockdns.proxied | dockdns.proxied=false | Cloudflare proxy (default) |
| dockdns.comment | dockdns.comment=Some comment | Record comment |

#### Zone-Specific Overrides

You can override any field for specific zones/providers using the format: `dockdns.<zone-id>.<field>=value`

The `<zone-id>` should match the zone's `id` field (or zone `name` if `id` is not set).

| Label | Example | Description |
|----------------------|-------------------------------------------|-------------|
| dockdns.\<id\>.a | dockdns.cloudflare-prod.a=10.0.0.5 | Zone-specific IPv4 address |
| dockdns.\<id\>.aaaa | dockdns.zone1.aaaa=2001:db8::5 | Zone-specific IPv6 address |
| dockdns.\<id\>.cname | dockdns.technitium-internal.cname=target.local | Zone-specific CNAME target |
| dockdns.\<id\>.ttl | dockdns.zone1.ttl=600 | Zone-specific TTL |
| dockdns.\<id\>.proxied | dockdns.cloudflare-prod.proxied=true | Zone-specific proxied setting |
| dockdns.\<id\>.comment | dockdns.zone1.comment=Zone comment | Zone-specific comment |

Example:
```yaml
# Group settings by zone
dockdns.cloudflare-prod.a=10.0.0.5
dockdns.cloudflare-prod.proxied=true
dockdns.technitium-internal.a=192.168.1.10
dockdns.technitium-internal.ttl=600
```

---

The `dockdns.name` label can also contain a comma separated list of names, e.g.

```ini
dockdns.name="somedomain.com,www.somedomain.com"
```

If no explicit IP address is set, the public IP will be fetched and set automatically (DynDNS).
If a `CNAME` is set, `A` and `AAAA` settings are ignored.

## Installation

### Go install

```
go install github.com/Tarow/dockdns@latest
```

By default, DockDNS looks for a `config.yaml` in the current folder. The location of the configuration file can be overwritten using the `-config` flag:

```
dockdns -config /path/to/config.yaml
```

### Docker

```bash
# Set HOST_HOSTNAME to the actual hostname of the server
docker run -e HOST_HOSTNAME=myserver1 -v ./config.yaml:/app/config.yaml -v /var/run/docker.sock:/var/run/docker.sock:ro ghcr.io/tarow/dockdns:latest
```

### Docker Compose

```yaml
services:
  dockdns:
    image: ghcr.io/tarow/dockdns:latest
    restart: unless-stopped
    environment:
      - HOST_HOSTNAME=myserver1  # Set to the actual hostname of this server
    volumes:
      - ./config.yaml:/app/config.yaml
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

### Nix

```bash
nix run github:tarow/dockdns
```

---

Note: To avoid direct socket access, you can also set environment variable `DOCKER_HOST`.
For example, if you use [docker-socket-proxy](https://github.com/Tecnativa/docker-socket-proxy), you may set the environment variable `DOCKER_HOST=tcp://docker-socket-proxy:2375`.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `DOCKER_HOST` | Docker daemon socket (e.g., `tcp://docker-socket-proxy:2375`) |
| `HOST_HOSTNAME` | Physical host machine's hostname. Used in Technitium DNS record comments to identify which host created the record. Set to `$(hostname)` when running in Docker. |
| `HOSTNAME_OVERRIDE` | Alternative to `HOST_HOSTNAME` for setting the hostname in record comments. |

## Development

You need:

* Go 1.23
* Templ