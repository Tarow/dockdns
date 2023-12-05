package dns

import (
	"log/slog"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
)

func (h handler) purgeUnknownRecords(domains []config.DomainRecord) {
	existingRecords, err := h.provider.List()
	if err != nil {
		slog.Error("failed to fetch existing records, skipping purge", "error", err)
		return
	}

	for _, record := range existingRecords {
		if !containsRecord(domains, record) {
			if err := h.provider.Delete(record); err != nil {
				slog.Error("failed to purge record", "name", record.Name, "type", record.Type)
			} else {
				slog.Info("successfully purged unknown record", "name", record.Name, "type", record.Type)
			}
		}
	}
}

// Check if an entry with same domain and type exists
func containsRecord(domains []config.DomainRecord, toCheck Record) bool {
	for _, domain := range domains {
		if domain.Name == toCheck.Name {
			if strings.TrimSpace(domain.IP4) != "" && toCheck.Type == "A" {
				return true
			}
			if strings.TrimSpace(domain.IP6) != "" && toCheck.Type == "AAAA" {
				return true
			}
		}
	}
	return false
}
