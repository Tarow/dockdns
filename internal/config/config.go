package config

import "github.com/Tarow/dockdns/internal/constants"

type AppConfig struct {
	Interval uint      `yaml:"interval" env-default:"600"`
	Log      LogConfig `yaml:"log"`
	Provider Provider  `yaml:"provider"`
	DNS      DNS       `yaml:"dns"`
	Domains  Domains   `yaml:"domains"`
}

type LogFormat string

const LogFormatSimple = "simple"
const LogFormatJson = "json"

type LogConfig struct {
	Level  string    `yaml:"level" env-default:"info"`
	Format LogFormat `yaml:"format" env-default:"simple"`
}

type Provider struct {
	Name     string `yaml:"name"`
	ApiToken string `yaml:"apiToken"`
	ZoneID   string `yaml:"zoneID"`
}

type DNS struct {
	EnableIP4    bool `yaml:"a"`
	EnableIP6    bool `yaml:"aaaa"`
	DefaultTTL   int  `yaml:"defaultTTL" env-default:"600"`
	PurgeUnknown bool `yaml:"purgeUnknown" env-default:"false"`
}

type Domains []DomainRecord

type DomainRecord struct {
	Name    string `yaml:"name" label:"dockdns.name"`
	IP4     string `yaml:"a" label:"dockdns.a"`
	IP6     string `yaml:"aaaa" label:"dockdns.aaaa"`
	TTL     int    `yaml:"ttl" label:"dockdns.ttl"`
	Proxied bool   `yaml:"proxied" label:"dockdns.proxied"`
}

func (d DomainRecord) GetIP(recordType string) string {
	if recordType == constants.RecordTypeA {
		return d.IP4
	} else {
		return d.IP6
	}
}
