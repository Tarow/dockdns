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
			record:     DomainRecord{IP4: "10.0.0.1", IP6: "::1", CName: "target.com"},
			recordType: constants.RecordTypeA,
			zoneID:     "zone1",
			expected:   "10.0.0.1",
		},
		{
			name: "A record with zone-specific override",
			record: DomainRecord{
				IP4:          "10.0.0.1",
				IP4Overrides: map[string]string{"zone1": "10.0.0.5"},
			},
			recordType: constants.RecordTypeA,
			zoneID:     "zone1",
			expected:   "10.0.0.5",
		},
		{
			name: "A record override not matching zone",
			record: DomainRecord{
				IP4:          "10.0.0.1",
				IP4Overrides: map[string]string{"other-zone": "10.0.0.5"},
			},
			recordType: constants.RecordTypeA,
			zoneID:     "zone1",
			expected:   "10.0.0.1",
		},
		{
			name: "A record override with empty value falls back to default",
			record: DomainRecord{
				IP4:          "10.0.0.1",
				IP4Overrides: map[string]string{"zone1": ""},
			},
			recordType: constants.RecordTypeA,
			zoneID:     "zone1",
			expected:   "10.0.0.1",
		},
		{
			name:       "AAAA record returns IP6",
			record:     DomainRecord{IP4: "10.0.0.1", IP6: "2001:db8::1", CName: "target.com"},
			recordType: constants.RecordTypeAAAA,
			zoneID:     "zone1",
			expected:   "2001:db8::1",
		},
		{
			name: "AAAA record with zone-specific override",
			record: DomainRecord{
				IP6:          "2001:db8::1",
				IP6Overrides: map[string]string{"zone1": "2001:db8::5"},
			},
			recordType: constants.RecordTypeAAAA,
			zoneID:     "zone1",
			expected:   "2001:db8::5",
		},
		{
			name: "AAAA record override not matching zone",
			record: DomainRecord{
				IP6:          "2001:db8::1",
				IP6Overrides: map[string]string{"other-zone": "2001:db8::5"},
			},
			recordType: constants.RecordTypeAAAA,
			zoneID:     "zone1",
			expected:   "2001:db8::1",
		},
		{
			name: "AAAA record override with empty value falls back to default",
			record: DomainRecord{
				IP6:          "2001:db8::1",
				IP6Overrides: map[string]string{"zone1": ""},
			},
			recordType: constants.RecordTypeAAAA,
			zoneID:     "zone1",
			expected:   "2001:db8::1",
		},
		{
			name:       "CNAME record returns CName",
			record:     DomainRecord{IP4: "10.0.0.1", CName: "target.com"},
			recordType: constants.RecordTypeCNAME,
			zoneID:     "zone1",
			expected:   "target.com",
		},
		{
			name: "CNAME with zone-specific override",
			record: DomainRecord{
				CName:          "default-target.com",
				CNameOverrides: map[string]string{"zone1": "override-target.com"},
			},
			recordType: constants.RecordTypeCNAME,
			zoneID:     "zone1",
			expected:   "override-target.com",
		},
		{
			name: "CNAME override not matching zone",
			record: DomainRecord{
				CName:          "default-target.com",
				CNameOverrides: map[string]string{"other-zone": "override-target.com"},
			},
			recordType: constants.RecordTypeCNAME,
			zoneID:     "zone1",
			expected:   "default-target.com",
		},
		{
			name: "CNAME override with empty value falls back to default",
			record: DomainRecord{
				CName:          "default-target.com",
				CNameOverrides: map[string]string{"zone1": ""},
			},
			recordType: constants.RecordTypeCNAME,
			zoneID:     "zone1",
			expected:   "default-target.com",
		},
		{
			name:       "Unknown record type returns empty",
			record:     DomainRecord{IP4: "10.0.0.1"},
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
			record:   DomainRecord{Proxied: true},
			zoneID:   "zone1",
			expected: true,
		},
		{
			name:     "default false",
			record:   DomainRecord{Proxied: false},
			zoneID:   "zone1",
			expected: false,
		},
		{
			name: "zone-specific override to true",
			record: DomainRecord{
				Proxied:          false,
				ProxiedOverrides: map[string]bool{"zone1": true},
			},
			zoneID:   "zone1",
			expected: true,
		},
		{
			name: "zone-specific override to false",
			record: DomainRecord{
				Proxied:          true,
				ProxiedOverrides: map[string]bool{"zone1": false},
			},
			zoneID:   "zone1",
			expected: false,
		},
		{
			name: "override not matching zone",
			record: DomainRecord{
				Proxied:          true,
				ProxiedOverrides: map[string]bool{"other-zone": false},
			},
			zoneID:   "zone1",
			expected: true,
		},
		{
			name:     "nil overrides map",
			record:   DomainRecord{Proxied: true, ProxiedOverrides: nil},
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
			record:   DomainRecord{TTL: 300},
			zoneID:   "zone1",
			expected: 300,
		},
		{
			name: "zone-specific override",
			record: DomainRecord{
				TTL:          300,
				TTLOverrides: map[string]int{"zone1": 600},
			},
			zoneID:   "zone1",
			expected: 600,
		},
		{
			name: "override not matching zone",
			record: DomainRecord{
				TTL:          300,
				TTLOverrides: map[string]int{"other-zone": 600},
			},
			zoneID:   "zone1",
			expected: 300,
		},
		{
			name:     "nil overrides map",
			record:   DomainRecord{TTL: 300, TTLOverrides: nil},
			zoneID:   "zone1",
			expected: 300,
		},
		{
			name: "zero value override is valid",
			record: DomainRecord{
				TTL:          300,
				TTLOverrides: map[string]int{"zone1": 0},
			},
			zoneID:   "zone1",
			expected: 0,
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
			record:   DomainRecord{Comment: "default comment"},
			zoneID:   "zone1",
			expected: "default comment",
		},
		{
			name: "zone-specific override",
			record: DomainRecord{
				Comment:          "default comment",
				CommentOverrides: map[string]string{"zone1": "zone-specific comment"},
			},
			zoneID:   "zone1",
			expected: "zone-specific comment",
		},
		{
			name: "override not matching zone",
			record: DomainRecord{
				Comment:          "default comment",
				CommentOverrides: map[string]string{"other-zone": "other comment"},
			},
			zoneID:   "zone1",
			expected: "default comment",
		},
		{
			name:     "nil overrides map",
			record:   DomainRecord{Comment: "default comment", CommentOverrides: nil},
			zoneID:   "zone1",
			expected: "default comment",
		},
		{
			name: "empty string override is valid",
			record: DomainRecord{
				Comment:          "default comment",
				CommentOverrides: map[string]string{"zone1": ""},
			},
			zoneID:   "zone1",
			expected: "",
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
	record := DomainRecord{
		IP4:   "10.0.0.1",
		IP6:   "::1",
		CName: "target.com",
	}

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
