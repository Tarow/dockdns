package provider

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/Tarow/dockdns/internal/provider/cloudflare"
	"github.com/Tarow/dockdns/internal/provider/rfc2136"
)

const (
	Cloudflare = "cloudflare"
	Rfc2136    = "rfc2136"
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
	Rfc2136: func(zoneCfg *config.Zone) (dns.Provider, error) {
		if zoneCfg.ApiHost == "" ||
			zoneCfg.ApiPort == "" ||
			zoneCfg.TsigName == "" ||
			zoneCfg.ApiToken == "" ||
			zoneCfg.TsigAlgo == "" ||
			zoneCfg.Name == "" {
			return nil, fmt.Errorf("RFC2136 provider requires ApiHost, ApiPort, TsigName, ApiToken (TsigSecret), TsigAlgo, Name (zone) to be set.  Got zoneCfg: %v", zoneCfg)
		}
		// Default to UDP if protocol not specified
		protocol := zoneCfg.Protocol
		if protocol == "" {
			protocol = "udp"
		}
		return rfc2136.New(
			zoneCfg.ApiHost,
			zoneCfg.ApiPort,
			protocol,
			zoneCfg.TsigName,
			zoneCfg.ApiToken,
			zoneCfg.TsigAlgo,
			zoneCfg.Name), nil
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
