package rfc2136

import (
	"os"
	"testing"

	internalDns "github.com/Tarow/dockdns/internal/dns"
)

func getTestProvider() rfc2136Provider {
	server := os.Getenv("RFC2136_SERVER")
	port := os.Getenv("RFC2136_PORT")
	tsigName := os.Getenv("RFC2136_TSIG_NAME")
	tsigSecret := os.Getenv("RFC2136_TSIG_SECRET")
	tsigAlgo := os.Getenv("RFC2136_TSIG_ALGO")
	zone := os.Getenv("RFC2136_ZONE")
	if server == "" || port == "" || tsigName == "" || tsigSecret == "" || tsigAlgo == "" || zone == "" {
		// Use .test zone for mock
		return New("mock", "53", "mock", "mock", "mock", "localdomain.test")
	}
	return New(server, port, tsigName, tsigSecret, tsigAlgo, zone)
}

func TestCreateRecord(t *testing.T) {
	provider := getTestProvider()
	record := internalDns.Record{
		Name:    "testrecord", // Let fqdn helper build FQDN
		Type:    "A",
		Content: "127.0.0.1",
		TTL:     60,
	}
	t.Logf("Attempting to create record: %+v", record)
	_, err := provider.Create(record)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
}

func TestDeleteRecord(t *testing.T) {
	provider := getTestProvider()
	record := internalDns.Record{
		Name:    "testrecord", // Let fqdn helper build FQDN
		Type:    "A",
		Content: "127.0.0.1",
		TTL:     60,
	}
	t.Logf("Attempting to delete record: %+v", record)
	err := provider.Delete(record)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

// To run against a real RFC2136 server, set these env vars:
// RFC2136_SERVER, RFC2136_PORT, RFC2136_TSIG_NAME, RFC2136_TSIG_SECRET, RFC2136_TSIG_ALGO, RFC2136_ZONE
// Example:
// RFC2136_SERVER=192.168.1.53 RFC2136_PORT=53 RFC2136_TSIG_NAME=keyname. RFC2136_TSIG_SECRET=base64secret RFC2136_TSIG_ALGO=hmac-sha256 RFC2136_ZONE=example.com go test -v ./internal/provider/rfc2136
