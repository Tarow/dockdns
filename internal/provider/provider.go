package provider

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/Tarow/dockdns/internal/provider/cloudflare"
	"github.com/Tarow/dockdns/internal/provider/technitium"
)

const (
	Cloudflare = "cloudflare"
	Technitium = "technitium"
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
	Technitium: func(zoneCfg *config.Zone) (dns.Provider, error) {
		if zoneCfg.ApiURL == "" || zoneCfg.Name == "" {
			return nil, fmt.Errorf("Technitium provider requires ApiURL and Name (zone) to be set. Got zoneCfg: %v", zoneCfg)
		}
		// Either apiToken OR (username and password) must be provided
		if zoneCfg.ApiToken == "" && (zoneCfg.ApiUsername == "" || zoneCfg.ApiPassword == "") {
			return nil, fmt.Errorf("Technitium provider requires either ApiToken, or ApiUsername and ApiPassword to be set. Got zoneCfg: %v", zoneCfg)
		}
		return technitium.New(
			zoneCfg.ApiURL,
			zoneCfg.ApiUsername,
			zoneCfg.ApiPassword,
			zoneCfg.ApiToken,
			zoneCfg.Name,
			zoneCfg.SkipTLSVerify)
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
