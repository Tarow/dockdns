package dns

import (
	"log/slog"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/ip"
	"github.com/docker/docker/client"
)

type handler struct {
	providers     map[string]Provider
	dnsCfg        config.DNS
	staticDomains config.Domains
	dockerCli     *client.Client
}

type Provider interface {
	List() ([]Record, error)
	Get(name string, recordType string) (Record, error)
	Create(record Record) (Record, error)
	Update(record Record) (Record, error)
	Delete(record Record) error
}

type Record struct {
	ID      string
	Name    string
	IP      string
	Type    string
	Proxied bool
	TTL     int
}

func NewHandler(providers map[string]Provider, dnsDefaultCfg config.DNS,
	staticDomains config.Domains, dockerCli *client.Client) handler {
	return handler{
		providers:     providers,
		dnsCfg:        dnsDefaultCfg,
		staticDomains: staticDomains,
		dockerCli:     dockerCli,
	}
}

func (h handler) Run() error {
	slog.Debug("starting dns update job")
	staticDomains := h.staticDomains
	slog.Debug("static config", "domains", staticDomains)

	var dockerDomains []config.DomainRecord
	var err error
	if h.dockerCli != nil {
		dockerDomains, err = h.filterDockerLabels()
	}
	if err != nil {
		slog.Error("could not fetch domains from docker labels, ignoring label configuration", "error", err)
	} else {
		slog.Debug("dynamic docker config", "domains", dockerDomains)
	}

	allDomains := removeDuplicates(staticDomains, dockerDomains)
	slog.Debug("removed duplicates", "deduped", allDomains)

	if len(allDomains) > 0 {
		var publicIp4, publicIp6 string
		var err error

		if h.dnsCfg.EnableIP4 {
			publicIp4, err = ip.GetPublicIP4Address()
			if err != nil {
				slog.Warn("could not fetch public IPv4 address, only static entries will be set", "error", err)
			} else {
				slog.Debug("got public IPv4 address", "ip", publicIp4)
			}
		}

		if h.dnsCfg.EnableIP6 {
			publicIp6, err = ip.GetPublicIP6Address()
			if err != nil {
				slog.Warn("could not fetch public IPv6 address, only static entries will be set", "error", err)
			} else {
				slog.Debug("got public IPv6 address", "ip", publicIp4)
			}
		}

		h.setIPs(allDomains, publicIp4, publicIp6)
		slog.Debug("set missing IPs", "domains", allDomains)

		h.applyDefaults(allDomains)
		slog.Debug("applied default values", "domains", allDomains)

		if err != nil {
			slog.Error("could not group domains by zone", "error", err)
			return err
		}
	} else {
		slog.Info("Found no records to update")
	}

	for zone, provider := range h.providers {
		domains := filterDomains(allDomains, zone)

		slog.Debug("starting update", "zone", zone, "domains", domains)
		h.updateRecords(provider, domains)
		slog.Debug("finished update", "zone", zone, "domains", domains)

		if h.dnsCfg.PurgeUnknown {
			slog.Debug("starting purge of unknown domains", "zone", zone, "domains", domains)
			h.purgeUnknownRecords(provider, domains)
			slog.Debug("finished purge of unknown domains", "zone", zone, "domains", domains)
		}
	}

	return nil
}

func (h handler) setIPs(domains []config.DomainRecord, publicIp4, publicIp6 string) {
	for i, domain := range domains {
		if strings.TrimSpace(domain.IP4) == "" && h.dnsCfg.EnableIP4 {
			domain.IP4 = publicIp4
		}
		if strings.TrimSpace(domain.IP6) == "" && h.dnsCfg.EnableIP6 {
			domain.IP6 = publicIp6
		}
		domains[i] = domain
	}
}

func (h handler) applyDefaults(domains []config.DomainRecord) {
	for i, domain := range domains {
		if domain.TTL == 0 {
			domain.TTL = h.dnsCfg.DefaultTTL
		}
		domains[i] = domain
	}
}

func removeDuplicates(staticDomains, dockerDomains []config.DomainRecord) []config.DomainRecord {
	result := staticDomains

	for _, dockerDomain := range dockerDomains {
		if containsDomain(staticDomains, dockerDomain.Name) {
			slog.Info("Found duplicate domain config, using static configuration", "subdomain", dockerDomain.Name)
		} else {
			result = append(result, dockerDomain)
		}
	}
	return result
}

func filterDomains(allDomains config.Domains, zoneName string) config.Domains {
	var result config.Domains

	for _, domain := range allDomains {
		if strings.HasSuffix(domain.Name, zoneName) {
			result = append(result, domain)
		}
	}

	return result
}

func containsDomain(domains []config.DomainRecord, domainName string) bool {
	for _, domain := range domains {
		if domain.Name == domainName {

			return true
		}
	}
	return false
}
