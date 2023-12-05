package provider

import (
	"errors"
	"fmt"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/Tarow/dockdns/internal/provider/cloudflare"
)

type Provider interface {
	Get(domain, recordType string) (dns.Record, error)
	Create(record dns.Record) (dns.Record, error)
	Update(record dns.Record) (dns.Record, error)
	Delete(record dns.Record) error
}

const (
	Cloudflare = "cloudflare"
)

type ProviderCreator func(config.Provider) (Provider, error)

var providers = map[string]func(config.Provider) (Provider, error){
	Cloudflare: func(providerCfg config.Provider) (Provider, error) {
		return cloudflare.New(providerCfg.ApiToken, providerCfg.ZoneID)
	},
}

func Get(providerCfg config.Provider) (Provider, error) {
	if providerCfg.Name == "" {
		return nil, errors.New("no DNS provider specified")
	}

	providerCreator, exists := providers[providerCfg.Name]
	if !exists {
		return nil, fmt.Errorf("invalid provider: %s", providerCfg.Name)
	}

	return providerCreator(providerCfg)
}
