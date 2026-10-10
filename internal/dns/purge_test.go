package dns

import (
	"testing"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
)

func TestContainsRecordUsesZoneSpecificCNAME(t *testing.T) {
	domains := []config.DomainRecord{
		{
			DomainRecordBase: config.DomainRecordBase{
				Name: "app.example.com",
				IP4:  "10.0.0.1",
			},
			Overrides: map[string]config.DomainRecordBase{
				"zone1": {CName: "target.example.com"},
			},
		},
	}
	dnsCfg := config.DNS{EnableIP4: true}

	if !containsRecord(domains, Record{Name: "app.example.com", Type: constants.RecordTypeCNAME}, dnsCfg, "zone1") {
		t.Fatal("zone-specific CNAME should be considered known")
	}
	if containsRecord(domains, Record{Name: "app.example.com", Type: constants.RecordTypeA}, dnsCfg, "zone1") {
		t.Fatal("base A should be unknown when zone-specific CNAME is effective")
	}
}

func TestContainsRecordUsesZoneSpecificIPOverrides(t *testing.T) {
	domains := []config.DomainRecord{
		{
			DomainRecordBase: config.DomainRecordBase{Name: "app.example.com"},
			Overrides: map[string]config.DomainRecordBase{
				"zone1": {IP4: "10.0.0.5", IP6: "2001:db8::5"},
			},
		},
	}
	dnsCfg := config.DNS{EnableIP4: true, EnableIP6: true}

	if !containsRecord(domains, Record{Name: "app.example.com", Type: constants.RecordTypeA}, dnsCfg, "zone1") {
		t.Fatal("zone-specific A should be considered known")
	}
	if !containsRecord(domains, Record{Name: "app.example.com", Type: constants.RecordTypeAAAA}, dnsCfg, "zone1") {
		t.Fatal("zone-specific AAAA should be considered known")
	}
	if containsRecord(domains, Record{Name: "app.example.com", Type: constants.RecordTypeA}, dnsCfg, "other-zone") {
		t.Fatal("A should be unknown for zones without effective A content")
	}
}

func TestPurgeUnknownRecordsKeepsZoneSpecificRecords(t *testing.T) {
	provider := &testProvider{
		records: []Record{
			{Name: "app.example.com", Type: constants.RecordTypeCNAME},
		},
	}
	handler := Handler{DnsCfg: config.DNS{PurgeUnknown: true}}
	domains := []config.DomainRecord{
		{
			DomainRecordBase: config.DomainRecordBase{Name: "app.example.com"},
			Overrides: map[string]config.DomainRecordBase{
				"zone1": {CName: "target.example.com"},
			},
		},
	}

	handler.purgeUnknownRecords(provider, domains, "zone1")

	if len(provider.deleted) != 0 {
		t.Fatalf("deleted records = %#v, want none", provider.deleted)
	}
}
