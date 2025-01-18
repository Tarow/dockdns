package schedule

import (
	"context"
	"log/slog"

	"github.com/Tarow/dockdns/internal/constants"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type DockerEventTrigger struct {
	client *client.Client
}

func NewDockerEventTrigger(dockerCli *client.Client) *DockerEventTrigger {
	return &DockerEventTrigger{
		client: dockerCli,
	}
}

func (d *DockerEventTrigger) Start(ctx context.Context, eventChan chan<- TriggerEvent) {
	filterArgs := filters.NewArgs(
		filters.Arg("type", "container"),
		filters.Arg("label", constants.DockdnsNameLabel),
	)
	containerEventTypes := []string{"start", "stop", "die"}

	for _, cet := range containerEventTypes {
		filterArgs.Add("event", cet)
	}

	events, errs := d.client.Events(ctx, events.ListOptions{Filters: filterArgs})

	for {
		select {
		case <-ctx.Done():
			slog.Debug("DockerEventTrigger received stop signal")
			return
		case <-events:
			eventChan <- TriggerEvent{
				Name: "DockerEventTrigger",
			}
		case err := <-errs:
			slog.Warn("Error listening to Docker events", "err", err)
		}
	}
}

func (d *DockerEventTrigger) Reset() {
	// No-op for Docker events
}
