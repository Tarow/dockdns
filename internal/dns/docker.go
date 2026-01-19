package dns

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
	"strings"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

func (h Handler) filterDockerLabels() ([]config.DomainRecord, error) {
	containers, err := h.dockerCli.ContainerList(context.Background(), container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", constants.DockdnsNameLabel)),
	})
	if err != nil {
		return nil, err
	}

	return parseContainerLabels(containers)
}

func parseContainerLabels(containers []container.Summary) ([]config.DomainRecord, error) {
	var labelRecords []config.DomainRecord

	for _, ctr := range containers {
		var record config.DomainRecord
		err := parseLabels(ctr, &record)
		if err != nil {
			slog.Warn("error parsing label configuration, skipping container", "container", ctr.Names, "error", err)
			continue
		}

		// Set container metadata for tracking record origin
		record.Source = "docker"
		record.ContainerID = getShortContainerID(ctr.ID)
		if len(ctr.Names) > 0 {
			// Container names start with '/', remove it
			record.ContainerName = strings.TrimPrefix(ctr.Names[0], "/")
		}

		// Name label can have multiple comma separated domains. Create a record for all of them
		domains := strings.Split(record.Name, ",")
		for _, domain := range domains {
			r := record
			r.Name = domain
			labelRecords = append(labelRecords, r)
		}
	}

	return labelRecords, nil
}

// getShortContainerID returns the first 12 characters of a container ID
func getShortContainerID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func parseLabels(ctr container.Summary, targetStruct *config.DomainRecord) error {
	containerLabels := ctr.Labels
	targetValue := reflect.ValueOf(targetStruct)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("targetStruct must be a pointer to a struct")
	}

	targetType := targetValue.Elem().Type()

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		label := field.Tag.Get("label")

		if label != "" {
			labelValue, exists := containerLabels[label]
			if exists && labelValue != "" {
				targetField := targetValue.Elem().Field(i)
				if err := setFieldValue(targetField, labelValue); err != nil {
					return fmt.Errorf("could not parse label value, label: %v, value: %v, error: %w", label, labelValue, err)
				}
			}
		}
	}

	// Parse provider-specific overrides (e.g., dockdns.cname.technitium, dockdns.proxied.cloudflare)
	parseProviderOverrides(containerLabels, targetStruct)

	return nil
}

// parseProviderOverrides extracts provider/zone-specific overrides from container labels.
// 
// Label format: "dockdns.<zone-id>.<field>=value"
// Examples: 
//   - dockdns.cloudflare-prod.a=10.0.0.5
//   - dockdns.technitium-internal.cname=internal.example.com
//   - dockdns.zone1.ttl=600
//
// The zone ID is the value of the zone's 'id' field in config, or the zone name if 'id' is not set.
func parseProviderOverrides(labels map[string]string, record *config.DomainRecord) {
	const dockdnsPrefix = "dockdns."
	
	for label, value := range labels {
		if !strings.HasPrefix(label, dockdnsPrefix) {
			continue
		}
		
		// Remove "dockdns." prefix
		rest := strings.TrimPrefix(label, dockdnsPrefix)
		
		// Split by dots to get parts: dockdns.<zone-id>.<field>
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) != 2 {
			// Not an override label (could be dockdns.name, dockdns.a, etc.)
			continue
		}
		
		zoneID := parts[0]
		field := parts[1]
		
		// Skip if zoneID or value is empty
		if zoneID == "" || value == "" {
			continue
		}
		
		// Process the override based on field type
		switch field {
		case "a":
			if record.IP4Overrides == nil {
				record.IP4Overrides = make(map[string]string)
			}
			record.IP4Overrides[zoneID] = value
			
		case "aaaa":
			if record.IP6Overrides == nil {
				record.IP6Overrides = make(map[string]string)
			}
			record.IP6Overrides[zoneID] = value
			
		case "cname":
			if record.CNameOverrides == nil {
				record.CNameOverrides = make(map[string]string)
			}
			record.CNameOverrides[zoneID] = value
			
		case "ttl":
			ttlValue, err := strconv.Atoi(value)
			if err != nil {
				slog.Warn("invalid integer value for ttl override", "label", label, "value", value)
				continue
			}
			if record.TTLOverrides == nil {
				record.TTLOverrides = make(map[string]int)
			}
			record.TTLOverrides[zoneID] = ttlValue
			
		case "proxied":
			boolValue, err := strconv.ParseBool(value)
			if err != nil {
				slog.Warn("invalid boolean value for proxied override", "label", label, "value", value)
				continue
			}
			if record.ProxiedOverrides == nil {
				record.ProxiedOverrides = make(map[string]bool)
			}
			record.ProxiedOverrides[zoneID] = boolValue
			
		case "comment":
			if record.CommentOverrides == nil {
				record.CommentOverrides = make(map[string]string)
			}
			record.CommentOverrides[zoneID] = value
		}
	}
}

func setFieldValue(field reflect.Value, labelValue string) error {
	if field.Kind() == reflect.Ptr {
		// If the field is a pointer, create a new instance of the underlying type and set the value
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(labelValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(labelValue, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(labelValue)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Uint8:
		byteValue := []byte(labelValue)
		field.SetBytes(byteValue)

	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}

	return nil
}
