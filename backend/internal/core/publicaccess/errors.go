// 本文件统一声明公开访问只读设置的错误标识和本地化错误构造。
package publicaccess

import sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"

// ErrCodeEnvManaged 标识公开访问开关由启动配置固定，供调用方稳定识别拒绝写入的原因。
const ErrCodeEnvManaged = "public_access_env_managed"

// errEnvManaged 创建包含固定错误码、原始提示和翻译模板的错误，不包装底层原因。
func errEnvManaged() error {
	return sharederrors.NewLocalizedError(
		ErrCodeEnvManaged,
		"Public access mode is set by environment configuration and cannot be changed here",
		"public access mode is env-managed",
		nil,
	)
}
