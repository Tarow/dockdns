package dns

import (
	"log/slog"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
)

func (h Handler) purgeUnknownRecords(provider Provider, domains []config.DomainRecord) {
	existingRecords, err := provider.List()
	if err != nil {
		slog.Error("failed to fetch existing records, skipping purge", "error", err)
		return
	}

	for _, record := range existingRecords {
		if !containsRecord(domains, record, h.DnsCfg) {
			if err := provider.Delete(record); err != nil {
				slog.Error("failed to purge record", "name", record.Name, "type", record.Type, "error", err)
			} else {
				slog.Info("successfully purged unknown record", "name", record.Name, "type", record.Type)
			}
		}
	}
}

// Check if an entry with same domain and type exists
func containsRecord(domains []config.DomainRecord, toCheck Record, dnsCfg config.DNS) bool {
	for _, domain := range domains {
		if domain.Name == toCheck.Name {
			// If a CNAME is configured, the A and AAAA settings will be considered unknown
			if strings.TrimSpace(domain.CName) != "" {
				if toCheck.Type == constants.RecordTypeCNAME {
					return true
				}
			} else {
				if dnsCfg.EnableIP4 && toCheck.Type == constants.RecordTypeA {
					return true
				}
				if dnsCfg.EnableIP6 && toCheck.Type == constants.RecordTypeAAAA {
					return true
				}
			}
		}
	}
	return false
}
