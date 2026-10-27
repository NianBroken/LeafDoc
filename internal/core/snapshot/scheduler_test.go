package snapshot

import (
	"testing"
	"time"
)

func waitForSnapshotSuccess(t *testing.T, manager *Manager, timeout time.Duration) time.Time {
	t.Helper()
	deadline := time.After(timeout)
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			last := manager.Status().LastSnapshotAt
			if last != nil && !last.IsZero() {
				return *last
			}
		case <-deadline:
			t.Fatal("timeout waiting for snapshot")
		}
	}
}

func TestScheduler_RunsOnStartAndSupportsManualTrigger(t *testing.T) {
	config := newTestConfig(t)
	config.Interval = 10 * time.Minute
	manager := NewManager(config)
	scheduler := NewScheduler(manager)
	defer scheduler.Stop()

	initial := waitForSnapshotSuccess(t, manager, 2*time.Second)
	if !scheduler.TriggerNow() {
		t.Fatal("manual trigger was not queued")
	}
	deadline := time.After(2 * time.Second)
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if last := manager.Status().LastSnapshotAt; last != nil && last.After(initial) {
				return
			}
		case <-deadline:
			t.Fatal("timeout waiting for manual trigger")
		}
	}
}

func TestScheduler_DisabledPlanSkipsStartupAndKeepsManualTrigger(t *testing.T) {
	manager := NewManager(newTestConfig(t))
	manager.UpdateSettings(Settings{
		Enabled:        false,
		RunOnStartup:   false,
		Mode:           scheduleModeInterval,
		Interval:       "24h",
		TimeZone:       "UTC",
		RetentionCount: 2,
	})
	scheduler := NewScheduler(manager)
	defer scheduler.Stop()

	time.Sleep(100 * time.Millisecond)
	if manager.Status().LastSnapshotAt != nil {
		t.Fatal("disabled plan ran a startup snapshot")
	}
	if !scheduler.TriggerNow() {
		t.Fatal("manual backup was unavailable while the plan was disabled")
	}
	waitForSnapshotSuccess(t, manager, 2*time.Second)
}

func TestScheduler_Stop(t *testing.T) {
	manager := NewManager(newTestConfig(t))
	scheduler := NewScheduler(manager)
	scheduler.Stop()
	scheduler.Stop()
}

func TestNextScheduledRun_UsesDailyTimeZone(t *testing.T) {
	now := time.Date(2026, time.October, 27, 16, 30, 0, 0, time.UTC)
	settings := Settings{Enabled: true, Mode: scheduleModeDaily, Hour: 2, Minute: 0, TimeZone: "Asia/Shanghai"}
	got, err := nextScheduledRun(now, settings)
	if err != nil {
		t.Fatalf("nextScheduledRun() error = %v", err)
	}
	want := time.Date(2026, time.October, 27, 18, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("nextScheduledRun() = %s, want %s", got, want)
	}
}

func TestNextScheduledRun_AtScheduledTimeMovesToFollowingDay(t *testing.T) {
	location, err := time.LoadLocation("UTC")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.October, 27, 2, 0, 0, 0, location)
	settings := Settings{Enabled: true, Mode: scheduleModeDaily, Hour: 2, Minute: 0, TimeZone: "UTC"}
	got, err := nextScheduledRun(now, settings)
	if err != nil {
		t.Fatalf("nextScheduledRun() error = %v", err)
	}
	want := time.Date(2026, time.October, 28, 2, 0, 0, 0, location)
	if !got.Equal(want) {
		t.Fatalf("nextScheduledRun() = %s, want %s", got, want)
	}
}
