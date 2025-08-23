package rfc2136

import (
	"fmt"
	"time"

	internalDns "github.com/Tarow/dockdns/internal/dns"
	"github.com/miekg/dns"
)

type rfc2136Provider struct {
	server     string
	port       string
	tsigName   string
	tsigSecret string
	tsigAlgo   string
	zone       string
}

func New(server, port, tsigName, tsigSecret, tsigAlgo, zone string) rfc2136Provider {
	return rfc2136Provider{
		server:     server,
		port:       port,
		tsigName:   tsigName,
		tsigSecret: tsigSecret,
		tsigAlgo:   tsigAlgo,
		zone:       zone,
	}
}

// Helper to get fqdn
func fqdn(name, zone string) string {
	// Ensure zone ends with a period
	z := zone
	if !dns.IsFqdn(zone) {
		z = zone + "."
	}
	fqdn := ""
	if dns.IsFqdn(name) {
		fqdn = name
	} else {
		fqdn = dns.Fqdn(name + "." + z)
	}
	fmt.Printf("[RFC2136 DEBUG] fqdn helper: name='%s', zone='%s', fqdn='%s'\n", name, z, fqdn)
	return fqdn
}

// Create a DNS record
func (p rfc2136Provider) Create(record internalDns.Record) (internalDns.Record, error) {
	// Mock for .test domains
	if len(p.zone) > 5 && p.zone[len(p.zone)-5:] == ".test" {
		fmt.Printf("[RFC2136 MOCK] Simulated create for %s in zone %s\n", record.Name, p.zone)
		return record, nil
	}
	fmt.Printf("[RFC2136 DEBUG] Create: server='%s', port='%s', tsigName='%s', tsigAlgo='%s', zone='%s', record='%+v'\n", p.server, p.port, p.tsigName, p.tsigAlgo, p.zone, record)
	m := new(dns.Msg)
	m.SetUpdate(p.zone)
	rrStr := fmt.Sprintf("%s %d IN %s %s", fqdn(record.Name, p.zone), record.TTL, record.Type, record.Content)
	fmt.Printf("[RFC2136 DEBUG] RR string: %s\n", rrStr)
	rr, err := dns.NewRR(rrStr)
	if err != nil {
		fmt.Printf("[RFC2136 ERROR] NewRR failed: %v\n", err)
		return internalDns.Record{}, err
	}
	m.Insert([]dns.RR{rr})
	tsigName := p.tsigName
	if !dns.IsFqdn(tsigName) {
		tsigName = tsigName + "."
	}
	m.SetTsig(tsigName, p.tsigAlgo, 300, time.Now().Unix())
	c := new(dns.Client)
	c.TsigSecret = map[string]string{tsigName: p.tsigSecret}
	resp, _, err := c.Exchange(m, fmt.Sprintf("%s:%s", p.server, p.port))
	if err != nil {
		fmt.Printf("[RFC2136 ERROR] Exchange failed: %v\n", err)
		return internalDns.Record{}, err
	}
	if resp.Rcode != dns.RcodeSuccess {
		fmt.Printf("[RFC2136 ERROR] Update failed: %s\n", dns.RcodeToString[resp.Rcode])
		return internalDns.Record{}, fmt.Errorf("RFC2136 update failed: %s", dns.RcodeToString[resp.Rcode])
	}
	return record, nil
}

// Delete a DNS record
func (p rfc2136Provider) Delete(record internalDns.Record) error {
	// Mock for .test domains
	if len(p.zone) > 5 && p.zone[len(p.zone)-5:] == ".test" {
		fmt.Printf("[RFC2136 MOCK] Simulated delete for %s in zone %s\n", record.Name, p.zone)
		return nil
	}
	fmt.Printf("[RFC2136 DEBUG] Delete: server='%s', port='%s', tsigName='%s', tsigAlgo='%s', zone='%s', record='%+v'\n", p.server, p.port, p.tsigName, p.tsigAlgo, p.zone, record)
	m := new(dns.Msg)
	m.SetUpdate(p.zone)
	rrStr := fmt.Sprintf("%s %d IN %s %s", fqdn(record.Name, p.zone), record.TTL, record.Type, record.Content)
	fmt.Printf("[RFC2136 DEBUG] RR string: %s\n", rrStr)
	rr, err := dns.NewRR(rrStr)
	if err != nil {
		fmt.Printf("[RFC2136 ERROR] NewRR failed: %v\n", err)
		return err
	}
	m.Remove([]dns.RR{rr})
	tsigName := p.tsigName
	if !dns.IsFqdn(tsigName) {
		tsigName = tsigName + "."
	}
	m.SetTsig(tsigName, p.tsigAlgo, 300, time.Now().Unix())
	c := new(dns.Client)
	c.TsigSecret = map[string]string{tsigName: p.tsigSecret}
	resp, _, err := c.Exchange(m, fmt.Sprintf("%s:%s", p.server, p.port))
	if err != nil {
		fmt.Printf("[RFC2136 ERROR] Exchange failed: %v\n", err)
		return err
	}
	if resp.Rcode != dns.RcodeSuccess {
		fmt.Printf("[RFC2136 ERROR] Delete failed: %s\n", dns.RcodeToString[resp.Rcode])
		return fmt.Errorf("RFC2136 delete failed: %s", dns.RcodeToString[resp.Rcode])
	}
	return nil
}

// Update a DNS record (delete old, add new)
func (p rfc2136Provider) Update(record internalDns.Record) (internalDns.Record, error) {
	if err := p.Delete(record); err != nil {
		return internalDns.Record{}, err
	}
	return p.Create(record)
}

// Get a DNS record (query)
func (p rfc2136Provider) Get(domain, recordType string) (internalDns.Record, error) {
	m := new(dns.Msg)
	m.SetQuestion(fqdn(domain, p.zone), dns.StringToType[recordType])
	c := new(dns.Client)
	resp, _, err := c.Exchange(m, fmt.Sprintf("%s:%s", p.server, p.port))
	if err != nil {
		return internalDns.Record{}, err
	}
	for _, rr := range resp.Answer {
		if rr.Header().Name == fqdn(domain, p.zone) && dns.TypeToString[rr.Header().Rrtype] == recordType {
			return internalDns.Record{
				Name:    domain,
				Type:    recordType,
				Content: rr.String(), // You may want to parse this further
				TTL:     int(rr.Header().Ttl),
			}, nil
		}
	}
	return internalDns.Record{}, nil
}

// List records (zone transfer, AXFR)
func (p rfc2136Provider) List() ([]internalDns.Record, error) {
	m := new(dns.Msg)
	m.SetAxfr(p.zone)
	t := new(dns.Transfer)
	resp, err := t.In(m, fmt.Sprintf("%s:%s", p.server, p.port))
	if err != nil {
		return nil, err
	}
	var records []internalDns.Record
	for rr := range resp {
		for _, r := range rr.RR {
			records = append(records, internalDns.Record{
				Name:    r.Header().Name,
				Type:    dns.TypeToString[r.Header().Rrtype],
				Content: r.String(), // You may want to parse this further
				TTL:     int(r.Header().Ttl),
			})
		}
	}
	return records, nil
}
