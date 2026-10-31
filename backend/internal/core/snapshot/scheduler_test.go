// 本文件使用固定时刻验证每日调度，无需等待真实计时器或执行归档。
package snapshot

import (
	"testing"
	"time"
)

// UTC 十六点三十分对应上海次日零点三十分，下一次上海两点对应 UTC 十八点。
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

// 当前时间恰好等于每日调度时间时，返回下一日而不是立即再次执行。
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

// 上海午夜调度分别覆盖午夜之前及恰好午夜，后一种输入必须落到再下一个午夜。
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
