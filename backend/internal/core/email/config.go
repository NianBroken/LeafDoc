// Package email 提供可选 SMTP 配置、消息构造和账号通知邮件发送。
package email

import "time"

// Security 选择 SMTP 连接的加密方式，none 使用普通 TCP，starttls 在 SMTP 会话中升级，
// tls 在会话开始前进行 TLS 握手。是否认证由用户名是否为空决定，不由 none 禁用。
type Security string

const (
	SecurityNone     Security = "none"
	SecurityStartTLS Security = "starttls"
	SecurityTLS      Security = "tls"
)

// defaultTimeout 用于未配置或非正数的邮件超时，服务层据此建立发送上下文。
const defaultTimeout = 10 * time.Second

// Config 描述 SMTP 连接、发件人、传输校验及邮件链接所用公开地址。
// 零值的 Enabled 返回 false，构造 Client 本身不会阻止使用不完整配置。
type Config struct {
	Host               string // SMTP 主机名，同时用于 TLS 服务器名称和 PLAIN 认证主机校验。
	Port               int
	Username           string
	Password           string
	From               string
	FromName           string
	Security           Security
	InsecureSkipVerify bool          // 为 true 时跳过 TLS 证书校验，零值保持校验。
	Timeout            time.Duration // 服务层发送期限，非正值使用 defaultTimeout。
	PublicURL          string        // 上层生成邮件操作链接使用的公开访问地址。
}

// Enabled 仅检查 Host 是否为空，不在此处校验端口、发件人、密码或安全模式。
func (c Config) Enabled() bool {
	return c.Host != ""
}

// TimeoutOrDefault 返回正数超时，未设置或非正数时返回十秒。
// 此方法只计算时长，实际上下文由服务层创建。
func (c Config) TimeoutOrDefault() time.Duration {
	if c.Timeout <= 0 {
		return defaultTimeout
	}
	return c.Timeout
}
