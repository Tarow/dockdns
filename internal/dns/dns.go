package dns

import (
	"log/slog"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/ip"
	"github.com/docker/docker/client"
)

type handler struct {
	provider      Provider
	dnsDefaultCfg config.DNS
	staticDomains config.Domains
	dockerCli     *client.Client
}

type Provider interface {
	List() ([]Record, error)
	Get(name, recordType string) (Record, error)
	Create(record Record) (Record, error)
	Update(record Record) (Record, error)
	Delete(record Record) error
}

type Record struct {
	ID      string
	Name    string
	IP      string
	Type    string
	Proxied *bool
	TTL     uint
}

func NewHandler(provider Provider, dnsDefaultCfg config.DNS,
	staticDomains config.Domains, dockerCli *client.Client) handler {
	return handler{
		provider:      provider,
		dnsDefaultCfg: dnsDefaultCfg,
		staticDomains: staticDomains,
		dockerCli:     dockerCli,
	}
}

func (h handler) Run() error {
	slog.Debug("starting dns update job")
	staticDomains := h.staticDomains
	slog.Debug("static config", "domains", staticDomains)

	dockerDomains, err := h.filterDockerLabels()
	if err != nil {
		slog.Error("Could not fetch domains from docker labels, ignoring label configuration", "error", err)
	} else {
		slog.Debug("dynamic docker config", "domains", dockerDomains)
	}

	allDomains := removeDuplicates(staticDomains, dockerDomains)
	slog.Debug("removed duplicates", "domains", allDomains)

	if len(allDomains) > 0 {
		publicIp4, err := ip.GetPublicIP4Address()
		if err != nil {
			return err
		} else {
			slog.Debug("got public ipv4 address", "ip", publicIp4)
		}
		publicIp6, err := ip.GetPublicIP6Address()
		if err != nil {
			return err
		} else {
			slog.Debug("got public ipv6 address", "ip", publicIp6)
		}

		h.setIPs(allDomains, publicIp4, publicIp6)
		slog.Debug("set missing ips", "domains", allDomains)

		h.updateRecords(allDomains, publicIp4, publicIp6)
	} else {
		slog.Info("Found no records to update")
	}

	if h.dnsDefaultCfg.PurgeUnknown {
		h.purgeUnknownRecords(allDomains)
	}

	return nil
}

func (h handler) setIPs(domains []config.DomainRecord, publicIp4, publicIp6 string) {
	for i, domain := range domains {
		if strings.TrimSpace(domain.IP4) == "" && h.dnsDefaultCfg.EnableIP4 {
			domain.IP4 = publicIp4
		}
		if strings.TrimSpace(domain.IP6) == "" && h.dnsDefaultCfg.EnableIP6 {
			domain.IP6 = publicIp6
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

func containsDomain(domains []config.DomainRecord, domainName string) bool {
	for _, domain := range domains {
		if domain.Name == domainName {

			return true
		}
	}
	return false
}
