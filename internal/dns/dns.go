package dns

import (
	"log/slog"
	"strconv"
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
	Get(domain, recordType string) (Record, error)
	Create(record Record) (Record, error)
	Update(record Record) (Record, error)
	Delete(record Record) error
}

type Record struct {
	ID      string
	Domain  string
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
	staticDomains := h.staticDomains

	dockerDomains, err := h.filterDockerLabels()
	if err != nil {
		slog.Error("Could not fetch domains from docker labels, ignoring label configuration", "error", err)
	}

	allDomains := append(staticDomains, dockerDomains...)
	slog.Debug("Extracted all domains", "domains", allDomains)

	if len(allDomains) > 0 {
		publicIp4, err := ip.GetPublicIP4Address()
		if err != nil {
			return err
		}
		publicIp6, err := ip.GetPublicIP6Address()
		if err != nil {
			return err
		}

		h.updateRecords(allDomains, publicIp4, publicIp6)
	} else {
		slog.Info("Found no records to update")
	}

	return nil
}

func (h handler) updateRecords(domains []config.DomainRecord, publicIp4, publicIp6 string) {

	for _, domain := range domains {
		// IPv4 record has to be updated
		if strings.TrimSpace(domain.IP4) != "" || h.dnsDefaultCfg.EnableIP4 {
			existingRecord, err := h.provider.Get(domain.Name, "A")
			if err != nil {
				slog.Error("failed to fetch existing record", "domain", domain.Name, "type", "A", "action", "skip record")
				continue
			}
			newRecord := getIp4Record(domain, h.dnsDefaultCfg, publicIp4)
			var updatedRecord Record
			if existingRecord.ID == "" {
				updatedRecord, err = h.provider.Create(newRecord)
				if err != nil {
					slog.Error("failed to update record", "record", newRecord, "error", err)
					continue
				}
			} else {
				newRecord.ID = existingRecord.ID
				updatedRecord, err = h.provider.Update(newRecord)
				if err != nil {
					slog.Error("failed to update record", "record", newRecord, "error", err)
					continue
				}
			}
			slog.Info("Successfully updated record", "domain", updatedRecord.Domain, "ip", updatedRecord.IP, "type", updatedRecord.Type)
		}

		// IPv6 record has to be updated
		if strings.TrimSpace(domain.IP6) != "" || h.dnsDefaultCfg.EnableIP6 {
			existingRecord, err := h.provider.Get(domain.Name, "AAAA")
			if err != nil {
				slog.Error("failed to fetch existing record", "domain", domain.Name, "type", "AAAA", "action", "skip record")
				continue
			}
			newRecord := getIp6Record(domain, h.dnsDefaultCfg, publicIp6)
			var updatedRecord Record
			if existingRecord.ID == "" {
				updatedRecord, err = h.provider.Create(newRecord)
				if err != nil {
					slog.Error("failed to update record", "record", newRecord, "error", err)
					continue
				}
			} else {
				newRecord.ID = existingRecord.ID
				updatedRecord, err = h.provider.Update(newRecord)
				if err != nil {
					slog.Error("failed to update record", "record", newRecord, "error", err)
					continue
				}
			}
			slog.Info("Successfully updated record", "domain", updatedRecord.Domain, "ip", updatedRecord.IP, "type", updatedRecord.Type)
		}
	}
}

func getIp4Record(domain config.DomainRecord, dnsCfg config.DNS, publicIp4 string) Record {
	ip := domain.IP4
	if strings.TrimSpace(ip) == "" {
		ip = publicIp4
	}

	proxied, _ := strconv.ParseBool(domain.Proxied)
	return Record{
		Domain:  domain.Name,
		IP:      ip,
		Type:    "A",
		TTL:     dnsCfg.TTL,
		Proxied: &proxied,
	}
}

func getIp6Record(domain config.DomainRecord, dnsCfg config.DNS, publicIp6 string) Record {
	ip := domain.IP6
	if strings.TrimSpace(ip) == "" {
		ip = publicIp6
	}

	proxied, _ := strconv.ParseBool(domain.Proxied)
	return Record{
		Domain:  domain.Name,
		IP:      ip,
		Type:    "AAAA",
		TTL:     dnsCfg.TTL,
		Proxied: &proxied,
	}
}
