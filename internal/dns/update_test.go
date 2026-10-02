package dns

import (
	"testing"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
)

type testProvider struct {
	records []Record
	created []Record
	updated []Record
	deleted []Record
}

func (p *testProvider) List() ([]Record, error) {
	return append([]Record(nil), p.records...), nil
}

func (p *testProvider) Get(name string, recordType string) (Record, error) {
	for _, record := range p.records {
		if record.Name == name && record.Type == recordType {
			return record, nil
		}
	}
	return Record{}, nil
}

func (p *testProvider) Create(record Record) (Record, error) {
	record.ID = "created"
	p.created = append(p.created, record)
	return record, nil
}

func (p *testProvider) Update(record Record) (Record, error) {
	p.updated = append(p.updated, record)
	return record, nil
}

func (p *testProvider) Delete(record Record) error {
	p.deleted = append(p.deleted, record)
	return nil
}

func TestUpdateRecordsUsesZoneSpecificIPOverrides(t *testing.T) {
	provider := &testProvider{}
	handler := Handler{DnsCfg: config.DNS{EnableIP4: true, EnableIP6: true}}
	domains := []config.DomainRecord{
		{
			DomainRecordBase: config.DomainRecordBase{Name: "app.example.com"},
			Overrides: map[string]config.DomainRecordBase{
				"zone1": {IP4: "10.0.0.5", IP6: "2001:db8::5"},
			},
		},
	}

	handler.updateRecords(provider, domains, "zone1")

	if len(provider.created) != 2 {
		t.Fatalf("created records = %#v, want 2 records", provider.created)
	}
	assertCreatedRecord(t, provider.created, constants.RecordTypeA, "10.0.0.5")
	assertCreatedRecord(t, provider.created, constants.RecordTypeAAAA, "2001:db8::5")
}

func TestUpdateRecordsZoneSpecificCNAMESuppressesBaseIPs(t *testing.T) {
	provider := &testProvider{}
	handler := Handler{DnsCfg: config.DNS{EnableIP4: true, EnableIP6: true}}
	domains := []config.DomainRecord{
		{
			DomainRecordBase: config.DomainRecordBase{
				Name: "app.example.com",
				IP4:  "10.0.0.1",
				IP6:  "2001:db8::1",
			},
			Overrides: map[string]config.DomainRecordBase{
				"zone1": {CName: "target.example.com"},
			},
		},
	}

	handler.updateRecords(provider, domains, "zone1")

	if len(provider.created) != 1 {
		t.Fatalf("created records = %#v, want 1 record", provider.created)
	}
	assertCreatedRecord(t, provider.created, constants.RecordTypeCNAME, "target.example.com")
}

func assertCreatedRecord(t *testing.T, records []Record, recordType, content string) {
	t.Helper()
	for _, record := range records {
		if record.Type == recordType && record.Content == content {
			return
		}
	}
	t.Fatalf("missing created %s record with content %q in %#v", recordType, content, records)
}
