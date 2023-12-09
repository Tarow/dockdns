package dns

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
)

func (h handler) updateRecords(domains []config.DomainRecord, publicIp4, publicIp6 string) {
	for _, domain := range domains {
		if strings.TrimSpace(domain.IP4) != "" && h.dnsCfg.EnableIP4 {
			h.updateRecord(domain, TypeA)
		}

		if strings.TrimSpace(domain.IP6) != "" && h.dnsCfg.EnableIP6 {
			h.updateRecord(domain, TypeAAAA)
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
	proxied, _ := strconv.ParseBool(domain.Proxied)
	return Record{
		Name:    domain.Name,
		IP:      domain.GetIP(recordType),
		Type:    recordType,
		TTL:     dnsCfg.TTL,
		Proxied: &proxied,
	}
}
