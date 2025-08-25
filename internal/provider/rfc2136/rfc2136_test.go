package rfc2136_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	internalDns "github.com/Tarow/dockdns/internal/dns"
	rfc2136 "github.com/Tarow/dockdns/internal/provider/rfc2136"
)

// loadEnv loads environment variables from a file
func loadEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
	return scanner.Err()
}

func getTestProvider() rfc2136.Rfc2136Provider {
	// Always load test.env before reading env vars
	_ = loadEnv("./internal/provider/rfc2136/test.env")
	server := os.Getenv("RFC2136_SERVER")
	port := os.Getenv("RFC2136_PORT")
	protocol := os.Getenv("RFC2136_PROTOCOL")
	tsigName := os.Getenv("RFC2136_TSIG_NAME")
	tsigSecret := os.Getenv("RFC2136_TSIG_SECRET")
	tsigAlgo := os.Getenv("RFC2136_TSIG_ALGO")
	zone := os.Getenv("RFC2136_ZONE")
	if server == "" || port == "" || tsigName == "" || tsigSecret == "" || tsigAlgo == "" || zone == "" {
		return rfc2136.Rfc2136Provider{Provider: nil}
	}
	return rfc2136.New(server, port, protocol, tsigName, tsigSecret, tsigAlgo, zone)
}

func TestCreateUpdateDelete(t *testing.T) {
	provider := getTestProvider()
	if provider.Provider == nil {
		t.Skip("RFC2136 provider not initialized; skipping test.")
	}
	record := internalDns.Record{
		Name:    "testrecord",
		Type:    "TXT",
		Content: "dockdns-test-value",
		TTL:     60,
	}
	// Create
	_, err := provider.Create(record)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	// Update
	updated := record
	updated.Content = "127.0.0.2"
	_, err = provider.Update(updated)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	// Delete
	err = provider.Delete(updated)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}
