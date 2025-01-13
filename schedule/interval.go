package schedule

import (
	"context"
	"log/slog"
	"time"
)

type IntervalTrigger struct {
	interval  time.Duration
	resetChan chan struct{}
}

func NewIntervalTrigger(interval time.Duration) *IntervalTrigger {
	return &IntervalTrigger{
		interval:  interval,
		resetChan: make(chan struct{}),
	}
}

func (i *IntervalTrigger) Start(ctx context.Context, eventChan chan<- TriggerEvent) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("IntervalTrigger received stop signal")
			return
		case <-i.resetChan:
			slog.Debug("Resetting interval timer")
			continue
		case <-time.After(i.interval):
			event := TriggerEvent{
				Name: "IntervalTrigger",
			}
			eventChan <- event
		}
	}
}

func (i *IntervalTrigger) Reset() {
	i.resetChan <- struct{}{}
}
