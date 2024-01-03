package dns

import (
	"log/slog"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
)

func (h Handler) updateRecords(provider Provider, domains []config.DomainRecord) {
	for _, domain := range domains {
		if strings.TrimSpace(domain.IP4) != "" && h.DnsCfg.EnableIP4 {
			h.updateRecord(provider, domain, constants.RecordTypeA)
		}

		if strings.TrimSpace(domain.IP6) != "" && h.DnsCfg.EnableIP6 {
			h.updateRecord(provider, domain, constants.RecordTypeAAAA)
		}
	}
}

func (h Handler) updateRecord(provider Provider, domain config.DomainRecord, recordType string) {
	existingRecord, err := provider.Get(domain.Name, recordType)
	if err != nil {
		slog.Error("failed to fetch existing record", "name", domain.Name, "type", recordType, "action", "skip record")
		return
	}
	if isEqual(existingRecord, domain, recordType) {
		slog.Debug("No change detected, skipping update", "name", domain.Name, "type", recordType)
		return
	}

	newRecord := createRecord(domain, h.DnsCfg, recordType)
	var updatedRecord Record
	if existingRecord.ID == "" {
		updatedRecord, err = provider.Create(newRecord)
		if err != nil {
			slog.Error("failed to create record", "record", newRecord, "error", err)
			return
		}
		slog.Info("Successfully created new record", "name", updatedRecord.Name, "ip", updatedRecord.IP, "type", updatedRecord.Type, "ttl", updatedRecord.TTL, "proxied", updatedRecord.Proxied)
	} else {
		newRecord.ID = existingRecord.ID
		updatedRecord, err = provider.Update(newRecord)
		if err != nil {
			slog.Error("failed to update record", "record", newRecord, "error", err)
			return
		}
		slog.Info("Successfully updated record", "name", updatedRecord.Name, "ip", updatedRecord.IP, "type", updatedRecord.Type, "ttl", updatedRecord.TTL, "proxied", updatedRecord.Proxied)
	}

}

func createRecord(domain config.DomainRecord, dnsCfg config.DNS, recordType string) Record {
	return Record{
		Name:    domain.Name,
		IP:      domain.GetIP(recordType),
		Type:    recordType,
		TTL:     domain.TTL,
		Proxied: domain.Proxied,
	}
}

func isEqual(record Record, domain config.DomainRecord, recordType string) bool {
	ip := domain.GetIP(recordType)

	if record.IP != ip {
		return false
	}

	if record.Name != domain.Name {
		return false
	}

	if record.Proxied != domain.Proxied {
		return false
	}

	// If domain is proxied, TTL will be auto, dont compare it
	if (!record.Proxied) && record.TTL != domain.TTL {
		return false
	}

	return true
}
