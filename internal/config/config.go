package config

type AppConfig struct {
	Interval uint     `yaml:"interval"`
	Provider Provider `yaml:"provider"`
	DNS      DNS      `yaml:"dns"`
	Domains  Domains  `yaml:"domains"`
}

type Provider struct {
	Name     string `yaml:"name"`
	ApiToken string `yaml:"apiToken"`
	ZoneID   string `yaml:"zoneID"`
}

type DNS struct {
	EnableIP4    bool `yaml:"a"`
	EnableIP6    bool `yaml:"aaaa"`
	TTL          uint `yaml:"ttl"`
	PurgeUnknown bool `yaml:"purgeUnknown"`
}

type Domains []DomainRecord

type DomainRecord struct {
	Name    string `yaml:"name" label:"dockdns.domain"`
	IP4     string `yaml:"a" label:"dockdns.a"`
	IP6     string `yaml:"aaaa" label:"dockdns.aaaa"`
	Proxied string `yaml:"dockdns.proxied"`
}
