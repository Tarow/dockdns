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

func (s *Scheduler) Start(ctx context.Context, debounceInterval time.Duration, runAtStart bool) {
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

	var lastEvent *TriggerEvent
	for {
		select {
		case e := <-eventChan:
			lastEvent = &e
			slog.Debug("Received event, resetting debounce timer", "name", e.Name)
			debounceTimer.Reset(debounceInterval)

		case <-debounceTimer.C:
			// lastEvent == nil -> No trigger, initial timer expired. Only run if runAtStart is true and there was no initial run yet.
			if lastEvent == nil {
				if !initRunDone {
					slog.Debug("Performing initial DNS update after startup")
					s.executeTask()
					initRunDone = true
				}
				continue
			}

			slog.Debug("Scheduler received trigger event", "name", lastEvent.Name)
			// Reset triggers, so that interval timer resets after a docker event triggered the task
			for _, trigger := range s.triggers {
				trigger.Reset()
			}
			s.executeTask()

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
