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
	Provider string `yaml:"provider"`
	Name     string `yaml:"name"`
	ApiToken string `yaml:"apiToken"`
	ZoneID   string `yaml:"zoneID"`
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
}

func (d DomainRecord) GetContent(recordType string) string {
	switch recordType {
	case constants.RecordTypeA:
		return d.IP4
	case constants.RecordTypeAAAA:
		return d.IP6
	case constants.RecordTypeCNAME:
		return d.CName
	default:
		return ""
	}
}
