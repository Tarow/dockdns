package cloudflare

import (
	"context"
	"fmt"
	"slices"

	"github.com/Tarow/dockdns/internal/constants"
	"github.com/Tarow/dockdns/internal/dns"
	"github.com/cloudflare/cloudflare-go/v5"
	cfDns "github.com/cloudflare/cloudflare-go/v5/dns"
	"github.com/cloudflare/cloudflare-go/v5/option"
	"github.com/cloudflare/cloudflare-go/v5/zones"
)

type cloudflareProvider struct {
	apiToken string
	zoneID   string
	service  *cfDns.RecordService
}

func New(apiToken, zoneID string) (cloudflareProvider, error) {
	return cloudflareProvider{
		apiToken: apiToken,
		zoneID:   zoneID,
		service:  cfDns.NewRecordService(option.WithEnvironmentProduction(), option.WithAPIToken(apiToken)),
	}, nil
}

func FetchZoneID(apiToken string, domain string) (string, error) {
	service := zones.NewZoneService(option.WithEnvironmentProduction(), option.WithAPIToken(apiToken))
	zones := service.ListAutoPaging(context.Background(), zones.ZoneListParams{
		Name: cloudflare.F(domain),
	})
	for zones.Next() {
		if zones.Err() != nil {
			return "", zones.Err()
		}
		if zones.Current().Name == domain {
			return zones.Current().ID, nil
		}
	}
	return "", fmt.Errorf("no zone found for domain %s", domain)
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
	var allRecords []cfDns.RecordResponse

	records := cfp.service.ListAutoPaging(context.Background(), cfDns.RecordListParams{
		ZoneID:  cloudflare.F(cfp.zoneID),
		Type:    cloudflare.F(cfDns.RecordListParamsType(recordType)),
		PerPage: cloudflare.F(float64(100)),
	})

	for records.Next() {
		if records.Err() != nil {
			return nil, records.Err()
		}
		allRecords = append(allRecords, records.Current())
	}

	return mapRecords(allRecords), nil
}

func (cfp cloudflareProvider) Get(domain, recordType string) (dns.Record, error) {
	records := cfp.service.ListAutoPaging(context.Background(), cfDns.RecordListParams{
		ZoneID: cloudflare.F(cfp.zoneID),
		Type:   cloudflare.F(cfDns.RecordListParamsType(recordType)),
		Name: cloudflare.F(cfDns.RecordListParamsName{
			Exact: cloudflare.F(domain),
		}),
	})
	if records.Err() != nil {
		return dns.Record{}, records.Err()
	}
	if !records.Next() {
		return dns.Record{}, nil
	}
	return mapRecord(records.Current()), nil
}

func (cfp cloudflareProvider) Create(record dns.Record) (dns.Record, error) {
	createdRecord, err := cfp.service.New(context.Background(), cfDns.RecordNewParams{
		ZoneID: cloudflare.F(cfp.zoneID),
		Body: cfDns.RecordNewParamsBody{
			Name:    cloudflare.F(record.Name),
			Type:    cloudflare.F(cfDns.RecordNewParamsBodyType(record.Type)),
			Proxied: cloudflare.F(record.Proxied),
			TTL:     cloudflare.F(cfDns.TTL(record.TTL)),
			Content: cloudflare.F(record.Content),
			Comment: cloudflare.F(record.Comment),
		},
	})

	if err != nil {
		return dns.Record{}, err
	}
	return mapRecord(*createdRecord), nil
}

func (cfp cloudflareProvider) Update(record dns.Record) (dns.Record, error) {
	updatedRecord, err := cfp.service.Update(context.Background(), record.ID, cfDns.RecordUpdateParams{
		ZoneID: cloudflare.F(cfp.zoneID),
		Body: cfDns.RecordUpdateParamsBody{
			Name:    cloudflare.F(record.Name),
			Type:    cloudflare.F(cfDns.RecordUpdateParamsBodyType(record.Type)),
			Proxied: cloudflare.F(record.Proxied),
			TTL:     cloudflare.F(cfDns.TTL(record.TTL)),
			Content: cloudflare.F(record.Content),
			Comment: cloudflare.F(record.Comment),
		},
	})

	if err != nil {
		return dns.Record{}, err
	}
	return mapRecord(*updatedRecord), nil
}

func (cfp cloudflareProvider) Delete(record dns.Record) error {
	_, err := cfp.service.Delete(context.Background(), record.ID, cfDns.RecordDeleteParams{
		ZoneID: cloudflare.F(cfp.zoneID),
	})
	return err
}

func mapRecords(records []cfDns.RecordResponse) []dns.Record {
	var mappedRecords []dns.Record

	for _, record := range records {
		mappedRecords = append(mappedRecords, mapRecord(record))
	}

	return mappedRecords
}

func mapRecord(r cfDns.RecordResponse) dns.Record {
	return dns.Record{
		ID:      r.ID,
		Name:    r.Name,
		Type:    string(r.Type),
		Content: r.Content,
		Proxied: r.Proxied,
		TTL:     int(r.TTL),
		Comment: r.Comment,
	}
}
