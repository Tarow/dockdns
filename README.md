![Build](https://github.com/tarow/dockdns/actions/workflows/ci.yaml/badge.svg)
[![go report](https://goreportcard.com/badge/github.com/Tarow/dockdns)](https://goreportcard.com/report/github.com/Tarow/dockdns)
[![Renovate](https://img.shields.io/badge/renovate-enabled-brightgreen.svg)](https://renovatebot.com)

# DockDNS - (Dynamic) DNS Client based on Docker Labels

DockDNS is a DNS updater, which supports configuring DNS records through Docker labels.
DockDNS supports Cloudflare and RFC2136-compliant DNS servers as providers.

## Features

- Dynamic DNS updates
- Static DNS entries (e.g. with a static IP address)
- Static DNS record configuration based on a config file
- Dynamic DNS record configuration based on Docker labels
- IPv4 & IPv6 support
- CNAME support
- Supports multiple zones
- RFC2136 provider: supports querying (List/Get) and updating TXT records
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
  provider: cloudflare # Name of the provider. Supported: cloudflare, rfc2136
## RFC2136 Provider

DockDNS supports any DNS server that implements RFC2136 Dynamic Updates. This allows integration with BIND, Knot, PowerDNS, and other standards-compliant DNS servers.

Example zone configuration:

```yaml
zones:
  - name: somedomain.com
    provider: rfc2136
    apiHost: dns.somedomain.com
    apiPort: 53
    apiToken: superSecret # TSIG Secret
    tsigName: keyName # TSIG Key Name
    tsigAlgo: HMAC-SHA256 # TSIG Key Algorithm
```

### Supported Operations
- Create, Update, Delete TXT records (for DNS-01 challenges and dynamic TXT entries)
- List all TXT records in a zone
- Get a specific TXT record by name

> Note: Listing and querying records uses DNS queries and does not require API access. Only TXT records are supported for RFC2136 operations.
    apiToken: ... # API Token, needs permission 'Zone.Zone' (read) and Zone.DNS (edit). Can also be passed as environment variable: SOMEDOMAIN_COM_API_TOKEN
    zoneID: ... # Optional: If not set, will be fetched dynamically. ZoneID of this zone. Can also be passed as environment variable: SOMEDOMAIN_COM_ZONE_ID
  - name: somedomain.com # Root name of the zone
    provider: rfc2136 # Any DNS server that complies with RFC2136 Dynamic Updates
    apiHost: dns.somedomain.com # Host of DNS server, eg mydnsserver.mydomain.com
    apiPort: 53 # The tcp port of the DNS server
    apiToken: superSecret # TSIG Secret
    tsigName: keyName # TSIG Key Name
    tsigAlgo: HMAC-SHA256 # TSIG Key Algorithm
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

## Development

You need:

* Go 1.23
* Templ