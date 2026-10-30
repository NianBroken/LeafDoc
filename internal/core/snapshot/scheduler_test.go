package snapshot

import (
	"testing"
	"time"
)

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

// Shanghai midnight is the next local calendar day, including exactly at midnight.
func TestNextScheduledShanghaiMidnight(t *testing.T) {
	settings := Settings{Enabled: true, Mode: scheduleModeDaily, Hour: 0, Minute: 0, TimeZone: "Asia/Shanghai"}
	for _, now := range []time.Time{time.Date(2026, 10, 30, 8, 0, 0, 0, time.UTC), time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC)} {
		got, err := nextScheduledRun(now, settings)
		if err != nil {
			t.Fatal(err)
		}
		want := time.Date(2026, 10, 30, 16, 0, 0, 0, time.UTC)
		if !want.After(now) {
			want = want.Add(24 * time.Hour)
		}
		if !got.Equal(want) {
			t.Fatal(got)
		}
	}
}
