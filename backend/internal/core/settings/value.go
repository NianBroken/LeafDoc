// 本文件用同一接口区分数据库托管值和启动时固定值，减少业务层反复判断配置来源。
package settings

import "errors"

// ErrEnvManaged 表示固定值拒绝运行时写入，调用方可通过 errors.Is 识别。
// 业务层据此转换成适合自身接口的提示，本包不直接选择 HTTP 状态码。
var ErrEnvManaged = errors.New("settings: value is managed by environment configuration")

// Value 为配置提供读取、写入、重载和来源标识。
// Managed 通过 Store 保存到 MySQL，Fixed 只保存构造时传入的值，二者均不以 JSON 文件存储。
type Value[T any] interface {
	Reloadable

	// Get 返回当前缓存值或固定值，是否重新读取存储由具体实现决定。
	Get() T

	// Put 请求保存输入，Fixed 始终返回 ErrEnvManaged 且不改变任何存储。
	Put(v T) error

	// EnvManaged 表示是否按环境或启动参数托管的只读来源向业务层呈现。
	EnvManaged() bool
}

// Managed 嵌入数据库 Store，直接复用其 Get、Put 与 Reload。
// 使用前需提供已构造的非 nil Store，零值本身不具备可用存储。
type Managed[T any] struct {
	*Store[T]
}

// EnvManaged 对数据库托管值始终返回 false。
func (Managed[T]) EnvManaged() bool { return false }

// Fixed 保存构造时的值，不创建持久化记录，不接受 Put，也不在 Reload 时访问数据库。
// 值按 T 保存，包含引用类型时仍需调用方保证其内部内容不被外部修改。
type Fixed[T any] struct {
	v T
}

// NewFixed 按值保存 v，不解析环境变量，也不对引用字段执行深拷贝。
func NewFixed[T any](v T) Fixed[T] { return Fixed[T]{v: v} }

// Get 返回保存的固定值副本。
func (f Fixed[T]) Get() T { return f.v }

// Put 不检查输入是否与当前值相同，所有写入请求都返回固定值只读错误。
func (Fixed[T]) Put(T) error { return ErrEnvManaged }

// Reload 直接成功返回，不改变值，也不发送变化通知。
func (Fixed[T]) Reload() error { return nil }

// EnvManaged 对固定值始终返回 true，与值实际由何种调用方式提供无关。
func (Fixed[T]) EnvManaged() bool { return true }

// 编译期接口检查使用整数示例，保证两种实现具备 Value 要求的方法集合。
var (
	_ Value[int] = Managed[int]{}
	_ Value[int] = Fixed[int]{}
)
