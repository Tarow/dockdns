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

// DomainRecordBase holds the DNS record fields that can be defined at the
// top level of a domain entry and, identically, inside a per-zone override.
// Reusing the same struct for base values and overrides keeps the two in
// sync: any field added here is automatically overridable per zone.
type DomainRecordBase struct {
	Name    string `yaml:"name" label:"dockdns.name"`
	IP4     string `yaml:"a" label:"dockdns.a"`
	IP6     string `yaml:"aaaa" label:"dockdns.aaaa"`
	CName   string `yaml:"cname" label:"dockdns.cname"`
	TTL     int    `yaml:"ttl" label:"dockdns.ttl"`
	Proxied bool   `yaml:"proxied" label:"dockdns.proxied"`
	Comment string `yaml:"comment" label:"dockdns.comment"`
}

type DomainRecord struct {
	DomainRecordBase `yaml:",inline"`

	// Overrides holds provider/zone-specific field overrides, keyed by the
	// zone's ID (if set) or Name (for backwards compatibility). Each override
	// reuses the base record shape; only the fields you set take effect, all
	// others inherit the base values.
	//
	// Config example:
	//   overrides:
	//     technitium-internal:
	//       cname: internal-target.local
	//       comment: Internal server
	//
	// Label example: dockdns.technitium-internal.cname=internal-target.local
	Overrides map[string]DomainRecordBase `yaml:"overrides,omitempty"`

	// Container metadata (populated for Docker-sourced records)
	ContainerID   string `yaml:"-"` // Docker container ID (short form)
	ContainerName string `yaml:"-"` // Docker container name
	Source        string `yaml:"-"` // Source of the record: "docker" or "static"
}

// override returns the override block for the given zone key, if one exists.
func (d DomainRecord) override(zoneID string) (DomainRecordBase, bool) {
	if d.Overrides == nil || zoneID == "" {
		return DomainRecordBase{}, false
	}
	o, ok := d.Overrides[zoneID]
	return o, ok
}

// GetContentForZone returns the content for the given record type, with zone-specific overrides.
// The zoneID parameter should be the zone's ID (if set) or Name (for backwards compatibility).
func (d DomainRecord) GetContentForZone(recordType string, zoneID string) string {
	o, hasOverride := d.override(zoneID)
	switch recordType {
	case constants.RecordTypeA:
		// Empty string check ensures invalid empty IPs are not used
		if hasOverride && o.IP4 != "" {
			return o.IP4
		}
		return d.IP4
	case constants.RecordTypeAAAA:
		if hasOverride && o.IP6 != "" {
			return o.IP6
		}
		return d.IP6
	case constants.RecordTypeCNAME:
		if hasOverride && o.CName != "" {
			return o.CName
		}
		return d.CName
	default:
		return ""
	}
}

// GetProxiedForZone returns the proxied setting, with zone-specific override if available.
// The zoneID parameter should be the zone's ID (if set) or Name (for backwards compatibility).
func (d DomainRecord) GetProxiedForZone(zoneID string) bool {
	if o, ok := d.override(zoneID); ok {
		return o.Proxied
	}
	return d.Proxied
}

// GetTTLForZone returns the TTL setting, with zone-specific override if available.
// The zoneID parameter should be the zone's ID (if set) or Name (for backwards compatibility).
// Note: a non-zero override TTL takes effect; zero inherits the base value.
func (d DomainRecord) GetTTLForZone(zoneID string) int {
	if o, ok := d.override(zoneID); ok && o.TTL != 0 {
		return o.TTL
	}
	return d.TTL
}

// GetCommentForZone returns the comment, with zone-specific override if available.
// The zoneID parameter should be the zone's ID (if set) or Name (for backwards compatibility).
// Note: a non-empty override comment takes effect; empty inherits the base value.
func (d DomainRecord) GetCommentForZone(zoneID string) string {
	if o, ok := d.override(zoneID); ok && o.Comment != "" {
		return o.Comment
	}
	return d.Comment
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
