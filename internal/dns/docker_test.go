package dns

import (
	"testing"

	"github.com/Tarow/dockdns/internal/config"
)

func TestParseProviderOverrides_NewFormat(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected config.DomainRecord
	}{
		{
			name: "new format - a record override",
			labels: map[string]string{
				"dockdns.zone1.a": "10.0.0.5",
			},
			expected: config.DomainRecord{
				IP4Overrides: map[string]string{"zone1": "10.0.0.5"},
			},
		},
		{
			name: "new format - aaaa record override",
			labels: map[string]string{
				"dockdns.zone1.aaaa": "2001:db8::5",
			},
			expected: config.DomainRecord{
				IP6Overrides: map[string]string{"zone1": "2001:db8::5"},
			},
		},
		{
			name: "new format - cname override",
			labels: map[string]string{
				"dockdns.technitium-internal.cname": "internal.example.com",
			},
			expected: config.DomainRecord{
				CNameOverrides: map[string]string{"technitium-internal": "internal.example.com"},
			},
		},
		{
			name: "new format - ttl override",
			labels: map[string]string{
				"dockdns.zone1.ttl": "600",
			},
			expected: config.DomainRecord{
				TTLOverrides: map[string]int{"zone1": 600},
			},
		},
		{
			name: "new format - proxied override",
			labels: map[string]string{
				"dockdns.cloudflare-prod.proxied": "true",
			},
			expected: config.DomainRecord{
				ProxiedOverrides: map[string]bool{"cloudflare-prod": true},
			},
		},
		{
			name: "new format - comment override",
			labels: map[string]string{
				"dockdns.zone1.comment": "Zone-specific comment",
			},
			expected: config.DomainRecord{
				CommentOverrides: map[string]string{"zone1": "Zone-specific comment"},
			},
		},
		{
			name: "new format - multiple overrides for different zones",
			labels: map[string]string{
				"dockdns.zone1.a":                   "10.0.0.5",
				"dockdns.zone2.a":                   "10.0.0.6",
				"dockdns.cloudflare-prod.proxied":   "true",
				"dockdns.technitium-internal.cname": "internal.local",
			},
			expected: config.DomainRecord{
				IP4Overrides:     map[string]string{"zone1": "10.0.0.5", "zone2": "10.0.0.6"},
				CNameOverrides:   map[string]string{"technitium-internal": "internal.local"},
				ProxiedOverrides: map[string]bool{"cloudflare-prod": true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &config.DomainRecord{}
			parseProviderOverrides(tt.labels, record)

			// Check IP4Overrides
			if !mapsEqual(record.IP4Overrides, tt.expected.IP4Overrides) {
				t.Errorf("IP4Overrides = %v, want %v", record.IP4Overrides, tt.expected.IP4Overrides)
			}

			// Check IP6Overrides
			if !mapsEqual(record.IP6Overrides, tt.expected.IP6Overrides) {
				t.Errorf("IP6Overrides = %v, want %v", record.IP6Overrides, tt.expected.IP6Overrides)
			}

			// Check CNameOverrides
			if !mapsEqual(record.CNameOverrides, tt.expected.CNameOverrides) {
				t.Errorf("CNameOverrides = %v, want %v", record.CNameOverrides, tt.expected.CNameOverrides)
			}

			// Check TTLOverrides
			if !intMapsEqual(record.TTLOverrides, tt.expected.TTLOverrides) {
				t.Errorf("TTLOverrides = %v, want %v", record.TTLOverrides, tt.expected.TTLOverrides)
			}

			// Check ProxiedOverrides
			if !boolMapsEqual(record.ProxiedOverrides, tt.expected.ProxiedOverrides) {
				t.Errorf("ProxiedOverrides = %v, want %v", record.ProxiedOverrides, tt.expected.ProxiedOverrides)
			}

			// Check CommentOverrides
			if !mapsEqual(record.CommentOverrides, tt.expected.CommentOverrides) {
				t.Errorf("CommentOverrides = %v, want %v", record.CommentOverrides, tt.expected.CommentOverrides)
			}
		})
	}
}

func TestParseProviderOverrides_LegacyFormat(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected config.DomainRecord
	}{
		{
			name: "legacy format - a record override",
			labels: map[string]string{
				"dockdns.a.zone1": "10.0.0.5",
			},
			expected: config.DomainRecord{
				IP4Overrides: map[string]string{"zone1": "10.0.0.5"},
			},
		},
		{
			name: "legacy format - aaaa record override",
			labels: map[string]string{
				"dockdns.aaaa.zone1": "2001:db8::5",
			},
			expected: config.DomainRecord{
				IP6Overrides: map[string]string{"zone1": "2001:db8::5"},
			},
		},
		{
			name: "legacy format - cname override",
			labels: map[string]string{
				"dockdns.cname.technitium-internal": "internal.example.com",
			},
			expected: config.DomainRecord{
				CNameOverrides: map[string]string{"technitium-internal": "internal.example.com"},
			},
		},
		{
			name: "legacy format - ttl override",
			labels: map[string]string{
				"dockdns.ttl.zone1": "600",
			},
			expected: config.DomainRecord{
				TTLOverrides: map[string]int{"zone1": 600},
			},
		},
		{
			name: "legacy format - proxied override",
			labels: map[string]string{
				"dockdns.proxied.cloudflare-prod": "true",
			},
			expected: config.DomainRecord{
				ProxiedOverrides: map[string]bool{"cloudflare-prod": true},
			},
		},
		{
			name: "legacy format - comment override",
			labels: map[string]string{
				"dockdns.comment.zone1": "Zone-specific comment",
			},
			expected: config.DomainRecord{
				CommentOverrides: map[string]string{"zone1": "Zone-specific comment"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &config.DomainRecord{}
			parseProviderOverrides(tt.labels, record)

			// Check IP4Overrides
			if !mapsEqual(record.IP4Overrides, tt.expected.IP4Overrides) {
				t.Errorf("IP4Overrides = %v, want %v", record.IP4Overrides, tt.expected.IP4Overrides)
			}

			// Check IP6Overrides
			if !mapsEqual(record.IP6Overrides, tt.expected.IP6Overrides) {
				t.Errorf("IP6Overrides = %v, want %v", record.IP6Overrides, tt.expected.IP6Overrides)
			}

			// Check CNameOverrides
			if !mapsEqual(record.CNameOverrides, tt.expected.CNameOverrides) {
				t.Errorf("CNameOverrides = %v, want %v", record.CNameOverrides, tt.expected.CNameOverrides)
			}

			// Check TTLOverrides
			if !intMapsEqual(record.TTLOverrides, tt.expected.TTLOverrides) {
				t.Errorf("TTLOverrides = %v, want %v", record.TTLOverrides, tt.expected.TTLOverrides)
			}

			// Check ProxiedOverrides
			if !boolMapsEqual(record.ProxiedOverrides, tt.expected.ProxiedOverrides) {
				t.Errorf("ProxiedOverrides = %v, want %v", record.ProxiedOverrides, tt.expected.ProxiedOverrides)
			}

			// Check CommentOverrides
			if !mapsEqual(record.CommentOverrides, tt.expected.CommentOverrides) {
				t.Errorf("CommentOverrides = %v, want %v", record.CommentOverrides, tt.expected.CommentOverrides)
			}
		})
	}
}

func TestParseProviderOverrides_MixedFormats(t *testing.T) {
	// Test that both formats can be used together
	labels := map[string]string{
		"dockdns.zone1.a":               "10.0.0.5", // New format
		"dockdns.cname.zone2":           "old.com",  // Legacy format
		"dockdns.cloudflare-prod.ttl":   "600",      // New format
		"dockdns.proxied.technitium":    "false",    // Legacy format
		"dockdns.zone3.comment":         "New style comment",
		"dockdns.comment.zone4":         "Old style comment",
	}

	record := &config.DomainRecord{}
	parseProviderOverrides(labels, record)

	// Verify all overrides were parsed correctly
	if record.IP4Overrides["zone1"] != "10.0.0.5" {
		t.Errorf("IP4Overrides[zone1] = %v, want 10.0.0.5", record.IP4Overrides["zone1"])
	}

	if record.CNameOverrides["zone2"] != "old.com" {
		t.Errorf("CNameOverrides[zone2] = %v, want old.com", record.CNameOverrides["zone2"])
	}

	if record.TTLOverrides["cloudflare-prod"] != 600 {
		t.Errorf("TTLOverrides[cloudflare-prod] = %v, want 600", record.TTLOverrides["cloudflare-prod"])
	}

	if record.ProxiedOverrides["technitium"] != false {
		t.Errorf("ProxiedOverrides[technitium] = %v, want false", record.ProxiedOverrides["technitium"])
	}

	if record.CommentOverrides["zone3"] != "New style comment" {
		t.Errorf("CommentOverrides[zone3] = %v, want 'New style comment'", record.CommentOverrides["zone3"])
	}

	if record.CommentOverrides["zone4"] != "Old style comment" {
		t.Errorf("CommentOverrides[zone4] = %v, want 'Old style comment'", record.CommentOverrides["zone4"])
	}
}

func TestParseProviderOverrides_SkipsInvalidLabels(t *testing.T) {
	labels := map[string]string{
		"dockdns.name":              "test.com",     // Not an override
		"dockdns.a":                 "10.0.0.1",     // Not an override
		"dockdns.zone1.invalid":     "value",        // Invalid field
		"dockdns.zone1.ttl":         "not-a-number", // Invalid TTL
		"dockdns.zone1.proxied":     "not-a-bool",   // Invalid boolean
		"dockdns.zone1.a":           "",             // Empty value
		"dockdns":                   "value",        // No parts
		"other.zone1.a":             "10.0.0.5",     // Wrong prefix
	}

	record := &config.DomainRecord{}
	parseProviderOverrides(labels, record)

	// All overrides should be empty
	if len(record.IP4Overrides) != 0 {
		t.Errorf("IP4Overrides should be empty, got %v", record.IP4Overrides)
	}
	if len(record.TTLOverrides) != 0 {
		t.Errorf("TTLOverrides should be empty, got %v", record.TTLOverrides)
	}
	if len(record.ProxiedOverrides) != 0 {
		t.Errorf("ProxiedOverrides should be empty, got %v", record.ProxiedOverrides)
	}
}

// Helper functions for map comparison
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func intMapsEqual(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func boolMapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
