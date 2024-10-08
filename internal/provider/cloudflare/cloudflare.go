package cloudflare

import (
	"context"
	"slices"

	"github.com/Tarow/dockdns/internal/constants"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/cloudflare/cloudflare-go"
)

type cloudflareProvider struct {
	apiToken string
	zoneID   *cloudflare.ResourceContainer
	api      *cloudflare.API
}

func New(apiToken, zoneId string) (cloudflareProvider, error) {
	api, err := cloudflare.NewWithAPIToken(apiToken)
	if err != nil {
		return cloudflareProvider{}, err
	}

	return cloudflareProvider{
		apiToken: apiToken,
		zoneID: &cloudflare.ResourceContainer{
			Type:       "zone",
			Identifier: zoneId,
		},
		api: api,
	}, nil
}

func (cfp cloudflareProvider) List() ([]dns.Record, error) {
	ip4Records, err := cfp.list(constants.RecordTypeA)
	if err != nil {
		return nil, err
	}

	ip6Records, err := cfp.list(constants.RecordTypeAAAA)
	if err != nil {
		return nil, err
	}

	cnameRecords, err := cfp.list(constants.RecordTypeCNAME)
	if err != nil {
		return nil, err
	}

	return slices.Concat(ip4Records, ip6Records, cnameRecords), nil
}

func (cfp cloudflareProvider) list(recordType string) ([]dns.Record, error) {
	var allRecords []cloudflare.DNSRecord

	page := 1
	perPage := 100
	for {
		records, resultInfo, err := cfp.api.ListDNSRecords(context.Background(), cfp.zoneID, cloudflare.ListDNSRecordsParams{
			Type: recordType,
			ResultInfo: cloudflare.ResultInfo{
				Page:    page,
				PerPage: perPage,
			},
		})
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, records...)
		if resultInfo.Page == resultInfo.TotalPages {
			return mapRecords(allRecords), nil
		}

		page++
	}
}

func (cfp cloudflareProvider) Get(domain, recordType string) (dns.Record, error) {
	records, _, err := cfp.api.ListDNSRecords(context.Background(), cfp.zoneID, cloudflare.ListDNSRecordsParams{
		Type: recordType,
		Name: domain,
	})
	if err != nil {
		return dns.Record{}, err
	}
	if len(records) == 0 {
		return dns.Record{}, nil
	}
	return mapRecord(records[0]), nil
}

func (cfp cloudflareProvider) Create(record dns.Record) (dns.Record, error) {
	createdRecord, err := cfp.api.CreateDNSRecord(context.Background(), cfp.zoneID, cloudflare.CreateDNSRecordParams{
		Name:    record.Name,
		Type:    record.Type,
		Proxied: &record.Proxied,
		TTL:     int(record.TTL),
		Content: record.Content,
	})

	if err != nil {
		return dns.Record{}, err
	}
	return mapRecord(createdRecord), nil
}

func (cfp cloudflareProvider) Update(record dns.Record) (dns.Record, error) {
	updatedRecord, err := cfp.api.UpdateDNSRecord(context.Background(), cfp.zoneID, cloudflare.UpdateDNSRecordParams{
		ID:      record.ID,
		Type:    record.Type,
		Proxied: &record.Proxied,
		TTL:     record.TTL,
		Content: record.Content,
	})

	if err != nil {
		return dns.Record{}, err
	}
	return mapRecord(updatedRecord), nil
}

func (cfp cloudflareProvider) Delete(record dns.Record) error {
	return cfp.api.DeleteDNSRecord(context.Background(), cfp.zoneID, record.ID)
}

func mapRecords(records []cloudflare.DNSRecord) []dns.Record {
	var mappedRecords []dns.Record

	for _, record := range records {
		mappedRecords = append(mappedRecords, mapRecord(record))
	}

	return mappedRecords
}

func mapRecord(r cloudflare.DNSRecord) dns.Record {
	var proxied bool
	if r.Proxied != nil {
		proxied = *r.Proxied
	}
	return dns.Record{
		ID:      r.ID,
		Name:    r.Name,
		Type:    r.Type,
		Content: r.Content,
		Proxied: proxied,
		TTL:     r.TTL,
	}
}
