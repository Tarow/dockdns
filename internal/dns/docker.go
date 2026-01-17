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

const (
	cnameOverridePrefix   = "dockdns.cname."
	proxiedOverridePrefix = "dockdns.proxied."
)

// parseProviderOverrides extracts provider/zone-specific overrides from container labels.
// Labels like "dockdns.cname.<zone-id>=internal.example.com" override the CNAME for that zone.
// Labels like "dockdns.proxied.<zone-id>=true" override the proxied setting for that zone.
// The zone ID is the value of the zone's 'id' field in config, or the zone name if 'id' is not set.
func parseProviderOverrides(labels map[string]string, record *config.DomainRecord) {
	for label, value := range labels {
		// Parse CNAME overrides (e.g., dockdns.cname.technitium-internal=internal.example.com)
		if strings.HasPrefix(label, cnameOverridePrefix) && label != "dockdns.cname" {
			zoneID := strings.TrimPrefix(label, cnameOverridePrefix)
			if zoneID != "" && value != "" {
				if record.CNameOverrides == nil {
					record.CNameOverrides = make(map[string]string)
				}
				record.CNameOverrides[zoneID] = value
			}
		}

		// Parse Proxied overrides (e.g., dockdns.proxied.cloudflare-prod=true)
		if strings.HasPrefix(label, proxiedOverridePrefix) && label != "dockdns.proxied" {
			zoneID := strings.TrimPrefix(label, proxiedOverridePrefix)
			if zoneID != "" && value != "" {
				boolValue, err := strconv.ParseBool(value)
				if err != nil {
					slog.Warn("invalid boolean value for proxied override", "label", label, "value", value)
					continue
				}
				if record.ProxiedOverrides == nil {
					record.ProxiedOverrides = make(map[string]bool)
				}
				record.ProxiedOverrides[zoneID] = boolValue
			}
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
