package dns

import (
	"log/slog"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
)

func (h Handler) updateRecords(provider Provider, domains []config.DomainRecord, zoneKey string) {
	for _, domain := range domains {
		// Important: If a CNAME is set, A and AAAA records for the same name cannot be set. They will be ignored!
		// Check zone-specific CNAME override first, then fall back to default
		cname := domain.GetContentForZone(constants.RecordTypeCNAME, zoneKey)
		if strings.TrimSpace(cname) != "" {
			h.updateRecord(provider, domain, constants.RecordTypeCNAME, zoneKey)
		} else {
			if strings.TrimSpace(domain.IP4) != "" && h.DnsCfg.EnableIP4 {
				h.updateRecord(provider, domain, constants.RecordTypeA, zoneKey)
			}

			if strings.TrimSpace(domain.IP6) != "" && h.DnsCfg.EnableIP6 {
				h.updateRecord(provider, domain, constants.RecordTypeAAAA, zoneKey)
			}
		}
	}
}

func (h Handler) updateRecord(provider Provider, domain config.DomainRecord, recordType string, zoneKey string) {
	existingRecord, err := provider.Get(domain.Name, recordType)
	if err != nil {
		slog.Error("failed to fetch existing record", "name", domain.Name, "type", recordType, "action", "skip record", "error", err)
		return
	}
	if isEqual(existingRecord, domain, recordType, zoneKey) {
		slog.Debug("No change detected, skipping update", "name", domain.Name, "type", recordType)
		return
	}

	newRecord := createRecord(domain, recordType, zoneKey)
	var updatedRecord Record
	if existingRecord.ID == "" {
		updatedRecord, err = provider.Create(newRecord)
		if err != nil {
			slog.Error("failed to create record", "record", newRecord, "error", err)
			return
		}
		slog.Info("Successfully created new record", "name", updatedRecord.Name, "content", updatedRecord.Content, "type", updatedRecord.Type, "ttl", updatedRecord.TTL, "proxied", updatedRecord.Proxied, "comment", updatedRecord.Comment)
	} else {
		newRecord.ID = existingRecord.ID
		updatedRecord, err = provider.Update(newRecord)
		if err != nil {
			slog.Error("failed to update record", "record", newRecord, "error", err)
			return
		}
		slog.Info("Successfully updated record", "name", updatedRecord.Name, "content", updatedRecord.Content, "type", updatedRecord.Type, "ttl", updatedRecord.TTL, "proxied", updatedRecord.Proxied, "comment", updatedRecord.Comment)
	}

}

func createRecord(domain config.DomainRecord, recordType string, zoneKey string) Record {
	return Record{
		Name:          domain.Name,
		Content:       domain.GetContentForZone(recordType, zoneKey),
		Type:          recordType,
		TTL:           domain.TTL,
		Proxied:       domain.GetProxiedForZone(zoneKey),
		Comment:       domain.Comment,
		Source:        domain.Source,
		ContainerID:   domain.ContainerID,
		ContainerName: domain.ContainerName,
	}
}

func isEqual(record Record, domain config.DomainRecord, recordType string, zoneKey string) bool {
	content := domain.GetContentForZone(recordType, zoneKey)

	if record.Content != content {
		return false
	}

	if record.Name != domain.Name {
		return false
	}

	proxied := domain.GetProxiedForZone(zoneKey)
	if record.Proxied != proxied {
		return false
	}

	if record.Comment != domain.Comment {
		return false
	}

	// If domain is proxied, TTL will be auto, dont compare it
	if (!record.Proxied) && record.TTL != domain.TTL {
		return false
	}

	return true
}
