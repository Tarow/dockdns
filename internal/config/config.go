package config

import (
	"os"
	"regexp"
	"strings"

	"github.com/Tarow/dockdns/internal/constants"
)

type AppConfig struct {
	Interval        int       `yaml:"interval" env-default:"600"`
	DebounceTime    int       `yaml:"debounceTime" env-default:"10"`
	MaxDebounceTime int       `yaml:"maxDebounceTime" env-default:"600"`
	WebUI           bool      `yaml:"webUI" env-default:"false"`
	Log             LogConfig `yaml:"log"`
	Zones           Zones     `yaml:"zones"`
	DNS             DNS       `yaml:"dns"`
	Domains         Domains   `yaml:"domains"`
}

func (c *AppConfig) EnrichZoneSecretsFromEnv() {
	sanitizeRegexp := regexp.MustCompile(`[^a-zA-Z0-9]`)

	for i, zone := range c.Zones {
		envZoneName := strings.ToUpper(sanitizeRegexp.ReplaceAllString(zone.Name, "_"))

		if zone.ApiToken == "" {
			e := envZoneName + "_API_TOKEN"
			if val, ok := os.LookupEnv(e); ok {
				c.Zones[i].ApiToken = val
			}
		}
		if zone.ZoneID == "" {
			e := envZoneName + "_ZONE_ID"
			if val, ok := os.LookupEnv(e); ok {
				c.Zones[i].ZoneID = val
			}
		}
	}
}

type LogFormat string

const LogFormatSimple = "simple"
const LogFormatJson = "json"

type LogConfig struct {
	Level  string    `yaml:"level" env-default:"info"`
	Format LogFormat `yaml:"format" env-default:"simple"`
}

type Zones []Zone
type Zone struct {
	// Shared
	Provider string `yaml:"provider"`

	// For all providers, the domain name / zone name
	Name string `yaml:"name"`

	// Optional user-defined identifier for this zone configuration.
	// Used as the key for provider-specific overrides (e.g., dockdns.cname.<id>).
	// If not set, defaults to the zone Name for backwards compatibility.
	ID string `yaml:"id,omitempty"`

	// For cloudflare and technitium, the API token
	ApiToken string `yaml:"apiToken"`

	// Cloudflare specific
	ZoneID string `yaml:"zoneID,omitempty"`

	// Technitium specific
	ApiURL        string `yaml:"apiURL,omitempty"`        // For technitium, the API URL (e.g., http://localhost:5380)
	ApiUsername   string `yaml:"apiUsername,omitempty"`   // For technitium, the username for authentication
	ApiPassword   string `yaml:"apiPassword,omitempty"`   // For technitium, the password for authentication
	SkipTLSVerify bool   `yaml:"skipTLSVerify,omitempty"` // For technitium, skip TLS certificate verification
}

type DNS struct {
	EnableIP4    bool `yaml:"a"`
	EnableIP6    bool `yaml:"aaaa"`
	DefaultTTL   int  `yaml:"defaultTTL" env-default:"300"`
	PurgeUnknown bool `yaml:"purgeUnknown" env-default:"false"`
}

type Domains []DomainRecord
type DomainRecord struct {
	Name    string `yaml:"name" label:"dockdns.name"`
	IP4     string `yaml:"a" label:"dockdns.a"`
	IP6     string `yaml:"aaaa" label:"dockdns.aaaa"`
	CName   string `yaml:"cname" label:"dockdns.cname"`
	TTL     int    `yaml:"ttl" label:"dockdns.ttl"`
	Proxied bool   `yaml:"proxied" label:"dockdns.proxied"`
	Comment string `yaml:"comment" label:"dockdns.comment"`

	// Provider-specific overrides (zone ID -> override value)
	// These allow different values per DNS provider/zone.
	// The key should be the zone's ID (if set) or Name (for backwards compatibility).
	CNameOverrides   map[string]string `yaml:"cnameOverrides,omitempty"`   // e.g., {"technitium-internal": "internal.example.com"}
	ProxiedOverrides map[string]bool   `yaml:"proxiedOverrides,omitempty"` // e.g., {"cloudflare-prod": true}

	// Container metadata (populated for Docker-sourced records)
	ContainerID   string `yaml:"-"` // Docker container ID (short form)
	ContainerName string `yaml:"-"` // Docker container name
	Source        string `yaml:"-"` // Source of the record: "docker" or "static"
}

// GetContentForZone returns the content for the given record type, with zone-specific overrides for CNAME.
// The zoneID parameter should be the zone's ID (if set) or Name (for backwards compatibility).
func (d DomainRecord) GetContentForZone(recordType string, zoneID string) string {
	switch recordType {
	case constants.RecordTypeA:
		return d.IP4
	case constants.RecordTypeAAAA:
		return d.IP6
	case constants.RecordTypeCNAME:
		// Check for zone-specific CNAME override
		if d.CNameOverrides != nil {
			if override, exists := d.CNameOverrides[zoneID]; exists && override != "" {
				return override
			}
		}
		return d.CName
	default:
		return ""
	}
}

// GetProxiedForZone returns the proxied setting, with zone-specific override if available.
// The zoneID parameter should be the zone's ID (if set) or Name (for backwards compatibility).
func (d DomainRecord) GetProxiedForZone(zoneID string) bool {
	if d.ProxiedOverrides != nil {
		if override, exists := d.ProxiedOverrides[zoneID]; exists {
			return override
		}
	}
	return d.Proxied
}

// GetKey returns the zone's key for use in override lookups.
// Returns the ID if set, otherwise returns the Name for backwards compatibility.
func (z Zone) GetKey() string {
	if z.ID != "" {
		return z.ID
	}
	return z.Name
}

// GetContent returns the content for the given record type (backwards compatible).
func (d DomainRecord) GetContent(recordType string) string {
	return d.GetContentForZone(recordType, "")
}
