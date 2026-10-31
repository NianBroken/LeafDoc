// 本文件声明可信代理身份头与外部页面地址，启动校验后传给路由认证层。
package main

import "github.com/urfave/cli/v3"

// proxyOptions 保存代理身份字段、自动建号策略、可信来源和外部跳转地址。
// 这些字段不自行读取请求头，实际信任判断由认证中间件执行。
type proxyOptions struct {
	enableHTTPRemoteUser           bool
	httpRemoteUserHeader           string
	enableHTTPRemoteUserAutoCreate bool
	httpRemoteUserEmailHeader      string
	httpRemoteUserDefaultRole      string
	trustedProxyIPs                string
	httpRemoteUserLogoutURL        string
	loginURL                       string
	logoutURL                      string
	userManagementURL              string
}

// Flags 声明代理与跳转参数，字符串去除两端空白。
// 跳转校验器只检查 HTTP 或 HTTPS 前缀，代理启用和自动建号组合在启动时校验。
func (o *proxyOptions) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "enable-http-remote-user",
			Destination: &o.enableHTTPRemoteUser,
			Category:    catProxyAuth,
			Usage:       "enable reverse-proxy authentication via HTTP header",
			Sources:     envBoolVars("LEAFDOC_ENABLE_HTTP_REMOTE_USER"),
		},
		&cli.StringFlag{
			Name: "http-remote-user-header-name",
			// 默认从 Remote-User 读取身份，只有可信代理校验通过后才使用该值。
			Destination: &o.httpRemoteUserHeader,
			Category:    catProxyAuth,
			Usage:       "HTTP header carrying the username or email from a trusted proxy",
			Value:       "Remote-User",
			Sources:     envVars("LEAFDOC_HTTP_REMOTE_USER_HEADER_NAME"),
			Config:      trimmed,
		},
		&cli.BoolFlag{
			Name:        "enable-http-remote-user-auto-create",
			Destination: &o.enableHTTPRemoteUserAutoCreate,
			Category:    catProxyAuth,
			Usage:       "auto-provision users asserted by the proxy but unknown to LeafDoc",
			Sources:     envBoolVars("LEAFDOC_ENABLE_HTTP_REMOTE_USER_AUTO_CREATE"),
		},
		&cli.StringFlag{
			Name:        "http-remote-user-email-header-name",
			Destination: &o.httpRemoteUserEmailHeader,
			Category:    catProxyAuth,
			Usage:       "HTTP header carrying the email for auto-created users",
			Sources:     envVars("LEAFDOC_HTTP_REMOTE_USER_EMAIL_HEADER_NAME"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "http-remote-user-default-role",
			// 默认只读角色，自动建号启动检查明确拒绝管理员角色。
			Destination: &o.httpRemoteUserDefaultRole,
			Category:    catProxyAuth,
			Usage:       "role assigned to auto-created users; must not be admin",
			Value:       "viewer",
			Sources:     envVars("LEAFDOC_HTTP_REMOTE_USER_DEFAULT_ROLE"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "trusted-proxy-ips",
			// 逗号分隔项交给认证层解析为 IP 或网段，不在此处拆分。
			Destination: &o.trustedProxyIPs,
			Category:    catProxyAuth,
			Usage:       "comma-separated trusted proxy IPs/CIDRs (e.g. 127.0.0.1,172.18.0.0/16)",
			Sources:     envVars("LEAFDOC_TRUSTED_PROXY_IPS"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "http-remote-user-logout-url",
			// 只有 logout-url 为空时启动入口才采用此兼容字段并记录弃用告警。
			Destination: &o.httpRemoteUserLogoutURL,
			Category:    catProxyAuth,
			Usage:       "deprecated: use --logout-url instead",
			Sources:     envVars("LEAFDOC_HTTP_REMOTE_USER_LOGOUT_URL"),
			Validator:   redirectURLValidator("http-remote-user-logout-url"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "login-url",
			Destination: &o.loginURL,
			Category:    catURLs,
			Usage:       "URL the frontend redirects to instead of the built-in login form (e.g. external SSO/IdP)",
			Sources:     envVars("LEAFDOC_LOGIN_URL"),
			Validator:   redirectURLValidator("login-url"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "logout-url",
			Destination: &o.logoutURL,
			Category:    catURLs,
			Usage:       "URL the frontend redirects to after logout (e.g. external SSO/IdP)",
			Sources:     envVars("LEAFDOC_LOGOUT_URL"),
			Validator:   redirectURLValidator("logout-url"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "user-management-url",
			// 作为前端外部管理入口传递，不在后端请求或代理该地址。
			Destination: &o.userManagementURL,
			Category:    catURLs,
			Usage:       "external user-management page; replaces the built-in User Management UI with a link",
			Sources:     envVars("LEAFDOC_USER_MANAGEMENT_URL"),
			Validator:   redirectURLValidator("user-management-url"),
			Config:      trimmed,
		},
	}
}
