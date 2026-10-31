// 本文件声明 SMTP 连接、发件资料与邮件链接地址，参数整体传给邮件配置。
package main

import (
	"time"

	"github.com/urfave/cli/v3"
)

// emailOptions 保存连接、认证、发件显示与传输安全参数。
// 主机为空时启动入口不启用邮件，并跳过其他邮件字段的组合校验。
type emailOptions struct {
	smtpHost               string
	smtpPort               int
	smtpUsername           string
	smtpPassword           string
	smtpFrom               string
	smtpFromName           string
	smtpSecurity           string
	smtpInsecureSkipVerify bool
	smtpTimeout            time.Duration
	publicURL              string
}

// Flags 将参数解析结果写入目标字段，字符串去除两端空白。
// 公开地址和传输模式的组合校验由启动入口执行。
func (o *emailOptions) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "smtp-host",
			// 主机是邮件启用条件，端口默认值不会单独启用邮件。
			Destination: &o.smtpHost,
			Category:    catEmail,
			Usage:       "SMTP host for password-reset/invite email; unset disables email entirely",
			Sources:     envVars("LEAFDOC_SMTP_HOST"),
			Config:      trimmed,
		},
		&cli.IntFlag{
			Name:        "smtp-port",
			Destination: &o.smtpPort,
			Category:    catEmail,
			Usage:       "SMTP server port",
			Value:       587,
			Sources:     envVars("LEAFDOC_SMTP_PORT"),
		},
		&cli.StringFlag{
			Name:        "smtp-username",
			Destination: &o.smtpUsername,
			Category:    catEmail,
			Usage:       "SMTP auth username",
			Sources:     envVars("LEAFDOC_SMTP_USERNAME"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "smtp-password",
			// 凭据从参数或环境读取，不作为默认值写入声明。
			Destination: &o.smtpPassword,
			Category:    catEmail,
			Usage:       "SMTP auth password (env var preferred)",
			Sources:     envVars("LEAFDOC_SMTP_PASSWORD"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "smtp-from",
			Destination: &o.smtpFrom,
			Category:    catEmail,
			Usage:       "From address for outgoing email (required when --smtp-host is set)",
			Sources:     envVars("LEAFDOC_SMTP_FROM"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "smtp-from-name",
			Destination: &o.smtpFromName,
			Category:    catEmail,
			Usage:       "From display name for outgoing email",
			Value:       "LeafDoc",
			Sources:     envVars("LEAFDOC_SMTP_FROM_NAME"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "smtp-security",
			// 模式要求固定小写值，启动检查不进行大小写归一化。
			Destination: &o.smtpSecurity,
			Category:    catEmail,
			Usage:       "SMTP transport security: none, starttls, or tls",
			Value:       "starttls",
			Sources:     envVars("LEAFDOC_SMTP_SECURITY"),
			Config:      trimmed,
		},
		&cli.BoolFlag{
			Name:        "smtp-insecure-skip-verify",
			Destination: &o.smtpInsecureSkipVerify,
			Category:    catEmail,
			Usage:       "skip TLS certificate verification for SMTP (do not use in production)",
			Sources:     envBoolVars("LEAFDOC_SMTP_INSECURE_SKIP_VERIFY"),
		},
		&cli.DurationFlag{
			Name: "smtp-timeout",
			// 邮件客户端使用此期限约束发送，包括连接与隐式 TLS 握手阶段。
			Destination: &o.smtpTimeout,
			Category:    catEmail,
			Usage:       "timeout for a single SMTP send (e.g. 10s)",
			Value:       10 * time.Second,
			Sources:     envVars("LEAFDOC_SMTP_TIMEOUT"),
			DefaultText: "10s",
		},
		&cli.StringFlag{
			Name: "public-url",
			// 操作链接使用此地址，启用 SMTP 时要求绝对 HTTP 或 HTTPS 地址。
			Destination: &o.publicURL,
			Category:    catEmail,
			Usage:       "base URL for links in outgoing email, e.g. https://wiki.example.com (required with --smtp-host)",
			Sources:     envVars("LEAFDOC_PUBLIC_URL"),
			Config:      trimmed,
		},
	}
}
