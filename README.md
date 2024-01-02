# DockDNS - (Dynamic) DNS Client based on Docker Labels

DockDNS is a DNS updater, which supports configuring DNS records through Docker labels.
Currently DockDNS only supports Cloudflare as a DNS provider.

## Features

- Dynamic DNS updates
- Static DNS entries (e.g. with a static IP address)
- Static DNS record configuration based on a config file
- Dynamic DNS record configuration based on Docker labels
- IPv4 & IPv6 support

## Configuration

The app configuration as well as the static domain entries are read from a configuration file (see [example configuration](config.example.yaml)).

```yaml
interval: 600 # Optional, the update interval in seconds. Defaults to 600

log:
  level: debug # Optional, Log level, one of 'debug', 'info', 'warn' or 'error'. Defaults to 'info'
  format: simple # Optional, output of the log format, 'simple' or 'json'. Defaults to 'simple'

zones: # Zone configuration (multiple zones can be provided)
  - name: somedomain.com # Root name of the zone
    provider: cloudflare # Name of the provider. Currently only Cloudflare is supported
    apiToken: ... # API Token, needs permission 'Zone.Zone' (read) and Zone.DNS (edit)
    zoneID: ... # ZoneID of this zone

dns:
  a: true # Update IPv4 addresses
  aaaa: false # Update IPv6 addresses
  defaultTTL: 300 # Optional, default TTL for all records. Defaults to 300
  purgeUnknown: true # Optional, delete unknown records. Defaults to false.

# Static domain configuration (optional)
domains:
  - name: "*.somedomain.com"

  - name: "somedomain.com"
    a: 10.0.0.2
    aaaa: ::1
```

## Dynamic Domains

Domains can also be configured using Docker labels.
Supported labels:
| Label | Example |
|-----------------|-----------------------------|
| dockdns.name | dockdns.name=somedomain.com |
| dockdns.a | dockdns.a=127.0.0.1 |
| dockdns.aaaa | dockdns.aaaa=::1 |
| dockdns.ttl | dockdns.ttl=600 |
| dockdns.proxied | dockdns.proxied=false |

---

If no explicit IP address is set, the public IP will be fetched and set automatically (DynDNS).

## Installation

### Go Binary

```
go install github.com/Tarow/dockdns@latest
```

Be default, DockDNS looks for a `config.yaml` in the current folder. The location of the configuration file can be overwritten by providing the `-config` flag:

```
dockdns -config /path/to/config.yaml
```

### Docker

```bash
docker run -v ./config.yaml:/app/config.yaml -v /var/run/docker.sock:/var/run/docker.sock:ro ghcr.io/tarow/dockdns:latest
```

### Docker Compose

```yaml
version: "3.7"

services:
  dockdns:
    image: ghcr.io/tarow/dockdns:latest
    restart: unless-stopped
    volumes:
      - ./config.yaml:/app/config.yaml
      - /var/run/docker.sock:/var/run/docker.sock:ro
```
