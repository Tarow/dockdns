package config

import (
	"os"
	"testing"

	"github.com/Tarow/dockdns/internal/constants"
)

func TestZone_GetKey(t *testing.T) {
	tests := []struct {
		name     string
		zone     Zone
		expected string
	}{
		{
			name:     "with ID set",
			zone:     Zone{Name: "example.com", ID: "my-custom-id"},
			expected: "my-custom-id",
		},
		{
			name:     "without ID (fallback to Name)",
			zone:     Zone{Name: "example.com", ID: ""},
			expected: "example.com",
		},
		{
			name:     "empty ID and Name",
			zone:     Zone{Name: "", ID: ""},
			expected: "",
		},
		{
			name:     "ID with special characters",
			zone:     Zone{Name: "example.com", ID: "cf-prod-zone"},
			expected: "cf-prod-zone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.zone.GetKey(); got != tt.expected {
				t.Errorf("Zone.GetKey() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// base is a small helper to build a DomainRecord with just the inline base fields set.
func base(b DomainRecordBase) DomainRecord {
	return DomainRecord{DomainRecordBase: b}
}

// withOverrides builds a DomainRecord with base fields and a per-zone override map.
func withOverrides(b DomainRecordBase, overrides map[string]DomainRecordBase) DomainRecord {
	return DomainRecord{DomainRecordBase: b, Overrides: overrides}
}

func TestDomainRecord_GetContentForZone(t *testing.T) {
	tests := []struct {
		name       string
		record     DomainRecord
		recordType string
		zoneID     string
		expected   string
	}{
		{
			name:       "A record returns IP4",
			record:     base(DomainRecordBase{IP4: "10.0.0.1", IP6: "::1", CName: "target.com"}),
			recordType: constants.RecordTypeA,
			zoneID:     "zone1",
			expected:   "10.0.0.1",
		},
		{
			name: "A record with zone-specific override",
			record: withOverrides(
				DomainRecordBase{IP4: "10.0.0.1"},
				map[string]DomainRecordBase{"zone1": {IP4: "10.0.0.5"}},
			),
			recordType: constants.RecordTypeA,
			zoneID:     "zone1",
			expected:   "10.0.0.5",
		},
		{
			name: "A record override not matching zone",
			record: withOverrides(
				DomainRecordBase{IP4: "10.0.0.1"},
				map[string]DomainRecordBase{"other-zone": {IP4: "10.0.0.5"}},
			),
			recordType: constants.RecordTypeA,
			zoneID:     "zone1",
			expected:   "10.0.0.1",
		},
		{
			name: "A record override with empty value falls back to default",
			record: withOverrides(
				DomainRecordBase{IP4: "10.0.0.1"},
				map[string]DomainRecordBase{"zone1": {IP4: ""}},
			),
			recordType: constants.RecordTypeA,
			zoneID:     "zone1",
			expected:   "10.0.0.1",
		},
		{
			name:       "AAAA record returns IP6",
			record:     base(DomainRecordBase{IP4: "10.0.0.1", IP6: "2001:db8::1", CName: "target.com"}),
			recordType: constants.RecordTypeAAAA,
			zoneID:     "zone1",
			expected:   "2001:db8::1",
		},
		{
			name: "AAAA record with zone-specific override",
			record: withOverrides(
				DomainRecordBase{IP6: "2001:db8::1"},
				map[string]DomainRecordBase{"zone1": {IP6: "2001:db8::5"}},
			),
			recordType: constants.RecordTypeAAAA,
			zoneID:     "zone1",
			expected:   "2001:db8::5",
		},
		{
			name: "AAAA record override not matching zone",
			record: withOverrides(
				DomainRecordBase{IP6: "2001:db8::1"},
				map[string]DomainRecordBase{"other-zone": {IP6: "2001:db8::5"}},
			),
			recordType: constants.RecordTypeAAAA,
			zoneID:     "zone1",
			expected:   "2001:db8::1",
		},
		{
			name: "AAAA record override with empty value falls back to default",
			record: withOverrides(
				DomainRecordBase{IP6: "2001:db8::1"},
				map[string]DomainRecordBase{"zone1": {IP6: ""}},
			),
			recordType: constants.RecordTypeAAAA,
			zoneID:     "zone1",
			expected:   "2001:db8::1",
		},
		{
			name:       "CNAME record returns CName",
			record:     base(DomainRecordBase{IP4: "10.0.0.1", CName: "target.com"}),
			recordType: constants.RecordTypeCNAME,
			zoneID:     "zone1",
			expected:   "target.com",
		},
		{
			name: "CNAME with zone-specific override",
			record: withOverrides(
				DomainRecordBase{CName: "default-target.com"},
				map[string]DomainRecordBase{"zone1": {CName: "override-target.com"}},
			),
			recordType: constants.RecordTypeCNAME,
			zoneID:     "zone1",
			expected:   "override-target.com",
		},
		{
			name: "CNAME override not matching zone",
			record: withOverrides(
				DomainRecordBase{CName: "default-target.com"},
				map[string]DomainRecordBase{"other-zone": {CName: "override-target.com"}},
			),
			recordType: constants.RecordTypeCNAME,
			zoneID:     "zone1",
			expected:   "default-target.com",
		},
		{
			name: "CNAME override with empty value falls back to default",
			record: withOverrides(
				DomainRecordBase{CName: "default-target.com"},
				map[string]DomainRecordBase{"zone1": {CName: ""}},
			),
			recordType: constants.RecordTypeCNAME,
			zoneID:     "zone1",
			expected:   "default-target.com",
		},
		{
			name:       "Unknown record type returns empty",
			record:     base(DomainRecordBase{IP4: "10.0.0.1"}),
			recordType: "MX",
			zoneID:     "zone1",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.record.GetContentForZone(tt.recordType, tt.zoneID); got != tt.expected {
				t.Errorf("GetContentForZone() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDomainRecord_GetProxiedForZone(t *testing.T) {
	tests := []struct {
		name     string
		record   DomainRecord
		zoneID   string
		expected bool
	}{
		{
			name:     "default proxied value",
			record:   base(DomainRecordBase{Proxied: true}),
			zoneID:   "zone1",
			expected: true,
		},
		{
			name:     "default false",
			record:   base(DomainRecordBase{Proxied: false}),
			zoneID:   "zone1",
			expected: false,
		},
		{
			name: "zone-specific override to true",
			record: withOverrides(
				DomainRecordBase{Proxied: false},
				map[string]DomainRecordBase{"zone1": {Proxied: true}},
			),
			zoneID:   "zone1",
			expected: true,
		},
		{
			name: "zone-specific override to false",
			record: withOverrides(
				DomainRecordBase{Proxied: true},
				map[string]DomainRecordBase{"zone1": {Proxied: false}},
			),
			zoneID:   "zone1",
			expected: false,
		},
		{
			name: "override not matching zone",
			record: withOverrides(
				DomainRecordBase{Proxied: true},
				map[string]DomainRecordBase{"other-zone": {Proxied: false}},
			),
			zoneID:   "zone1",
			expected: true,
		},
		{
			name:     "nil overrides map",
			record:   base(DomainRecordBase{Proxied: true}),
			zoneID:   "zone1",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.record.GetProxiedForZone(tt.zoneID); got != tt.expected {
				t.Errorf("GetProxiedForZone() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDomainRecord_GetTTLForZone(t *testing.T) {
	tests := []struct {
		name     string
		record   DomainRecord
		zoneID   string
		expected int
	}{
		{
			name:     "default TTL value",
			record:   base(DomainRecordBase{TTL: 300}),
			zoneID:   "zone1",
			expected: 300,
		},
		{
			name: "zone-specific override",
			record: withOverrides(
				DomainRecordBase{TTL: 300},
				map[string]DomainRecordBase{"zone1": {TTL: 600}},
			),
			zoneID:   "zone1",
			expected: 600,
		},
		{
			name: "override not matching zone",
			record: withOverrides(
				DomainRecordBase{TTL: 300},
				map[string]DomainRecordBase{"other-zone": {TTL: 600}},
			),
			zoneID:   "zone1",
			expected: 300,
		},
		{
			name:     "nil overrides map",
			record:   base(DomainRecordBase{TTL: 300}),
			zoneID:   "zone1",
			expected: 300,
		},
		{
			name: "zero value override inherits base TTL",
			record: withOverrides(
				DomainRecordBase{TTL: 300},
				map[string]DomainRecordBase{"zone1": {TTL: 0}},
			),
			zoneID:   "zone1",
			expected: 300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.record.GetTTLForZone(tt.zoneID); got != tt.expected {
				t.Errorf("GetTTLForZone() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDomainRecord_GetCommentForZone(t *testing.T) {
	tests := []struct {
		name     string
		record   DomainRecord
		zoneID   string
		expected string
	}{
		{
			name:     "default comment value",
			record:   base(DomainRecordBase{Comment: "default comment"}),
			zoneID:   "zone1",
			expected: "default comment",
		},
		{
			name: "zone-specific override",
			record: withOverrides(
				DomainRecordBase{Comment: "default comment"},
				map[string]DomainRecordBase{"zone1": {Comment: "zone-specific comment"}},
			),
			zoneID:   "zone1",
			expected: "zone-specific comment",
		},
		{
			name: "override not matching zone",
			record: withOverrides(
				DomainRecordBase{Comment: "default comment"},
				map[string]DomainRecordBase{"other-zone": {Comment: "other comment"}},
			),
			zoneID:   "zone1",
			expected: "default comment",
		},
		{
			name:     "nil overrides map",
			record:   base(DomainRecordBase{Comment: "default comment"}),
			zoneID:   "zone1",
			expected: "default comment",
		},
		{
			name: "empty string override inherits base comment",
			record: withOverrides(
				DomainRecordBase{Comment: "default comment"},
				map[string]DomainRecordBase{"zone1": {Comment: ""}},
			),
			zoneID:   "zone1",
			expected: "default comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.record.GetCommentForZone(tt.zoneID); got != tt.expected {
				t.Errorf("GetCommentForZone() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDomainRecord_GetContent(t *testing.T) {
	record := base(DomainRecordBase{
		IP4:   "10.0.0.1",
		IP6:   "::1",
		CName: "target.com",
	})

	tests := []struct {
		name       string
		recordType string
		expected   string
	}{
		{"A record", constants.RecordTypeA, "10.0.0.1"},
		{"AAAA record", constants.RecordTypeAAAA, "::1"},
		{"CNAME record", constants.RecordTypeCNAME, "target.com"},
		{"Unknown type", "TXT", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := record.GetContent(tt.recordType); got != tt.expected {
				t.Errorf("GetContent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppConfig_EnrichZoneSecretsFromEnv(t *testing.T) {
	// Save and restore environment
	originalToken := os.Getenv("EXAMPLE_COM_API_TOKEN")
	originalZoneID := os.Getenv("EXAMPLE_COM_ZONE_ID")
	defer func() {
		if originalToken != "" {
			os.Setenv("EXAMPLE_COM_API_TOKEN", originalToken)
		} else {
			os.Unsetenv("EXAMPLE_COM_API_TOKEN")
		}
		if originalZoneID != "" {
			os.Setenv("EXAMPLE_COM_ZONE_ID", originalZoneID)
		} else {
			os.Unsetenv("EXAMPLE_COM_ZONE_ID")
		}
	}()

	t.Run("enriches ApiToken from environment", func(t *testing.T) {
		os.Setenv("EXAMPLE_COM_API_TOKEN", "env-token-value")
		os.Unsetenv("EXAMPLE_COM_ZONE_ID")

		cfg := &AppConfig{
			Zones: Zones{
				{Name: "example.com", ApiToken: ""},
			},
		}

		cfg.EnrichZoneSecretsFromEnv()

		if cfg.Zones[0].ApiToken != "env-token-value" {
			t.Errorf("ApiToken = %v, want %v", cfg.Zones[0].ApiToken, "env-token-value")
		}
	})

	t.Run("enriches ZoneID from environment", func(t *testing.T) {
		os.Unsetenv("EXAMPLE_COM_API_TOKEN")
		os.Setenv("EXAMPLE_COM_ZONE_ID", "env-zone-id")

		cfg := &AppConfig{
			Zones: Zones{
				{Name: "example.com", ZoneID: ""},
			},
		}

		cfg.EnrichZoneSecretsFromEnv()

		if cfg.Zones[0].ZoneID != "env-zone-id" {
			t.Errorf("ZoneID = %v, want %v", cfg.Zones[0].ZoneID, "env-zone-id")
		}
	})

	t.Run("does not override existing values", func(t *testing.T) {
		os.Setenv("EXAMPLE_COM_API_TOKEN", "env-token")
		os.Setenv("EXAMPLE_COM_ZONE_ID", "env-zone-id")

		cfg := &AppConfig{
			Zones: Zones{
				{Name: "example.com", ApiToken: "config-token", ZoneID: "config-zone-id"},
			},
		}

		cfg.EnrichZoneSecretsFromEnv()

		if cfg.Zones[0].ApiToken != "config-token" {
			t.Errorf("ApiToken = %v, want %v", cfg.Zones[0].ApiToken, "config-token")
		}
		if cfg.Zones[0].ZoneID != "config-zone-id" {
			t.Errorf("ZoneID = %v, want %v", cfg.Zones[0].ZoneID, "config-zone-id")
		}
	})

	t.Run("handles special characters in zone name", func(t *testing.T) {
		os.Setenv("INT_SCHITTKO_ME_API_TOKEN", "special-token")
		defer os.Unsetenv("INT_SCHITTKO_ME_API_TOKEN")

		cfg := &AppConfig{
			Zones: Zones{
				{Name: "int.schittko.me", ApiToken: ""},
			},
		}

		cfg.EnrichZoneSecretsFromEnv()

		if cfg.Zones[0].ApiToken != "special-token" {
			t.Errorf("ApiToken = %v, want %v", cfg.Zones[0].ApiToken, "special-token")
		}
	})
}
