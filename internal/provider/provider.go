package provider

import (
	"errors"
	"fmt"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/Tarow/dockdns/internal/provider/cloudflare"
)

const (
	Cloudflare = "cloudflare"
)

type ProviderCreator func(config.Provider) (dns.Provider, error)

var providers = map[string]func(config.Provider) (dns.Provider, error){
	Cloudflare: func(providerCfg config.Provider) (dns.Provider, error) {
		return cloudflare.New(providerCfg.ApiToken, providerCfg.ZoneID)
	},
}

func Get(providerCfg config.Provider) (dns.Provider, error) {
	if providerCfg.Name == "" {
		return nil, errors.New("no DNS provider specified")
	}

	providerCreator, exists := providers[providerCfg.Name]
	if !exists {
		return nil, fmt.Errorf("invalid provider: %s", providerCfg.Name)
	}

	return providerCreator(providerCfg)
}
