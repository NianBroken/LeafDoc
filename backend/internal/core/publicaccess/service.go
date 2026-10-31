// 本文件管理实例公开读取开关，统一提供固定配置和数据库托管配置两种来源。
package publicaccess

import (
	"errors"
	"fmt"

	"github.com/NianBroken/LeafDoc/internal/core/defaults"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// fileConfig 是 public-access.json 数据库记录的 JSON 内容，不对应磁盘配置文件。
// Enabled 只表示公开读取设置，具体请求的身份和访问判断由 HTTP 层执行。
type fileConfig struct {
	Enabled bool `json:"enabled"`
}

// Service 通过 settings.Value 读取和保存开关。
// 数据库来源由 Store 的锁保护，固定来源保存构造时的布尔值。
type Service struct {
	val settings.Value[fileConfig]
}

// NewEnvManaged 将 enabled 保存为固定值，不读取数据库。
// 此实例的 SetEnabled 返回本地化只读错误，Reload 则直接成功返回。
func NewEnvManaged(enabled bool) *Service {
	return &Service{val: settings.NewFixed(fileConfig{Enabled: enabled})}
}

// NewSettingsManaged 加载 settings 命名空间中的 public-access.json 数据库记录。
// 记录缺失时写入共享默认值，缺失字段由通用存储物化，非法记录使构造失败。
func NewSettingsManaged(storageDir string) (*Service, error) {
	st, err := settings.New(storageDir, "public-access.json", 0o600, fileConfig{Enabled: defaults.PublicAccess}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read public-access config: %w", err)
	}
	return &Service{val: settings.Managed[fileConfig]{Store: st}}, nil
}

// Enabled 返回当前开关，不在此方法内判断具体页面、HTTP 方法或用户身份。
func (s *Service) Enabled() bool {
	return s.val.Get().Enabled
}

// EnvManaged 返回配置来源是否固定，来源在构造时确定，不随 SetEnabled 改变。
func (s *Service) EnvManaged() bool {
	return s.val.EnvManaged()
}

// SetEnabled 请求保存开关，成功后由数据库存储更新缓存并通知订阅者。
// 固定值错误转换为 ErrCodeEnvManaged，其他保存错误包装成更新失败并保留底层原因。
func (s *Service) SetEnabled(enabled bool) error {
	switch err := s.val.Put(fileConfig{Enabled: enabled}); {
	case errors.Is(err, settings.ErrEnvManaged):
		return errEnvManaged()
	case err != nil:
		return sharederrors.NewLocalizedError(
			"public_access_update_failed",
			"Failed to update public access mode",
			"failed to persist public-access config",
			err,
		)
	}
	return nil
}

// Reload 让底层来源重新加载配置，数据库来源读取当前记录，固定来源不作任何修改。
// 读取失败加上公开访问上下文后返回，不在这里切换来源或采用其他默认值。
func (s *Service) Reload() error {
	if err := s.val.Reload(); err != nil {
		return fmt.Errorf("failed to reload public-access config: %w", err)
	}
	return nil
}
