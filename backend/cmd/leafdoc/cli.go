// 本文件组装根命令与维护子命令，并集中定义参数来源、输入校验和用法错误输出。
// 各参数组负责声明字段，服务启动与维护动作在 main.go 中执行。
package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/serviceports"
	"github.com/urfave/cli/v3"
)

// 分类文字供 cli 帮助生成使用，属于用户可见输出而非注释。
const (
	catServer    = "Server"
	catLogging   = "Logging"
	catAuth      = "Authentication"
	catAdmin     = "Admin bootstrap"
	catProxyAuth = "Reverse-proxy authentication"
	catURLs      = "External URLs"
	catFrontend  = "Frontend"
	catFeatures  = "Features"
	catRevisions = "Revisions"
	catMetrics   = "Metrics"
	catSnapshots = "Snapshots & restore"
	catEmail     = "Email (SMTP)"
)

const rootUsageText = `leafdoc --jwt-secret <SECRET> --admin-password <PASSWORD> [--host <HOST>] [--port <PORT>] [--data-dir <DIR>]
leafdoc [--host <HOST>] [--port <PORT>] [--data-dir <DIR>]
leafdoc reset-admin-password
leafdoc [--data-dir <DIR>] restore-snapshot <path-to-zip>
leafdoc --help`

const rootDescription = `Every option can also be set through the environment variable shown next to it.
Command line flags win over environment variables, which win over the defaults.
An environment variable that is set but empty counts as unset.

LEAFDOC_LOG_LEVEL (debug, info, warn, error; default info) is available as an
environment variable only.

When --snapshot is enabled, live restore-from-snapshot is also available via the
admin UI (Settings > Full Backup) and gates writes (503) while a restore is
swapping files. For disaster recovery or migrating a snapshot to a fresh
instance, use the "restore-snapshot" command instead: run it before starting the
server against that data directory.`

// serverConfig 按监听、认证、代理、界面、指标、备份和邮件分组保存解析结果。
// 每组通过自己的 Flags 方法将声明绑定到字段，动作共享同一个配置对象。
type serverConfig struct {
	server   serverOptions
	auth     authOptions
	proxy    proxyOptions
	frontend frontendOptions
	metrics  metricsOptions
	backup   backupOptions
	email    emailOptions
}

// flags 按固定分组顺序拼接参数清单，此处未纳入的参数组不会被根命令解析。
func (c *serverConfig) flags() []cli.Flag {
	return slices.Concat(
		c.server.Flags(),
		c.auth.Flags(),
		c.proxy.Flags(),
		c.frontend.Flags(),
		c.metrics.Flags(),
		c.backup.Flags(),
		c.email.Flags(),
	)
}

// newRootCommand 为一次调用分配独立配置对象，避免命令解析状态在实例间共用。
func newRootCommand() *cli.Command {
	return newRootCommandWithConfig(&serverConfig{})
}

// newRootCommandWithConfig 将参数目标绑定到传入配置，注册服务端口与维护子命令。
// 未指定子命令时执行服务动作，多余位置参数按未知命令拒绝。
func newRootCommandWithConfig(cfg *serverConfig) *cli.Command {
	return &cli.Command{
		Name:        "LeafDoc",
		Usage:       "LeafDoc documentation server",
		UsageText:   rootUsageText,
		Description: rootDescription,
		Version:     Version,
		Suggest:     true,
		// 根据同一组参数声明生成 completion 子命令，输出各终端的补全脚本。
		EnableShellCompletion: true,
		// 用法错误在这里输出一次并返回已报告标记，main 据此避免重复打印。
		OnUsageError: reportUsageError,
		Flags:        cfg.flags(),
		Commands: []*cli.Command{
			{Name: "service-port", Usage: "Print a named fixed service port", Action: func(_ context.Context, cmd *cli.Command) error {
				for _, name := range serviceports.Names() {
					if cmd.Args().Len() == 1 && name == cmd.Args().First() {
						_, err := fmt.Fprintln(cmd.Writer, serviceports.Port(name))
						return err
					}
				}
				return fmt.Errorf("one allocated service name is required")
			}},
			{Name: "release-ports", Usage: "Release explicitly named fixed service listeners", Action: func(_ context.Context, cmd *cli.Command) error { return serviceports.Release(cmd.Args().Slice()...) }},
			newResetAdminPasswordCommand(cfg),
			newRestoreSnapshotCommand(cfg),
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			// 默认值和输入值已通过日志格式校验，此处取得规范化值并安装日志器。
			format, _ := parseLogFormat(cfg.server.logFormat)
			setupLogger(os.Stdout, format)
			return ctx, nil
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.Args().Present() {
				return cli.Exit(fmt.Sprintf("Unknown command: %s\nRun \"leafdoc --help\" to see the available commands and options.", cmd.Args().First()), 1)
			}
			return runServerCommand(ctx, cmd, cfg)
		},
	}
}

// newResetAdminPasswordCommand 复用根配置中的数据目录与管理员资料。
// 显式密码由维护动作直接读取进程环境，不从帮助文字或输出中取得。
func newResetAdminPasswordCommand(cfg *serverConfig) *cli.Command {
	return &cli.Command{
		Name:        "reset-admin-password",
		Usage:       "Reset the admin password and print the generated password",
		Description: "LEAFDOC_ADMIN_PASSWORD supplies an explicit password. Explicit passwords are never printed. The account ID is retained and existing sessions are revoked.",
		Action: func(_ context.Context, _ *cli.Command) error {
			return runResetAdminPasswordCommand(cfg)
		},
	}
}

// newRestoreSnapshotCommand 将第一个位置参数作为离线备份路径交给恢复动作。
// 缺少路径由动作返回专用用法错误，此处不启动 HTTP 服务。
func newRestoreSnapshotCommand(cfg *serverConfig) *cli.Command {
	return &cli.Command{
		Name:      "restore-snapshot",
		Usage:     "Restore a snapshot ZIP into the data directory while the server is stopped",
		ArgsUsage: "<path-to-zip>",
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runRestoreSnapshotCommand(cfg.server.dataDir, cmd.Args().First())
		},
	}
}

// reportUsageError 向根命令的错误输出写入解析失败、可用的相近参数提示和帮助入口。
// 返回已报告标记，让进程仍以失败结束而不由 main 再打印同一错误。
func reportUsageError(_ context.Context, cmd *cli.Command, err error, _ bool) error {
	out := cmd.Root().ErrWriter
	_, _ = fmt.Fprintf(out, "Incorrect usage: %s\n", err)
	if name := unknownFlagName(err); name != "" && cmd.Root().Suggest {
		if suggestion := cli.SuggestFlag(cmd.Flags, name, true); suggestion != "" && suggestion != name {
			_, _ = fmt.Fprintf(out, "Did you mean %q?\n", suggestion)
		}
	}
	_, _ = fmt.Fprintf(out, "Run %q to see the available commands and options.\n", "leafdoc --help")
	return errReported
}

// unknownFlagName 从错误文字的固定片段后提取参数名并去掉前导短横线。
// 找不到片段时返回空值，调用方不生成拼写建议。
func unknownFlagName(err error) string {
	_, name, found := strings.Cut(err.Error(), "not defined: ")
	if !found {
		return ""
	}
	return strings.TrimLeft(name, "-")
}

// validateLogFormatValue 使用实际日志格式解析函数校验输入。
// 声明中启用 ValidateDefaults 的参数会对默认值执行同一检查。
func validateLogFormatValue(v string) error {
	if _, ok := parseLogFormat(v); !ok {
		return fmt.Errorf("expected text or json, got %q", v)
	}
	return nil
}

// rejectUnixSocket 允许空字段，拒绝任何非空套接字路径以落实固定 TCP 监听策略。
func rejectUnixSocket(value string) error {
	if value == "" {
		return nil
	}
	return fmt.Errorf("unix sockets are disabled; use the fixed backend TCP port")
}

// validateByteSizeValue 复用启动时的字节解析规则，只返回错误而不保存转换结果。
func validateByteSizeValue(v string) error {
	_, err := parseByteSize(v)
	return err
}

// redirectURLValidator 捕获参数名称，使共用的地址检查能够报告对应选项。
func redirectURLValidator(flagName string) func(string) error {
	return func(v string) error {
		return validateRedirectURL(flagName, v)
	}
}

// trimmed 让字符串参数去除两端空白，命令行与环境来源使用同一配置。
var trimmed = cli.StringConfig{TrimSpace: true}

// envVars 按给定顺序构造环境来源链。空值和纯空白视为未设置，非空值裁剪空白。
// 例如空的主机环境值不会覆盖回环默认主机，从而避免被解释为所有网卡监听。
func envVars(keys ...string) cli.ValueSourceChain {
	sources := make([]cli.ValueSource, 0, len(keys))
	for _, key := range keys {
		sources = append(sources, &envVarSource{key: key})
	}
	return cli.NewValueSourceChain(sources...)
}

// envBoolVars 为环境布尔值增加 yes/no、on/off、y/n 的归一化支持。
// 无法识别的非空文字原样交给 cli 校验，不默认为 false，也不扩展命令行语法。
func envBoolVars(keys ...string) cli.ValueSourceChain {
	sources := make([]cli.ValueSource, 0, len(keys))
	for _, key := range keys {
		sources = append(sources, &envVarSource{key: key, normalize: normalizeBoolValue})
	}
	return cli.NewValueSourceChain(sources...)
}

// normalizeBoolValue 将已识别值转换为标准 true 或 false，其他文字原样返回。
func normalizeBoolValue(raw string) string {
	if b, ok := parseBool(raw); ok {
		if b {
			return "true"
		}
		return "false"
	}
	return raw
}

// envVarSource 适配一个环境变量，可选地归一化非空值。
// 它同时提供环境来源元信息，使帮助生成器显示参数对应的变量名。
type envVarSource struct {
	key       string
	normalize func(string) string
}

// Lookup 每次读取当前进程环境，布尔返回值区分有效非空来源与未设置来源。
func (e *envVarSource) Lookup() (string, bool) {
	val := strings.TrimSpace(os.Getenv(e.key))
	if val == "" {
		return "", false
	}
	if e.normalize != nil {
		val = e.normalize(val)
	}
	return val, true
}

// IsFromEnv 将此来源标识为环境变量，而非配置文件或其他来源。
func (e *envVarSource) IsFromEnv() bool { return true }

// Key 返回原始变量名，供 cli 生成环境说明。
func (e *envVarSource) Key() string { return e.key }

// String 描述来源名称，不输出环境变量的实际值。
func (e *envVarSource) String() string { return fmt.Sprintf("environment variable %q", e.key) }

// GoString 提供调试表示，只包含变量名，不包含读取到的凭据或配置值。
func (e *envVarSource) GoString() string { return fmt.Sprintf("&envVarSource{Key:%q}", e.key) }
