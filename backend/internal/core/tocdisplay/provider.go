// 本文件定义目录显示的只读接口及无持久化的固定实现。
package tocdisplay

// Provider 同时提供兼容布尔值和模式字符串，供配置接口读取。
type Provider interface {
	// AlwaysShow 表示当前是否采用始终显示模式，不在此接口中计算页面标题数量。
	AlwaysShow() bool
	// Mode 返回目录初始显示模式的文本标识。
	Mode() string
}

// Fixed 将传入布尔值包装成只读提供者，true 对应 always，false 对应 auto。
// 它不能表达 never，不读取数据库或环境配置。
func Fixed(alwaysShow bool) Provider { return fixedProvider(alwaysShow) }

// fixedProvider 直接保存目录显示布尔值，不提供修改方法。
type fixedProvider bool

// AlwaysShow 返回固定布尔值。
func (f fixedProvider) AlwaysShow() bool { return bool(f) }

// Mode 将固定布尔值映射为兼容的两种模式。
func (f fixedProvider) Mode() string {
	if f {
		return "always"
	}
	return "auto"
}
