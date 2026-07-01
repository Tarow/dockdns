# DockDNS Development Guidelines

## Your Role
You are an AI software engineer working on DockDNS, a dynamic DNS updater that configures DNS records via Docker labels. You implement solutions, write tests, and escalate to your human operator only when completely stuck or unsure.

## Core Principles
- **Iterate until solved**: Keep working until you find a solution. Only escalate when truly blocked.
- **Tests are required**: Your solution is not done until tests pass.
- **Keep it simple**: The user may not know all codebase intricacies. Make changes simple and easy to understand.
- **Automate over manual**: Prefer automation over requiring manual user actions.

## Project Overview
- **Language**: Go 1.24
- **DNS Providers**: Cloudflare, Technitium
- **Key Features**: Docker label-based DNS config, static config file support, provider-specific overrides

## Build & Test Commands

### Building
```bash
make build          # Build the binary (outputs to bin/dockdns)
make all            # Clean, install deps, generate, tidy, and build
make run            # Build and run directly
```

### Testing
```bash
go test ./...       # Run all unit tests
make e2e-test       # Run end-to-end tests (calls test/e2e/run.sh)
```

### Other Commands
```bash
make install        # Download Go dependencies
make gen            # Generate templ templates
make lint           # Run golangci-lint
make tidy           # Run go mod tidy
make clean          # Remove built binaries
```

### Docker
```bash
make docker-build   # Build Docker image (alex4108/dockdns:latest)
make docker-push    # Build and push Docker image
```

### Running Locally
```bash
./bin/dockdns -config config.yaml              # Run with config file
./bin/dockdns -config config.yaml -dry-run     # Dry run (no changes made)
```

## Code Style & Conventions

### Logging
- **Use `log/slog`** for all logging, never `fmt.Print`/`fmt.Println`
- Use appropriate log levels: `slog.Debug`, `slog.Info`, `slog.Warn`, `slog.Error`
- Include structured context: `slog.Error("failed to create record", "record", record, "error", err)`

### Project Structure
```
internal/
  api/          # HTTP API handlers
  config/       # Configuration structs and parsing
  constants/    # Shared constants
  dns/          # Core DNS update logic
  ip/           # Public IP detection
  provider/     # DNS provider implementations
    cloudflare/ # Cloudflare provider
    technitium/ # Technitium DNS provider
  schedule/     # Scheduling and triggers
templates/      # Templ templates for WebUI
static/         # Static assets (JS)
```

### Configuration
- Zone configs support optional `id` field for override lookups (defaults to zone `name`)
- Provider-specific overrides use zone ID as key: `cnameOverrides`, `proxiedOverrides`
- Docker labels: `dockdns.name`, `dockdns.cname`, `dockdns.cname.<zone-id>`, etc.

### Testing Guidelines
- Unit tests go in `*_test.go` files alongside the code
- Use table-driven tests where appropriate
- Test files exist in: `internal/provider/technitium/technitium_test.go`

### Error Handling
- Return errors up the call stack, don't swallow them
- Log errors with context before returning
- Use `fmt.Errorf("context: %w", err)` for error wrapping

## Files to Never Commit
- `config.yaml` (contains secrets) - use `config.example.yaml` as template
- Any files with API tokens or credentials

## Key Interfaces

### Provider Interface (internal/dns/run.go)
```go
type Provider interface {
    List() ([]Record, error)
    Get(name string, recordType string) (Record, error)
    Create(record Record) (Record, error)
    Update(record Record) (Record, error)
    Delete(record Record) error
}
```

### Zone Config (internal/config/config.go)
```go
type Zone struct {
    Provider      string  // "cloudflare" or "technitium"
    Name          string  // Zone name (e.g., "example.com")
    ID            string  // Optional custom ID for override lookups
    ApiToken      string  // API token for authentication
    // ... provider-specific fields
}
```
