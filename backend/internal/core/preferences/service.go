// Package preferences 保存实例共用的界面语言、日期和时间格式，不按用户分别存储。
package preferences

import (
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/defaults"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
)

// Config 只包含三个显示选项，不保存账号资料或认证凭据。
type Config struct {
	Language   string `json:"language"`
	DateFormat string `json:"dateFormat"`
	TimeFormat string `json:"timeFormat"`
}

// Service 用互斥锁串行执行局部更新与重载，避免同一实例内读后写覆盖其他更新。
// 底层 Store 另行保护缓存读写，不提供跨进程的局部更新事务。
type Service struct {
	mu    sync.Mutex
	store *settings.Store[Config]
}

// Defaults 从共享初始值构造配置副本，不读取当前设备语言或日期区域设置。
func Defaults() Config {
	return Config{Language: defaults.Language, DateFormat: defaults.DateFormat, TimeFormat: defaults.TimeFormat}
}

// New 加载 preferences.json 数据库记录，校验后再次保存并发送设置变化通知。
// 记录含非法选项时构造失败，不用默认选项掩盖已存储的非法值。
func New(dir string) (*Service, error) {
	st, err := settings.New(dir, "preferences.json", 0600, Defaults(), nil)
	if err != nil {
		return nil, err
	}
	if err := validate(st.Get()); err != nil {
		return nil, err
	}
	if err := st.Put(st.Get()); err != nil {
		return nil, err
	}
	return &Service{store: st}, nil
}

// Get 返回当前配置副本，不读取单个用户的偏好。
func (s *Service) Get() Config { return s.store.Get() }

// Reload 持有服务锁重新加载、校验并保存记录，底层重载和保存各自发送通知。
// 校验失败不在本方法中恢复先前缓存，错误直接交给调用方。
func (s *Service) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Reload(); err != nil {
		return err
	}
	if err := validate(s.store.Get()); err != nil {
		return err
	}
	return s.store.Put(s.store.Get())
}

// Patch 只更新传入 map 中明确出现的三个合法字段，未传字段沿用当前值。
// 值必须为对应枚举字符串，未知字段立即报错，整个循环通过后才写数据库。
// map 遍历顺序不固定，失败时返回的配置副本包含此前已检查字段的内存赋值，
// 但这些部分赋值不会保存；写入错误也与候选配置一并返回。
func (s *Service) Patch(patch map[string]any) (Config, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.Get()
	for k, x := range patch {
		switch k {
		case "language":
			val, ok := x.(string)
			if !ok || !oneOf(val, "zh", "en", "de", "es", "fr") {
				return v, invalidChoice("invalid language")
			}
			v.Language = val
		case "dateFormat":
			val, ok := x.(string)
			if !ok || !oneOf(val, "iso", "dmy_dot", "mdy_slash", "dmy_slash") {
				return v, invalidChoice("invalid date format")
			}
			v.DateFormat = val
		case "timeFormat":
			val, ok := x.(string)
			if !ok || !oneOf(val, "24h", "12h") {
				return v, invalidChoice("invalid time format")
			}
			v.TimeFormat = val
		default:
			return v, invalidChoice("unknown preference")
		}
	}
	return v, s.store.Put(v)
}

// oneOf 按精确字符串比较检查允许列表，不忽略大小写或去掉空白。
func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}

// validate 按语言、日期、时间的顺序校验完整配置，返回遇到的第一个错误。
func validate(value Config) error {
	if !oneOf(value.Language, "zh", "en", "de", "es", "fr") {
		return invalidChoice("invalid language")
	}
	if !oneOf(value.DateFormat, "iso", "dmy_dot", "mdy_slash", "dmy_slash") {
		return invalidChoice("invalid date format")
	}
	if !oneOf(value.TimeFormat, "24h", "12h") {
		return invalidChoice("invalid time format")
	}
	return nil
}

// invalidChoice 为非法选项生成统一错误码，提示文本同时作为翻译模板。
// 不包装底层错误，使选项错误与实际存储失败保持不同来源。
func invalidChoice(message string) error {
	return sharederrors.NewLocalizedError("invalid_preferences", message, message, nil)
}
