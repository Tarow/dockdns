package dns

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Tarow/dockdns/internal/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

const domainLabel = "dockdns.domain"

func (h handler) filterDockerLabels() ([]config.DomainRecord, error) {
	containers, err := h.dockerCli.ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", domainLabel)),
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
			return nil, err
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
			if exists {
				targetField := targetValue.Elem().Field(i)
				targetField.SetString(labelValue)
			}
		}
	}

	return nil
}
