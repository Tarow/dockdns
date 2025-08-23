package rfc2136

import (
	"os"
	"testing"
	"github.com/bodgit/dns"
)

func TestBodgitRFC2136Update(t *testing.T) {
	server := os.Getenv("RFC2136_SERVER")
	port := os.Getenv("RFC2136_PORT")
	zone := os.Getenv("RFC2136_ZONE")
	tsigName := os.Getenv("RFC2136_TSIG_NAME")
	tsigSecret := os.Getenv("RFC2136_TSIG_SECRET")
	tsigAlgo := os.Getenv("RFC2136_TSIG_ALGO")
	if server == "" || port == "" || zone == "" || tsigName == "" || tsigSecret == "" || tsigAlgo == "" {
		t.Skip("RFC2136 env not set")
	}

	fqdn := dns.Fqdn("testrecord." + zone)
	m := new(dns.Msg)
	m.SetUpdate(dns.Fqdn(zone))
	rr, err := dns.NewRR(fqdn + " 60 IN A 127.0.0.1")
	if err != nil {
		t.Fatalf("NewRR failed: %v", err)
	}
	m.Insert([]dns.RR{rr})

	algos := []string{tsigAlgo, "hmac-sha256.", "hmac-sha256", "HMAC-SHA256", "HMAC-SHA256."}
	var lastErr error
	for _, algo := range algos {
		t.Logf("Trying TSIG algo: %s", algo)
		m.SetTsig(dns.Fqdn(tsigName), algo, 300, 0)
		c := new(dns.Client)
		c.Net = "tcp"
		c.TsigSecret = map[string]string{dns.Fqdn(tsigName): tsigSecret}
		resp, _, err := c.Exchange(m, server+":"+port)
		if err == nil && resp.Rcode == dns.RcodeSuccess {
			t.Logf("Update succeeded with algo: %s", algo)
			return
		}
		lastErr = err
	}
	t.Fatalf("Exchange failed for all algos, last error: %v", lastErr)
	}
