package dns

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/Tarow/dockdns/internal/constants"
	"github.com/docker/docker/api/types"
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

func parseContainerLabels(containers []types.Container) ([]config.DomainRecord, error) {
	var labelRecords []config.DomainRecord

	for _, container := range containers {
		var record config.DomainRecord
		err := parseLabels(container, &record)
		if err != nil {
			slog.Warn("error parsing label configuration, skipping container", "container", container.Names, "error", err)
			continue
		}

		labelRecords = append(labelRecords, record)
	}

	return labelRecords, nil
}

func parseLabels(container types.Container, targetStruct *config.DomainRecord) error {
	containerLabels := container.Labels
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

	return nil
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
