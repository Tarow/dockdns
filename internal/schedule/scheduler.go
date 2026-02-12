package schedule

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type TriggerEvent struct {
	Name string
}

type Trigger interface {
	Start(ctx context.Context, eventChan chan<- TriggerEvent)
	Reset()
}

type Scheduler struct {
	triggers  []Trigger
	task      func()
	taskMutex sync.Mutex
}

func NewScheduler(task func()) *Scheduler {
	return &Scheduler{
		triggers: []Trigger{},
		task:     task,
	}
}

func (s *Scheduler) Register(trigger Trigger) {
	s.triggers = append(s.triggers, trigger)
}

func (s *Scheduler) Start(ctx context.Context, debounceInterval time.Duration, maxDebounceDuration time.Duration, runAtStart bool) {
	eventChan := make(chan TriggerEvent, 1)
	var wg sync.WaitGroup

	for _, trigger := range s.triggers {
		wg.Add(1)
		go func(t Trigger) {
			defer wg.Done()
			t.Start(ctx, eventChan)
		}(trigger)
	}

	initRunDone := !runAtStart

	// Use debounce logic, so multiple docker events etc. will just trigger one update run
	debounceTimer := time.NewTimer(debounceInterval)
	// Set the maximum time until a debounce can happen. Will be set at every start of a debounce interval
	maxDebounceTime := time.Time{}

	var lastEvent *TriggerEvent

	handleEvent := func(e TriggerEvent) {
		lastEvent = &e

		// If resetting the debounceTimer exceeds the maxDebounceDuration, only reset it to the point the maxDebounceDuration is reached
		if !maxDebounceTime.IsZero() && time.Now().Add(debounceInterval).After(maxDebounceTime) {
			slog.Debug("Received event, resetting would exceed maximum debounce time, won't reset beyond", "name", e.Name, "event", e)
			debounceTimer.Reset(time.Until(maxDebounceTime))
			return
		}

		slog.Debug("Received event, resetting debounce timer", "name", e.Name, "event", e)
		slog.Debug("TriggerEvent details", "event", e)
		debounceTimer.Reset(debounceInterval)

		// If we start debouncing, set the max deboune time
		if maxDebounceTime.IsZero() {
			maxDebounceTime = time.Now().Add(maxDebounceDuration)
		}
	}

	handleTrigger := func() {
		// lastEvent == nil -> No trigger, initial timer expired. Only run if runAtStart is true and there was no initial run yet.
		if lastEvent == nil {
			if !initRunDone {
				slog.Debug("Performing initial DNS update after startup")
				s.executeTask()
				initRunDone = true
			}
			return
		}

		slog.Debug("Received trigger event", "name", lastEvent.Name)
		// Update gets triggered, reset the max debounce time
		maxDebounceTime = time.Time{}

		// Reset triggers, so that interval timer resets after a docker event triggered the task
		for _, trigger := range s.triggers {
			trigger.Reset()
		}
		s.executeTask()
	}

	for {
		select {
		case e := <-eventChan:
			handleEvent(e)
		case <-debounceTimer.C:
			handleTrigger()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Scheduler) executeTask() {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()
	slog.Debug("Executing scheduled task")
	s.task()
}
