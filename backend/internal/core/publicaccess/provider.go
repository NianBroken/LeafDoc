// 本文件定义公开读取配置的最小接口，供路由和逐请求权限判断共用。
package publicaccess

// Provider 只暴露开关和来源，不向 HTTP 消费者提供写入方法。
// Service 与 Fixed 返回的对象都实现此接口。
type Provider interface {
	// Enabled 返回公开读取开关，由消费方结合请求规则使用。
	Enabled() bool
	// EnvManaged 返回来源是否固定，供设置界面决定是否允许编辑。
	EnvManaged() bool
}

// ReadGate 仅要求读取开关，逐请求读取门禁无需依赖配置来源。
type ReadGate interface {
	Enabled() bool
}

// Fixed 将传入的运行时布尔值包装为只读 Provider，无需构造真实服务。
// 返回对象始终报告来源固定，不读取环境变量，也不查询数据库。
func Fixed(enabled bool) Provider { return fixedProvider(enabled) }

// fixedProvider 直接以布尔值保存状态，不包含可变引用。
type fixedProvider bool

// Enabled 返回底层布尔值，EnvManaged 始终返回 true。
func (f fixedProvider) Enabled() bool    { return bool(f) }
func (f fixedProvider) EnvManaged() bool { return true }
