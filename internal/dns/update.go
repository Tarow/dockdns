package dns

import (
	"log/slog"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
)

func (h Handler) updateRecords(provider Provider, domains []config.DomainRecord) {
	for _, domain := range domains {
		// Important: If a CNAME is set, A and AAAA records for the same name cannot be set. They will be ignored!
		if strings.TrimSpace(domain.CName) != "" {
			h.updateRecord(provider, domain, constants.RecordTypeCNAME)
		} else {
			if strings.TrimSpace(domain.IP4) != "" && h.DnsCfg.EnableIP4 {
				h.updateRecord(provider, domain, constants.RecordTypeA)
			}

			if strings.TrimSpace(domain.IP6) != "" && h.DnsCfg.EnableIP6 {
				h.updateRecord(provider, domain, constants.RecordTypeAAAA)
			}
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

	newRecord := createRecord(domain, recordType)
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

func createRecord(domain config.DomainRecord, recordType string) Record {
	return Record{
		Name:    domain.Name,
		Content: domain.GetContent(recordType),
		Type:    recordType,
		TTL:     domain.TTL,
		Proxied: domain.Proxied,
		Comment: domain.Comment,
	}
}

func isEqual(record Record, domain config.DomainRecord, recordType string) bool {
	content := domain.GetContent(recordType)

	if record.Content != content {
		return false
	}

	if record.Name != domain.Name {
		return false
	}

	if record.Proxied != domain.Proxied {
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
