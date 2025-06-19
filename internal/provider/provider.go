package provider

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/Tarow/dockdns/internal/provider/cloudflare"
)

const (
	Cloudflare = "cloudflare"
)

type ProviderCreator func(config.Zone) (dns.Provider, error)

var providers = map[string]func(*config.Zone) (dns.Provider, error){
	Cloudflare: func(zoneCfg *config.Zone) (dns.Provider, error) {
		if zoneCfg.ZoneID == "" {
			slog.Debug("zone id not set. Trying to fetch it dynamically", "zone", zoneCfg.Name)
			zoneID, err := cloudflare.FetchZoneID(zoneCfg.ApiToken, zoneCfg.Name)
			if err != nil {
				return nil, fmt.Errorf("no zone id set for domain %s and could not fetch it: %w", zoneCfg.Name, err)
			}
			slog.Debug("Fetched zone id", "domain", zoneCfg.Name, "zoneID", zoneID)
			zoneCfg.ZoneID = zoneID
		}

		return cloudflare.New(zoneCfg.ApiToken, zoneCfg.ZoneID)
	},
}

func Get(zoneCfg *config.Zone, dryRun bool) (dns.Provider, error) {
	if zoneCfg.Provider == "" {
		return nil, errors.New("no DNS provider specified")
	}

	providerCreator, exists := providers[zoneCfg.Provider]
	if !exists {
		return nil, fmt.Errorf("invalid provider: %s", zoneCfg.Provider)
	}

	provider, err := providerCreator(zoneCfg)
	if dryRun {
		provider = NewDryRunProvider(provider)
	}
	return provider, err
}
