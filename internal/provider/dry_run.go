package provider

import (
	"fmt"
	"log/slog"

	"github.com/Tarow/dockdns/internal/dns"
)

type dryRunProvider struct {
	dns.Provider
}

func NewDryRunProvider(p dns.Provider) dryRunProvider {
	return dryRunProvider{
		Provider: p,
	}
}

func (drp dryRunProvider) Create(record dns.Record) (dns.Record, error) {
	logDryRunRecordAction("CREATE", record)
	return record, nil
}

func (drp dryRunProvider) Update(record dns.Record) (dns.Record, error) {
	logDryRunRecordAction("UPDATE", record)
	return record, nil
}

func (drp dryRunProvider) Delete(record dns.Record) error {
	logDryRunRecordAction("DELETE", record)
	return nil
}

func logDryRunRecordAction(msg string, record dns.Record) {
	slog.Info(fmt.Sprintf("DRY RUN %v", msg),
		slog.String("ID", record.ID),
		slog.String("content", record.Content),
		slog.String("name", record.Name),
		slog.String("type", record.Type),
		slog.Int("TTL", record.TTL),
		slog.Bool("proxied", record.Proxied),
	)
}
