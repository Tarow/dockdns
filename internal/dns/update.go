package dns

import (
	"log/slog"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
)

func (h handler) updateRecords(domains []config.DomainRecord, publicIp4, publicIp6 string) {
	for _, domain := range domains {
		if strings.TrimSpace(domain.IP4) != "" && h.dnsCfg.EnableIP4 {
			h.updateRecord(domain, constants.RecordTypeA)
		}

		if strings.TrimSpace(domain.IP6) != "" && h.dnsCfg.EnableIP6 {
			h.updateRecord(domain, constants.RecordTypeAAAA)
		}
	}
}

func (h handler) updateRecord(domain config.DomainRecord, recordType string) {
	existingRecord, err := h.provider.Get(domain.Name, recordType)
	if err != nil {
		slog.Error("failed to fetch existing record", "name", domain.Name, "type", recordType, "action", "skip record")
		return
	}
	if existingRecord.IP == domain.GetIP(recordType) {
		slog.Debug("No IP change detected, skipping update", "name", domain.Name, "type", recordType)
		return
	}

	newRecord := createRecord(domain, h.dnsCfg, recordType)
	var updatedRecord Record
	if existingRecord.ID == "" {
		updatedRecord, err = h.provider.Create(newRecord)
		if err != nil {
			slog.Error("failed to create record", "record", newRecord, "error", err)
			return
		}
		slog.Info("Successfully created new record", "name", updatedRecord.Name, "ip", updatedRecord.IP, "type", updatedRecord.Type)
	} else {
		newRecord.ID = existingRecord.ID
		updatedRecord, err = h.provider.Update(newRecord)
		if err != nil {
			slog.Error("failed to update record", "record", newRecord, "error", err)
			return
		}
		slog.Info("Successfully updated record", "name", updatedRecord.Name, "ip", updatedRecord.IP, "type", updatedRecord.Type)
	}

}

func createRecord(domain config.DomainRecord, dnsCfg config.DNS, recordType string) Record {
	return Record{
		Name:    domain.Name,
		IP:      domain.GetIP(recordType),
		Type:    recordType,
		TTL:     dnsCfg.TTL,
		Proxied: domain.Proxied,
	}
}
