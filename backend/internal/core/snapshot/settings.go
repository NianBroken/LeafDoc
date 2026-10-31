// 本文件保存快照调度设置，校验间隔、每日时间、时区和保留数量，
// 持久化成功后通过回调把设置交给运行中的管理器。
package snapshot

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/defaults"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
)

// 调度只接受间隔和每日两种模式，启用的间隔至少一分钟。
// 保留数为零表示不按数量清理，上限只在 ValidateSettings 中检查。
const (
	scheduleModeInterval = "interval"
	scheduleModeDaily    = "daily"
	maxRetentionCount    = 100000
	minSnapshotInterval  = time.Minute
)

// Settings 是持久化与接口共用的调度值。不同模式使用不同字段，
// Enabled 控制周期计时，RunOnStartup 由调度器单独判断。
type Settings struct {
	Enabled        bool   `json:"enabled"`
	RunOnStartup   bool   `json:"runOnStartup"`
	Mode           string `json:"mode"`
	Interval       string `json:"interval"`
	Hour           int    `json:"hour"`
	Minute         int    `json:"minute"`
	TimeZone       string `json:"timeZone"`
	RetentionCount int    `json:"retentionCount"`
}

// SettingsStore 包装通用设置存储，自己的锁只保护应用回调的读写。
type SettingsStore struct {
	store *settings.Store[Settings]
	mu    sync.RWMutex
	apply func(Settings)
}

// DefaultSettings 使用统一默认模式和时区生成初值。负间隔归零，正值不足一分钟
// 提升至一分钟；负保留数归零，不在此限制保留数上限。启动时运行标记固定为真。
func DefaultSettings(interval time.Duration, retentionCount int) Settings {
	if interval < 0 {
		interval = 0
	} else if interval > 0 && interval < minSnapshotInterval {
		interval = minSnapshotInterval
	}
	if retentionCount < 0 {
		retentionCount = 0
	}
	return Settings{
		Enabled:        interval > 0,
		RunOnStartup:   true,
		Mode:           defaults.SnapshotMode,
		Interval:       interval.String(),
		Hour:           0,
		TimeZone:       defaults.SnapshotTimeZone,
		RetentionCount: retentionCount,
	}
}

// NewSettingsStore 加载持久化设置，并校验加载结果。读取或校验失败均返回错误，
// 不把无效记录直接替换为默认值；名称中的 .json 是通用设置存储的键。
func NewSettingsStore(dataDir string, interval time.Duration, retentionCount int) (*SettingsStore, error) {
	initial := DefaultSettings(interval, retentionCount)
	store, err := settings.New(dataDir, "snapshot-settings.json", 0o600, initial, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot settings: %w", err)
	}
	value := store.Get()
	if err := ValidateSettings(value); err != nil {
		return nil, fmt.Errorf("invalid snapshot settings: %w", err)
	}
	return &SettingsStore{store: store}, nil
}

// Get 返回当前设置值的副本，不触发应用回调。
func (s *SettingsStore) Get() Settings {
	return s.store.Get()
}

// Put 先校验，再写入通用存储，只有写入成功才调用运行设置回调。
func (s *SettingsStore) Put(value Settings) error {
	if err := ValidateSettings(value); err != nil {
		return err
	}
	if err := s.store.Put(value); err != nil {
		return err
	}
	s.applySettings(value)
	return nil
}

// Reload 先重新读取，再校验并应用。校验失败时不调用回调，
// 也不在此把通用存储已经读入的值恢复成此前值。
func (s *SettingsStore) Reload() error {
	if err := s.store.Reload(); err != nil {
		return err
	}
	if err := ValidateSettings(s.store.Get()); err != nil {
		return err
	}
	s.applySettings(s.store.Get())
	return nil
}

// SetApply 保存回调后立即用当前设置调用应用入口，空回调则不执行应用动作。
func (s *SettingsStore) SetApply(apply func(Settings)) {
	s.mu.Lock()
	s.apply = apply
	s.mu.Unlock()
	s.applySettings(s.store.Get())
}

// applySettings 只在读取回调指针时持锁，回调执行期间不占用该锁。
func (s *SettingsStore) applySettings(value Settings) {
	s.mu.RLock()
	apply := s.apply
	s.mu.RUnlock()
	if apply != nil {
		apply(value)
	}
}

// ValidateSettings 总是检查模式和保留数。间隔仅在已启用的间隔模式校验，
// 每日模式无论是否启用都检查时分和明确时区，不接受空值或 Local。
func ValidateSettings(value Settings) error {
	if value.Mode != scheduleModeInterval && value.Mode != scheduleModeDaily {
		return fmt.Errorf("schedule mode must be interval or daily")
	}
	if value.RetentionCount < 0 || value.RetentionCount > maxRetentionCount {
		return fmt.Errorf("retention count must be between 0 and %d", maxRetentionCount)
	}
	// 间隔校验先裁空白，但不把裁剪后的文本写回设置。
	if value.Enabled && value.Mode == scheduleModeInterval {
		interval, err := time.ParseDuration(strings.TrimSpace(value.Interval))
		if err != nil || interval < minSnapshotInterval {
			return sharederrors.NewLocalizedError("snapshot_invalid_interval", "Invalid backup interval", "interval must be at least %s", nil, minSnapshotInterval.String())
		}
	}
	if value.Mode == scheduleModeDaily {
		if value.Hour < 0 || value.Hour > 23 || value.Minute < 0 || value.Minute > 59 {
			return fmt.Errorf("daily schedule time is invalid")
		}
		if strings.TrimSpace(value.TimeZone) == "" || strings.TrimSpace(value.TimeZone) == "Local" {
			return fmt.Errorf("time zone is required")
		}
		// 时区加载使用原值，边缘空白不会因前面的空值检查而被自动移除。
		if _, err := time.LoadLocation(value.TimeZone); err != nil {
			return sharederrors.NewLocalizedError("snapshot_invalid_time_zone", "Invalid IANA time zone", "invalid time zone %s", nil, value.TimeZone)
		}
	}
	return nil
}
