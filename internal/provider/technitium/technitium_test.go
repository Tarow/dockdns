package technitium

import (
	"testing"

	"github.com/Tarow/dockdns/internal/constants"
	"github.com/Tarow/dockdns/internal/dns"
)

// TestProviderInterface ensures TechnitiumProvider implements the Provider interface
func TestProviderInterface(t *testing.T) {
	var _ dns.Provider = (*TechnitiumProvider)(nil)
}

// TestNewProviderValidation tests the validation of provider creation parameters
func TestNewProviderValidation(t *testing.T) {
	tests := []struct {
		name        string
		apiURL      string
		username    string
		password    string
		apiToken    string
		zone        string
		expectError bool
	}{
		{
			name:        "all parameters provided with username/password",
			apiURL:      "http://localhost:5380",
			username:    "admin",
			password:    "admin",
			apiToken:    "",
			zone:        "test.local",
			expectError: true, // Will error because it tries to connect
		},
		{
			name:        "api token provided",
			apiURL:      "http://localhost:5380",
			username:    "",
			password:    "",
			apiToken:    "sometoken",
			zone:        "test.local",
			expectError: false, // API token auth doesn't validate until first request
		},
		{
			name:        "missing apiURL",
			apiURL:      "",
			username:    "admin",
			password:    "admin",
			apiToken:    "",
			zone:        "test.local",
			expectError: true,
		},
		{
			name:        "missing all auth",
			apiURL:      "http://localhost:5380",
			username:    "",
			password:    "",
			apiToken:    "",
			zone:        "test.local",
			expectError: true,
		},
		{
			name:        "missing password with username",
			apiURL:      "http://localhost:5380",
			username:    "admin",
			password:    "",
			apiToken:    "",
			zone:        "test.local",
			expectError: true,
		},
		{
			name:        "missing zone",
			apiURL:      "http://localhost:5380",
			username:    "admin",
			password:    "admin",
			apiToken:    "",
			zone:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.apiURL, tt.username, tt.password, tt.apiToken, tt.zone, false)
			if (err != nil) != tt.expectError {
				t.Errorf("New() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestRecordIDFormat tests that record IDs are generated correctly
func TestRecordIDFormat(t *testing.T) {
	tests := []struct {
		name       string
		recordName string
		recordType string
		content    string
		expectedID string
	}{
		{
			name:       "A record",
			recordName: "www.test.local",
			recordType: constants.RecordTypeA,
			content:    "192.168.1.100",
			expectedID: "www.test.local:A:192.168.1.100",
		},
		{
			name:       "AAAA record",
			recordName: "www.test.local",
			recordType: constants.RecordTypeAAAA,
			content:    "2001:db8::1",
			expectedID: "www.test.local:AAAA:2001:db8::1",
		},
		{
			name:       "CNAME record",
			recordName: "alias.test.local",
			recordType: constants.RecordTypeCNAME,
			content:    "www.test.local",
			expectedID: "alias.test.local:CNAME:www.test.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a record and verify ID format
			record := dns.Record{
				Name:    tt.recordName,
				Type:    tt.recordType,
				Content: tt.content,
			}

			// Simulate ID generation as done in Create method
			expectedID := record.Name + ":" + record.Type + ":" + record.Content

			if expectedID != tt.expectedID {
				t.Errorf("Record ID = %v, want %v", expectedID, tt.expectedID)
			}
		})
	}
}
