// 本文件声明业务监听、数据目录、URL 前缀和日志参数，供启动流程统一读取。
package main

import (
	"github.com/NianBroken/LeafDoc/internal/core/serviceports"
	"github.com/urfave/cli/v3"
)

// serverOptions 保存业务服务参数。unixSocket 保留输入字段供明确拒绝，
// 对外命令行只接受共享规则分配的 TCP 端口。
type serverOptions struct {
	host              string
	port              string
	unixSocket        string
	dataDir           string
	basePath          string
	allowInsecure     bool
	logFormat         string
	disableRequestLog bool
}

// Flags 声明默认回环监听和相对数据目录。字符串去除两端空白，
// 固定端口与日志格式连同默认值一起接受校验，避免错误配置进入服务启动。
func (o *serverOptions) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "host",
			Destination: &o.host,
			Category:    catServer,
			Usage:       "host/IP address to bind the server to (e.g. 127.0.0.1 or 0.0.0.0)",
			Value:       "127.0.0.1",
			Sources:     envVars("LEAFDOC_HOST"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "port",
			// 默认端口与校验器使用同一共享分配，不接受临时端口零。
			Destination:      &o.port,
			Category:         catServer,
			Usage:            "port to run the server on",
			Value:            serviceports.Port("backend"),
			Validator:        serviceports.Validator("backend"),
			ValidateDefaults: true,
			Sources:          envVars("LEAFDOC_PORT"),
			Config:           trimmed,
		},
		&cli.StringFlag{
			Name: "unix-socket",
			// 非空值在参数校验阶段被拒绝，不启用内部保留的套接字监听分支。
			Destination: &o.unixSocket,
			Category:    catServer,
			Usage:       "unsupported: the backend requires its fixed TCP listener",
			Validator:   rejectUnixSocket,
			Sources:     envVars("LEAFDOC_UNIX_SOCKET"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "data-dir",
			Destination: &o.dataDir,
			Category:    catServer,
			Usage:       "path to data directory",
			Value:       "./data",
			Sources:     envVars("LEAFDOC_DATA_DIR"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name:        "base-path",
			Destination: &o.basePath,
			Category:    catServer,
			Usage:       "URL prefix when served behind a reverse proxy (e.g. /wiki)",
			Sources:     envVars("LEAFDOC_BASE_PATH"),
			Config:      trimmed,
		},
		&cli.BoolFlag{
			Name: "allow-insecure",
			// 此值传给认证 Cookie 配置，允许不带 Secure 属性的 Cookie。
			Destination: &o.allowInsecure,
			Category:    catServer,
			Usage:       "allow insecure HTTP; auth cookies may then travel in plain text",
			Sources:     envBoolVars("LEAFDOC_ALLOW_INSECURE"),
		},
		&cli.StringFlag{
			Name: "log-format",
			// 允许 text 和 json，大小写与两端空白由解析函数统一处理。
			Destination:      &o.logFormat,
			Category:         catLogging,
			Usage:            "log output format: text or json",
			Value:            "text",
			Sources:          envVars("LEAFDOC_LOG_FORMAT"),
			Validator:        validateLogFormatValue,
			ValidateDefaults: true,
			Config:           trimmed,
		},
		&cli.BoolFlag{
			Name:        "disable-request-log",
			Destination: &o.disableRequestLog,
			Category:    catLogging,
			Usage:       "suppress per-request HTTP access log lines",
			Sources:     envBoolVars("LEAFDOC_DISABLE_REQUEST_LOG"),
		},
	}
}
