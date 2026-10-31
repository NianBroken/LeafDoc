// 本文件声明身份认证、令牌期限与管理员初始化参数，解析结果由启动入口传给 Wiki。
package main

import (
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/defaults"

	"github.com/urfave/cli/v3"
)

// authOptions 保存签名密钥、匿名访问初始值、令牌期限与管理员初始化资料。
// 现有管理员由存储服务读取，普通启动不会使用这些参数重设其密码。
type authOptions struct {
	jwtSecret           string
	publicAccess        bool
	accessTokenTimeout  time.Duration
	refreshTokenTimeout time.Duration
	adminUsername       string
	adminEmail          string
	adminPassword       string
	editorLimit         int
}

// Flags 把选项名称、默认值、环境来源与目标字段放在同一个声明中。
// 帮助由 cli 库生成，字符串通过 trimmed 去除两端空白。
func (o *authOptions) Flags() []cli.Flag {
	return []cli.Flag{
		// 签名密钥没有默认值，服务启动检查非空后传给身份服务。
		&cli.StringFlag{
			Name:        "jwt-secret",
			Destination: &o.jwtSecret,
			Category:    catAuth,
			Usage:       "secret for signing auth tokens (JWT); required",
			Sources:     envVars("LEAFDOC_JWT_SECRET"),
			Config:      trimmed,
		},
		&cli.BoolFlag{
			Name: "public-access",
			// 此值作为首次设置来源，已有持久化设置由启动入口保留。
			Destination: &o.publicAccess,
			Value:       defaults.PublicAccess,
			Category:    catAuth,
			Usage:       "allow public access to the wiki with read access",
			Sources:     envBoolVars("LEAFDOC_PUBLIC_ACCESS"),
		},
		&cli.DurationFlag{
			Name: "access-token-timeout",
			// 访问令牌默认十五分钟，期限与刷新会话独立配置。
			Destination: &o.accessTokenTimeout,
			Category:    catAuth,
			Usage:       "access token timeout duration (e.g. 24h, 15m)",
			Value:       15 * time.Minute,
			Sources:     envVars("LEAFDOC_ACCESS_TOKEN_TIMEOUT"),
			DefaultText: "15m",
		},
		&cli.DurationFlag{
			Name: "refresh-token-timeout",
			// 七天默认值传给身份服务管理刷新令牌的有效期限。
			Destination: &o.refreshTokenTimeout,
			Category:    catAuth,
			Usage:       "refresh token timeout duration (e.g. 168h)",
			Value:       7 * 24 * time.Hour,
			Sources:     envVars("LEAFDOC_REFRESH_TOKEN_TIMEOUT"),
			DefaultText: "168h",
		},
		&cli.StringFlag{
			Name: "admin-username",
			// 管理员默认资料集中读取 defaults，维护子命令也复用这些字段。
			Destination: &o.adminUsername,
			Value:       defaults.AdminUsername,
			Category:    catAdmin,
			Usage:       "initial admin username, used only if no admin exists (default: Nian)",
			Sources:     envVars("LEAFDOC_ADMIN_USERNAME"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "admin-email",
			Destination: &o.adminEmail,
			Value:       defaults.AdminEmail,
			Category:    catAdmin,
			Usage:       "initial admin email, used only if no admin exists (default: Nian@LeafDoc.doc)",
			Sources:     envVars("LEAFDOC_ADMIN_EMAIL"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "admin-password",
			Destination: &o.adminPassword,
			Value:       defaults.AdminPassword,
			Category:    catAdmin,
			Usage:       "initial admin password, used only if no admin exists; min 8 characters",
			Sources:     envVars("LEAFDOC_ADMIN_PASSWORD"),
			Config:      trimmed,
		},
		&cli.IntFlag{
			Name: "editor-limit",
			// 用户服务用此值限制管理员与编辑者总数，零表示不设上限。
			Destination: &o.editorLimit,
			Category:    catAuth,
			Usage:       "maximum admin+editor users allowed; 0 = unlimited",
			Sources:     envVars("LEAFDOC_EDITOR_LIMIT"),
		},
	}
}
