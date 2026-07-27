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
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func (h Handler) filterDockerLabels() ([]config.DomainRecord, error) {
	filterArgs := client.Filters{}
	filterArgs.Add("label", constants.DockdnsNameLabel)
	result, err := h.dockerCli.ContainerList(context.Background(), client.ContainerListOptions{
		Filters: filterArgs,
	})
	if err != nil {
		return nil, err
	}

	return parseContainerLabels(result.Items)
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

	// The label-tagged fields live on the embedded DomainRecordBase, so reflect
	// over that struct directly.
	if err := parseLabeledFields(&targetStruct.DomainRecordBase, containerLabels); err != nil {
		return err
	}

	// Parse provider-specific overrides (e.g., dockdns.overrides.technitium.a, dockdns.overrides.cloudflare.proxied)
	parseProviderOverrides(containerLabels, targetStruct)

	return nil
}

// parseLabeledFields populates the `label`-tagged fields of the target struct
// from the given container labels.
func parseLabeledFields(targetStruct *config.DomainRecordBase, containerLabels map[string]string) error {
	targetValue := reflect.ValueOf(targetStruct)
	if targetValue.Kind() != reflect.Pointer || targetValue.Elem().Kind() != reflect.Struct {
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

	return nil
}

// parseProviderOverrides extracts provider/zone-specific overrides from container labels.
//
// Label format: "dockdns.overrides.<zone-id>.<field>=value"
// Examples:
//   - dockdns.overrides.cloudflare_prod.a=10.0.0.5
//   - dockdns.overrides.technitium_internal.cname=internal.example.com
//   - dockdns.overrides.zone1.ttl=600
//
// The zone ID is the value of the zone's 'id' field in config. Docker label
// overrides require IDs with only letters, numbers, and underscores.
// Each override is stored as a config.DomainRecordBase in record.Overrides[zoneID],
// reusing the same field shape as the base record.
func parseProviderOverrides(labels map[string]string, record *config.DomainRecord) {
	const overridePrefix = "dockdns.overrides."

	setOverride := func(zoneID string, mutate func(*config.DomainRecordBase)) {
		if record.Overrides == nil {
			record.Overrides = make(map[string]config.DomainRecordBase)
		}
		base := record.Overrides[zoneID]
		mutate(&base)
		record.Overrides[zoneID] = base
	}

	for label, value := range labels {
		if !strings.HasPrefix(label, overridePrefix) {
			continue
		}

		// Remove "dockdns.overrides." prefix and split into <zone-id>.<field>.
		rest := strings.TrimPrefix(label, overridePrefix)
		parts := strings.Split(rest, ".")
		if len(parts) != 2 {
			slog.Warn("invalid override label format", "label", label)
			continue
		}

		zoneID := parts[0]
		field := parts[1]

		if !config.IsValidOverrideZoneID(zoneID) {
			slog.Warn("invalid override zone id", "label", label, "zoneID", zoneID)
			continue
		}

		// Empty values are treated as unset, so overrides inherit the base value.
		if value == "" {
			continue
		}

		// Process the override based on field type
		switch field {
		case "a":
			setOverride(zoneID, func(b *config.DomainRecordBase) { b.IP4 = value })

		case "aaaa":
			setOverride(zoneID, func(b *config.DomainRecordBase) { b.IP6 = value })

		case "cname":
			setOverride(zoneID, func(b *config.DomainRecordBase) { b.CName = value })

		case "ttl":
			ttlValue, err := strconv.Atoi(value)
			if err != nil {
				slog.Warn("invalid integer value for ttl override", "label", label, "value", value)
				continue
			}
			setOverride(zoneID, func(b *config.DomainRecordBase) { b.TTL = ttlValue })

		case "proxied":
			boolValue, err := strconv.ParseBool(value)
			if err != nil {
				slog.Warn("invalid boolean value for proxied override", "label", label, "value", value)
				continue
			}
			setOverride(zoneID, func(b *config.DomainRecordBase) { b.Proxied = &boolValue })

		case "comment":
			setOverride(zoneID, func(b *config.DomainRecordBase) { b.Comment = value })

		default:
			slog.Warn("unknown override field", "label", label, "field", field)
		}
	}
}

func setFieldValue(field reflect.Value, labelValue string) error {
	if field.Kind() == reflect.Pointer {
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
