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

type ProviderCreator func(config.Zone) (dns.Provider, error)

var providers = map[string]func(config.Zone) (dns.Provider, error){
	Cloudflare: func(zoneCfg config.Zone) (dns.Provider, error) {
		return cloudflare.New(zoneCfg.ApiToken, zoneCfg.ZoneID)
	},
}

func Get(zoneCfg config.Zone) (dns.Provider, error) {
	if zoneCfg.Provider == "" {
		return nil, errors.New("no DNS provider specified")
	}

	providerCreator, exists := providers[zoneCfg.Provider]
	if !exists {
		return nil, fmt.Errorf("invalid provider: %s", zoneCfg.Provider)
	}

	return providerCreator(zoneCfg)
}
