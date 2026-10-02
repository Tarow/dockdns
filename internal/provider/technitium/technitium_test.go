package technitium

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tarow/dockdns/internal/constants"
	"github.com/Tarow/dockdns/internal/dns"
)

// TestProviderInterface ensures TechnitiumProvider implements the Provider interface
func TestProviderInterface(t *testing.T) {
	var _ dns.Provider = (*TechnitiumProvider)(nil)
}

func TestCreateUsesDesiredComment(t *testing.T) {
	var gotComment string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/zones/records/add" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gotComment = r.Form.Get("comments")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	provider, err := New(server.URL, "", "", "token", "test.local", false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	record := dns.Record{
		Name:    "www.test.local",
		Type:    constants.RecordTypeA,
		Content: "192.168.1.100",
		TTL:     300,
		Comment: "desired comment",
	}
	created, err := provider.Create(record)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if gotComment != "desired comment" {
		t.Fatalf("comments form value = %q, want %q", gotComment, "desired comment")
	}
	if created.Comment != "desired comment" {
		t.Fatalf("created.Comment = %q, want %q", created.Comment, "desired comment")
	}
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
			if got := recordID(tt.recordName, tt.recordType, tt.content); got != tt.expectedID {
				t.Errorf("recordID() = %v, want %v", got, tt.expectedID)
			}
		})
	}
}

func TestParseRecordID(t *testing.T) {
	tests := []struct {
		name            string
		id              string
		expectedName    string
		expectedType    string
		expectedContent string
		expectedOK      bool
	}{
		{
			name:            "A record",
			id:              "www.test.local:A:192.168.1.100",
			expectedName:    "www.test.local",
			expectedType:    constants.RecordTypeA,
			expectedContent: "192.168.1.100",
			expectedOK:      true,
		},
		{
			name:            "AAAA record preserves IPv6 colons",
			id:              "www.test.local:AAAA:2001:db8::1",
			expectedName:    "www.test.local",
			expectedType:    constants.RecordTypeAAAA,
			expectedContent: "2001:db8::1",
			expectedOK:      true,
		},
		{
			name:       "invalid ID",
			id:         "invalid",
			expectedOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, recordType, content, ok := parseRecordID(tt.id)
			if ok != tt.expectedOK {
				t.Fatalf("parseRecordID() ok = %v, want %v", ok, tt.expectedOK)
			}
			if name != tt.expectedName || recordType != tt.expectedType || content != tt.expectedContent {
				t.Fatalf("parseRecordID() = (%q, %q, %q), want (%q, %q, %q)", name, recordType, content, tt.expectedName, tt.expectedType, tt.expectedContent)
			}
		})
	}
}
