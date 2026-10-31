// Package tocdisplay 管理实例目录显示模式，将设置保存在 MySQL 的 settings 记录中。
// 模式包括 auto、always 和 never，本包保存初始显示偏好，不执行页面渲染。
package tocdisplay

import (
	"encoding/json"
	"fmt"

	"github.com/NianBroken/LeafDoc/internal/core/defaults"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// fileConfig 是 toc-display.json 数据库记录的 JSON 结构，保留布尔字段供兼容读取。
type fileConfig struct {
	AlwaysShow bool   `json:"alwaysShow"`
	Mode       string `json:"mode,omitempty"`
}

// UnmarshalJSON 用指针区分字段缺失与显式布尔值，再确定模式。
// 初始采用共享默认模式，存在布尔值时映射为 auto 或 always，显式 mode 最后覆盖。
// 仅接受三种模式，AlwaysShow 字段保留传入布尔值，不强制与显式 mode 同步。
func (c *fileConfig) UnmarshalJSON(raw []byte) error {
	var value struct {
		AlwaysShow *bool   `json:"alwaysShow"`
		Mode       *string `json:"mode"`
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	c.Mode = defaults.TocMode
	c.AlwaysShow = false
	if value.AlwaysShow != nil {
		c.AlwaysShow = *value.AlwaysShow
		c.Mode = "auto"
		if c.AlwaysShow {
			c.Mode = "always"
		}
	}
	if value.Mode != nil {
		c.Mode = *value.Mode
	}
	if c.Mode != "auto" && c.Mode != "always" && c.Mode != "never" {
		return sharederrors.NewLocalizedError("toc_display_invalid_mode", "Invalid TOC mode", "invalid toc mode", nil)
	}
	return nil
}

// Service 持有带锁的配置存储，读取与写入通过通用 Store 完成。
type Service struct {
	store *settings.Store[fileConfig]
}

// New 加载 settings 命名空间中的 toc-display.json 记录，缺失时保存默认模式。
// storageDir 用于定位存储上下文，不会在该目录创建同名 JSON 文件。
func New(storageDir string) (*Service, error) {
	st, err := settings.New(storageDir, "toc-display.json", 0o600, fileConfig{Mode: defaults.TocMode}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read toc-display config: %w", err)
	}
	return &Service{store: st}, nil
}

// AlwaysShow 根据最终 Mode 是否为 always 判断，不直接读取存储的兼容布尔字段。
func (s *Service) AlwaysShow() bool {
	return s.Mode() == "always"
}

// SetAlwaysShow 保存仅含指定布尔值的结构，其 Mode 为零值空串。
// 当前缓存读取通过 Mode 的兼容分支映射为 always 或 auto，保存错误包装为本地化错误。
func (s *Service) SetAlwaysShow(alwaysShow bool) error {
	if err := s.store.Put(fileConfig{AlwaysShow: alwaysShow}); err != nil {
		return sharederrors.NewLocalizedError(
			"toc_display_update_failed",
			"Failed to update TOC display setting",
			"failed to persist toc-display config",
			err,
		)
	}
	return nil
}

// Reload 重新读取数据库记录并经自定义 JSON 解析确定模式，失败时添加上下文返回。
func (s *Service) Reload() error {
	if err := s.store.Reload(); err != nil {
		return fmt.Errorf("failed to reload toc-display config: %w", err)
	}
	return nil
}

// Mode 优先返回非空模式，模式为空时根据兼容布尔字段返回 always 或 auto。
func (s *Service) Mode() string {
	value := s.store.Get()
	if value.Mode != "" {
		return value.Mode
	}
	if value.AlwaysShow {
		return "always"
	}
	return "auto"
}

// SetMode 只接受三个明确模式，并同时写入一致的兼容布尔字段。
// 非法输入不会调用存储，保存失败则原样返回底层错误。
func (s *Service) SetMode(mode string) error {
	if mode != "auto" && mode != "always" && mode != "never" {
		return sharederrors.NewLocalizedError("toc_display_invalid_mode", "Invalid TOC mode", "invalid toc mode", nil)
	}
	return s.store.Put(fileConfig{AlwaysShow: mode == "always", Mode: mode})
}
