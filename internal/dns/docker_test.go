package dns

import (
	"testing"

	"github.com/Tarow/dockdns/internal/config"
)

func TestParseProviderOverrides_NewFormat(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected map[string]config.DomainRecordBase
	}{
		{
			name: "a record override",
			labels: map[string]string{
				"dockdns.zone1.a": "10.0.0.5",
			},
			expected: map[string]config.DomainRecordBase{
				"zone1": {IP4: "10.0.0.5"},
			},
		},
		{
			name: "aaaa record override",
			labels: map[string]string{
				"dockdns.zone1.aaaa": "2001:db8::5",
			},
			expected: map[string]config.DomainRecordBase{
				"zone1": {IP6: "2001:db8::5"},
			},
		},
		{
			name: "cname override",
			labels: map[string]string{
				"dockdns.technitium-internal.cname": "internal.example.com",
			},
			expected: map[string]config.DomainRecordBase{
				"technitium-internal": {CName: "internal.example.com"},
			},
		},
		{
			name: "ttl override",
			labels: map[string]string{
				"dockdns.zone1.ttl": "600",
			},
			expected: map[string]config.DomainRecordBase{
				"zone1": {TTL: 600},
			},
		},
		{
			name: "proxied override",
			labels: map[string]string{
				"dockdns.cloudflare-prod.proxied": "true",
			},
			expected: map[string]config.DomainRecordBase{
				"cloudflare-prod": {Proxied: true},
			},
		},
		{
			name: "comment override",
			labels: map[string]string{
				"dockdns.zone1.comment": "Zone-specific comment",
			},
			expected: map[string]config.DomainRecordBase{
				"zone1": {Comment: "Zone-specific comment"},
			},
		},
		{
			name: "multiple fields in one zone merge into a single override block",
			labels: map[string]string{
				"dockdns.technitium-internal.cname":   "internal.local",
				"dockdns.technitium-internal.comment": "Internal server",
			},
			expected: map[string]config.DomainRecordBase{
				"technitium-internal": {CName: "internal.local", Comment: "Internal server"},
			},
		},
		{
			name: "multiple overrides for different zones",
			labels: map[string]string{
				"dockdns.zone1.a":                   "10.0.0.5",
				"dockdns.zone2.a":                   "10.0.0.6",
				"dockdns.cloudflare-prod.proxied":   "true",
				"dockdns.technitium-internal.cname": "internal.local",
			},
			expected: map[string]config.DomainRecordBase{
				"zone1":               {IP4: "10.0.0.5"},
				"zone2":               {IP4: "10.0.0.6"},
				"cloudflare-prod":     {Proxied: true},
				"technitium-internal": {CName: "internal.local"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &config.DomainRecord{}
			parseProviderOverrides(tt.labels, record)

			if !overridesEqual(record.Overrides, tt.expected) {
				t.Errorf("Overrides = %#v, want %#v", record.Overrides, tt.expected)
			}
		})
	}
}

func TestParseProviderOverrides_MultipleZones(t *testing.T) {
	// Test multiple overrides for multiple zones
	labels := map[string]string{
		"dockdns.zone1.a":             "10.0.0.5",
		"dockdns.zone2.cname":         "target.com",
		"dockdns.cloudflare-prod.ttl": "600",
		"dockdns.technitium.proxied":  "false",
		"dockdns.zone3.comment":       "Production server",
		"dockdns.zone4.aaaa":          "2001:db8::10",
	}

	record := &config.DomainRecord{}
	parseProviderOverrides(labels, record)

	// Verify all overrides were parsed correctly
	if record.Overrides["zone1"].IP4 != "10.0.0.5" {
		t.Errorf("Overrides[zone1].IP4 = %v, want 10.0.0.5", record.Overrides["zone1"].IP4)
	}

	if record.Overrides["zone2"].CName != "target.com" {
		t.Errorf("Overrides[zone2].CName = %v, want target.com", record.Overrides["zone2"].CName)
	}

	if record.Overrides["cloudflare-prod"].TTL != 600 {
		t.Errorf("Overrides[cloudflare-prod].TTL = %v, want 600", record.Overrides["cloudflare-prod"].TTL)
	}

	// Explicit override to false must be recorded (present in the map).
	if _, ok := record.Overrides["technitium"]; !ok {
		t.Errorf("Overrides[technitium] should exist for explicit proxied=false")
	}
	if record.Overrides["technitium"].Proxied != false {
		t.Errorf("Overrides[technitium].Proxied = %v, want false", record.Overrides["technitium"].Proxied)
	}

	if record.Overrides["zone3"].Comment != "Production server" {
		t.Errorf("Overrides[zone3].Comment = %v, want 'Production server'", record.Overrides["zone3"].Comment)
	}

	if record.Overrides["zone4"].IP6 != "2001:db8::10" {
		t.Errorf("Overrides[zone4].IP6 = %v, want '2001:db8::10'", record.Overrides["zone4"].IP6)
	}
}

func TestParseProviderOverrides_SkipsInvalidLabels(t *testing.T) {
	labels := map[string]string{
		"dockdns.name":          "test.com",     // Not an override
		"dockdns.a":             "10.0.0.1",     // Not an override
		"dockdns.zone1.invalid": "value",        // Invalid field
		"dockdns.zone1.ttl":     "not-a-number", // Invalid TTL
		"dockdns.zone1.proxied": "not-a-bool",   // Invalid boolean
		"dockdns.zone1.a":       "",             // Empty value
		"dockdns":               "value",        // No parts
		"other.zone1.a":         "10.0.0.5",     // Wrong prefix
	}

	record := &config.DomainRecord{}
	parseProviderOverrides(labels, record)

	// No valid overrides should have been recorded.
	if len(record.Overrides) != 0 {
		t.Errorf("Overrides should be empty, got %#v", record.Overrides)
	}
}

// overridesEqual compares two override maps by value.
func overridesEqual(a, b map[string]config.DomainRecordBase) bool {
	if len(a) != len(b) {
		return false
	}
	for k, av := range a {
		bv, ok := b[k]
		if !ok || av != bv {
			return false
		}
	}
	return true
}
