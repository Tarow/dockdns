package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/Tarow/dockdns/internal/constants"
	"github.com/moby/moby/client"
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
	filterArgs := client.Filters{}
	filterArgs.Add("type", "container")
	filterArgs.Add("label", constants.DockdnsNameLabel)
	containerEventTypes := []string{"start", "stop", "die"}

	for _, cet := range containerEventTypes {
		filterArgs.Add("event", cet)
	}

	for {
		result := d.client.Events(ctx, client.EventsListOptions{Filters: filterArgs})

		for {
			select {
			case <-ctx.Done():
				slog.Debug("DockerEventTrigger received stop signal")
				return
			case _, ok := <-result.Messages:
				if !ok {
					goto reconnect
				} else {
					eventChan <- TriggerEvent{
						Name: "DockerEventTrigger",
					}
				}
			case err, ok := <-result.Err:
				if !ok {
					goto reconnect
				} else {
					slog.Warn("Error listening to Docker events", "err", err)
				}

			}
		}
	reconnect:
		slog.Warn("Docker event channel was closed. Trying to reconnect in 5 seconds")
		time.Sleep(5 * time.Second)
	}
}

func (d *DockerEventTrigger) Reset() {
	// No-op for Docker events
}
