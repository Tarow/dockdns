![Build](https://github.com/tarow/dockdns/actions/workflows/ci.yaml/badge.svg)
[![go report](https://goreportcard.com/badge/github.com/Tarow/dockdns)](https://goreportcard.com/report/github.com/Tarow/dockdns)
[![Renovate](https://img.shields.io/badge/renovate-enabled-brightgreen.svg)](https://renovatebot.com)

# DockDNS - (Dynamic) DNS Client based on Docker Labels

DockDNS is a DNS updater, which supports configuring DNS records through Docker labels.
Currently DockDNS only supports Cloudflare as a DNS provider.

## Features

- Dynamic DNS updates
- Static DNS entries (e.g. with a static IP address)
- Static DNS record configuration based on a config file
- Dynamic DNS record configuration based on Docker labels
- IPv4 & IPv6 support
- CNAME support
- Supports multiple zones
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
    provider: cloudflare # Name of the provider. Currently only Cloudflare is supported
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
```

## Dynamic Domains

Domains can also be configured using Docker labels.
Supported labels:
| Label | Example |
|-----------------|-----------------------------|
| dockdns.name | dockdns.name=somedomain.com |
| dockdns.a | dockdns.a=127.0.0.1 |
| dockdns.aaaa | dockdns.aaaa=::1 |
| dockdns.cname | dockdns.cname=target.otherdomain.com |
| dockdns.ttl | dockdns.ttl=600 |
| dockdns.proxied | dockdns.proxied=false |
| dockdns.comment | dockdns.comment=Some comment |

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
docker run -v ./config.yaml:/app/config.yaml -v /var/run/docker.sock:/var/run/docker.sock:ro ghcr.io/tarow/dockdns:latest
```

### Docker Compose

```yaml
services:
  dockdns:
    image: ghcr.io/tarow/dockdns:latest
    restart: unless-stopped
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
