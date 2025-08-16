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
	protocol   string
	tsigName   string
	tsigSecret string
	tsigAlgo   string
	zone       string
}

func New(server, port, protocol, tsigName, tsigSecret, tsigAlgo, zone string) rfc2136Provider {
	return rfc2136Provider{
		server:     server,
		port:       port,
		protocol:   protocol,
		tsigName:   tsigName,
		tsigSecret: tsigSecret,
		tsigAlgo:   tsigAlgo,
		zone:       zone,
	}
}

// Helper to get fqdn
func fqdn(name, zone string) string {
	if dns.IsFqdn(name) {
		return name
	}
	return dns.Fqdn(name + "." + zone)
}

// Create a DNS record
func (p rfc2136Provider) Create(record internalDns.Record) (internalDns.Record, error) {
	m := new(dns.Msg)
	m.SetUpdate(p.zone)
	rr, err := dns.NewRR(fmt.Sprintf("%s %d IN %s %s", fqdn(record.Name, p.zone), record.TTL, record.Type, record.Content))
	if err != nil {
		return internalDns.Record{}, err
	}
	m.Insert([]dns.RR{rr})
	m.SetTsig(p.tsigName, p.tsigAlgo, 300, time.Now().Unix())
	c := new(dns.Client)
	c.Net = p.protocol
	c.TsigSecret = map[string]string{p.tsigName: p.tsigSecret}
	resp, _, err := c.Exchange(m, fmt.Sprintf("%s:%s", p.server, p.port))
	if err != nil {
		return internalDns.Record{}, err
	}
	if resp.Rcode != dns.RcodeSuccess {
		return internalDns.Record{}, fmt.Errorf("RFC2136 update failed: %s", dns.RcodeToString[resp.Rcode])
	}
	return record, nil
}

// Delete a DNS record
func (p rfc2136Provider) Delete(record internalDns.Record) error {
	m := new(dns.Msg)
	m.SetUpdate(p.zone)
	rr, err := dns.NewRR(fmt.Sprintf("%s %d IN %s %s", fqdn(record.Name, p.zone), record.TTL, record.Type, record.Content))
	if err != nil {
		return err
	}
	m.Remove([]dns.RR{rr})
	m.SetTsig(p.tsigName, p.tsigAlgo, 300, time.Now().Unix())
	c := new(dns.Client)
	c.Net = p.protocol
	c.TsigSecret = map[string]string{p.tsigName: p.tsigSecret}
	resp, _, err := c.Exchange(m, fmt.Sprintf("%s:%s", p.server, p.port))
	if err != nil {
		return err
	}
	if resp.Rcode != dns.RcodeSuccess {
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
	c.Net = p.protocol
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
