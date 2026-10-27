package snapshot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "time/tzdata"
)

type Scheduler struct {
	manager   *Manager
	manual    chan struct{}
	updated   chan struct{}
	done      chan struct{}
	wg        sync.WaitGroup
	closeOnce sync.Once
}

func NewScheduler(manager *Manager) *Scheduler {
	s := &Scheduler{
		manager: manager,
		manual:  make(chan struct{}, 1),
		updated: make(chan struct{}, 1),
		done:    make(chan struct{}),
	}
	initialSettings := manager.Settings()
	if initialSettings.RunOnStartup {
		s.manual <- struct{}{}
	}
	s.wg.Add(1)
	go s.run()
	return s
}

func (s *Scheduler) run() {
	defer s.wg.Done()
	for {
		timer, timerC := s.nextTimer()
		select {
		case <-timerC:
			if err := s.manager.RunOnce(context.Background()); err != nil {
				slog.Error("snapshot failed", "error", err)
			}
		case <-s.manual:
			if err := s.manager.RunOnce(context.Background()); err != nil {
				slog.Error("snapshot failed", "error", err)
			}
		case <-s.updated:
		case <-s.done:
			stopTimer(timer)
			return
		}
		stopTimer(timer)
	}
}

func (s *Scheduler) nextTimer() (*time.Timer, <-chan time.Time) {
	settings := s.manager.Settings()
	if !settings.Enabled {
		return nil, nil
	}
	next, err := nextScheduledRun(time.Now(), settings)
	if err != nil {
		return nil, nil
	}
	delay := time.Until(next)
	if delay < 0 {
		delay = 0
	}
	timer := time.NewTimer(delay)
	return timer, timer.C
}

func nextScheduledRun(now time.Time, settings Settings) (time.Time, error) {
	var next time.Time
	switch settings.Mode {
	case scheduleModeDaily:
		location, err := time.LoadLocation(settings.TimeZone)
		if err != nil {
			return time.Time{}, err
		}
		localNow := now.In(location)
		next = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), settings.Hour, settings.Minute, 0, 0, location)
		if !next.After(localNow) {
			next = time.Date(localNow.Year(), localNow.Month(), localNow.Day()+1, settings.Hour, settings.Minute, 0, 0, location)
		}
	default:
		interval, err := time.ParseDuration(settings.Interval)
		if err != nil || interval < minSnapshotInterval {
			return time.Time{}, fmt.Errorf("invalid interval %q", settings.Interval)
		}
		return now.Add(interval), nil
	}
	return next, nil
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func (s *Scheduler) UpdateSettings() {
	select {
	case s.updated <- struct{}{}:
	default:
	}
}

func (s *Scheduler) TriggerNow() bool {
	select {
	case s.manual <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *Scheduler) Stop() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
	s.wg.Wait()
}
