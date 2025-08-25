package rfc2136

import (
	"fmt"
	"strings"
	"github.com/miekg/dns"
	internalDns "github.com/Tarow/dockdns/internal/dns"
	"github.com/go-acme/lego/v4/providers/dns/rfc2136"
)

// Alias for compatibility with tests and other code
type Rfc2136Provider = LegoRFC2136Provider

type LegoRFC2136Provider struct {
	Provider   *rfc2136.DNSProvider
	zone       string
	nameserver string
}

func New(server, port, protocol, tsigName, tsigSecret, tsigAlgo, zone string) LegoRFC2136Provider {
	config := rfc2136.NewDefaultConfig()
	config.Nameserver = fmt.Sprintf("%s:%s", server, port)
	config.TSIGKey = tsigName
	config.TSIGSecret = tsigSecret
	config.TSIGAlgorithm = tsigAlgo
	// TTL and other config can be set here if needed
	provider, err := rfc2136.NewDNSProviderConfig(config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create lego RFC2136 provider: %v", err))
	}
       return LegoRFC2136Provider{
	       Provider: provider,
	       zone:     zone,
	       nameserver: fmt.Sprintf("%s:%s", server, port),
       }
}

func (p LegoRFC2136Provider) List() ([]internalDns.Record, error) {
       // Query all TXT records in the zone using miekg/dns
       var records []internalDns.Record
       m := new(dns.Msg)
       m.SetQuestion(dns.Fqdn(p.zone), dns.TypeTXT)
       c := new(dns.Client)
	resp, _, err := c.Exchange(m, p.nameserver)
       if err != nil {
	       return nil, fmt.Errorf("DNS query failed: %v", err)
       }
       for _, ans := range resp.Answer {
	       if txt, ok := ans.(*dns.TXT); ok {
		       name := strings.TrimSuffix(txt.Hdr.Name, ".")
		       for _, txtVal := range txt.Txt {
			       records = append(records, internalDns.Record{
				       Name:    name,
				       Type:    "TXT",
				       Content: txtVal,
				       TTL:     int(txt.Hdr.Ttl),
			       })
		       }
	       }
       }
       return records, nil
}

func (p LegoRFC2136Provider) Get(domain, recordType string) (internalDns.Record, error) {
       // Query a specific record in the zone using miekg/dns
       if recordType != "TXT" {
	       return internalDns.Record{}, fmt.Errorf("only TXT records are supported")
       }
       fqdn := dns.Fqdn(domain + "." + p.zone)
       m := new(dns.Msg)
       m.SetQuestion(fqdn, dns.TypeTXT)
       c := new(dns.Client)
	resp, _, err := c.Exchange(m, p.nameserver)
       if err != nil {
	       return internalDns.Record{}, fmt.Errorf("DNS query failed: %v", err)
       }
       for _, ans := range resp.Answer {
	       if txt, ok := ans.(*dns.TXT); ok {
		       name := strings.TrimSuffix(txt.Hdr.Name, ".")
		       for _, txtVal := range txt.Txt {
			       return internalDns.Record{
				       Name:    name,
				       Type:    "TXT",
				       Content: txtVal,
				       TTL:     int(txt.Hdr.Ttl),
			       }, nil
		       }
	       }
       }
       return internalDns.Record{}, fmt.Errorf("TXT record not found for domain: %s", domain)
}

func (p LegoRFC2136Provider) Create(record internalDns.Record) (internalDns.Record, error) {
	// Only TXT records are supported by lego
	if record.Type != "TXT" {
		return internalDns.Record{}, fmt.Errorf("lego RFC2136 only supports TXT records")
	}
	err := p.Provider.Present(record.Name+"."+p.zone, "", record.Content)
	if err != nil {
		return internalDns.Record{}, err
	}
	return record, nil
}

func (p LegoRFC2136Provider) Update(record internalDns.Record) (internalDns.Record, error) {
	// For RFC2136, update is delete then create
	err := p.Delete(record)
	if err != nil {
		return internalDns.Record{}, err
	}
	return p.Create(record)
}

func (p LegoRFC2136Provider) Delete(record internalDns.Record) error {
	if record.Type != "TXT" {
		return fmt.Errorf("lego RFC2136 only supports TXT records")
	}
	err := p.Provider.CleanUp(record.Name+"."+p.zone, "", record.Content)
	if err != nil {
		return err
	}
	return nil
}
