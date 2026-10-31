// 本文件验证命令解析、配置校验、独立监听、服务关停以及构建版本传递。
// 参数测试替换根动作避免启动实例，生命周期测试使用专用测试端口和通道控制时序。
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/serviceports"

	"github.com/urfave/cli/v3"

	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// runRootCommand 保留真实命令树与参数解析，将根动作替换为配置检查回调。
// 标准输出与错误输出丢弃，解析失败直接使测试失败，不创建 Wiki 或监听业务端口。
func runRootCommand(t *testing.T, args []string, inspect func(cfg *serverConfig)) {
	t.Helper()

	cfg := &serverConfig{}
	cmd := newRootCommandWithConfig(cfg)
	cmd.Writer = io.Discard
	cmd.ErrWriter = io.Discard
	cmd.Action = func(_ context.Context, _ *cli.Command) error {
		inspect(cfg)
		return nil
	}

	if err := cmd.Run(context.Background(), append([]string{"leafdoc"}, args...)); err != nil {
		t.Fatalf("cmd.Run(%q) error = %v", args, err)
	}
}

// TestRootCommandHelp_DocumentsFlagsAndEnvVars 实际运行帮助入口，
// 检查关键参数、对应环境变量和维护子命令均出现在生成内容中。
func TestRootCommandHelp_DocumentsFlagsAndEnvVars(t *testing.T) {
	var buf bytes.Buffer

	cmd := newRootCommand()
	cmd.Writer = &buf
	cmd.ErrWriter = &buf
	if err := cmd.Run(context.Background(), []string{"leafdoc", "--help"}); err != nil {
		t.Fatalf("cmd.Run(--help) error = %v", err)
	}

	output := buf.String()
	for _, expected := range []string{
		"--jwt-secret",
		"--admin-password",
		"--admin-username",
		"--admin-email",
		"--allow-insecure",
		"--enable-metrics",
		"--metrics-host",
		"--metrics-port",
		"--data-dir",
		"--unix-socket",
		"--log-format",
		"LEAFDOC_UNIX_SOCKET",
		"LEAFDOC_LOG_FORMAT",
		"LEAFDOC_ADMIN_USERNAME",
		"LEAFDOC_ADMIN_EMAIL",
		"LEAFDOC_ENABLE_METRICS",
		"LEAFDOC_METRICS_HOST",
		"LEAFDOC_METRICS_PORT",
		"reset-admin-password",
		"restore-snapshot",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected usage output to contain %q, got %q", expected, output)
		}
	}
}

// TestRootCommand_AcceptsSingleDashLongFlags 用单短横线长参数解析认证与指标字段，
// 同时核对字符串目标值和两个布尔开关，不执行真正的服务动作。
func TestRootCommand_AcceptsSingleDashLongFlags(t *testing.T) {
	runRootCommand(t, []string{
		"-jwt-secret=test-secret",
		"-admin-password=test-password",
		"-admin-username=test-admin",
		"-admin-email=test-admin@example.com",
		"-allow-insecure=true",
		"-enable-metrics=true",
		"-metrics-host=127.0.0.2",
		"-metrics-port=24694",
	}, func(cfg *serverConfig) {
		for _, tc := range []struct {
			flag string
			got  string
			want string
		}{
			{"jwt-secret", cfg.auth.jwtSecret, "test-secret"},
			{"admin-password", cfg.auth.adminPassword, "test-password"},
			{"admin-username", cfg.auth.adminUsername, "test-admin"},
			{"admin-email", cfg.auth.adminEmail, "test-admin@example.com"},
			{"metrics-host", cfg.metrics.metricsHost, "127.0.0.2"},
			{"metrics-port", cfg.metrics.metricsPort, "24694"},
		} {
			if tc.got != tc.want {
				t.Fatalf("expected --%s %q, got %q", tc.flag, tc.want, tc.got)
			}
		}
		if !cfg.server.allowInsecure {
			t.Fatal("expected --allow-insecure to be true")
		}
		if !cfg.metrics.enableMetrics {
			t.Fatal("expected --enable-metrics to be true")
		}
	})
}

// TestValidateHTTPRemoteUserConfig 覆盖代理关闭、单项与多项来源、
// 空值及纯分隔符输入，验证启用认证必须具备非空来源项。
func TestValidateHTTPRemoteUserConfig(t *testing.T) {
	tests := []struct {
		name            string
		enabled         bool
		trustedProxyIPs string
		wantErr         bool
	}{
		{"disabled, no IPs", false, "", false},
		{"disabled, with IPs", false, "127.0.0.1", false},
		{"enabled, with IPs", true, "127.0.0.1", false},
		{"enabled, multiple IPs", true, "127.0.0.1,172.18.0.0/16", false},
		{"enabled, no IPs", true, "", true},
		{"enabled, whitespace only", true, "   ", true},
		{"enabled, commas only", true, ",,,", true},
		{"enabled, commas and whitespace", true, " , , ", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHTTPRemoteUserConfig(tc.enabled, tc.trustedProxyIPs)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateHTTPRemoteUserConfig(%v, %q) error = %v, wantErr %v", tc.enabled, tc.trustedProxyIPs, err, tc.wantErr)
			}
		})
	}
}

// TestValidateHTTPRemoteUserAutoCreateConfig 覆盖自动建号与代理开关组合，
// 确认可用的只读和编辑角色通过，管理员与未知角色被拒绝。
func TestValidateHTTPRemoteUserAutoCreateConfig(t *testing.T) {
	tests := []struct {
		name              string
		autoCreateEnabled bool
		remoteUserEnabled bool
		defaultRole       string
		wantErr           bool
	}{
		{"auto-create disabled, everything else irrelevant", false, false, "", false},
		{"auto-create enabled, remote-user disabled", true, false, "viewer", true},
		{"auto-create enabled, remote-user enabled, valid role", true, true, "viewer", false},
		{"auto-create enabled, remote-user enabled, editor role", true, true, "editor", false},
		{"auto-create enabled, remote-user enabled, admin role forbidden", true, true, "admin", true},
		{"auto-create enabled, remote-user enabled, invalid role", true, true, "superuser", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateHTTPRemoteUserAutoCreateConfig(tc.autoCreateEnabled, tc.remoteUserEnabled, tc.defaultRole)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateHTTPRemoteUserAutoCreateConfig(%v, %v, %q) error = %v, wantErr %v", tc.autoCreateEnabled, tc.remoteUserEnabled, tc.defaultRole, err, tc.wantErr)
			}
		})
	}
}

// TestValidateRedirectURL 检查空地址、不同大小写的 HTTP 协议前缀，
// 并拒绝相对路径、脚本协议和数据协议。
func TestValidateRedirectURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"empty", "", false},
		{"http", "http://idp.example.com/login", false},
		{"https", "https://idp.example.com/login", false},
		{"uppercase scheme", "HTTPS://idp.example.com/login", false},
		{"mixed-case scheme", "Https://idp.example.com/login", false},
		{"javascript scheme", "javascript:alert(1)", true},
		{"relative path", "/login", true},
		{"data scheme", "data:text/html,<script>alert(1)</script>", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedirectURL("login-url", tc.url)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateRedirectURL(%q) error = %v, wantErr %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

// TestValidateRedirectURL_UserManagementURL 对外部用户管理地址执行同一前缀检查，
// 另外确认失败文字包含参数名，便于定位是哪一个外部入口配置无效。
func TestValidateRedirectURL_UserManagementURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"empty", "", false},
		{"https", "https://idp.example.com/users", false},
		{"javascript scheme", "javascript:alert(1)", true},
		{"relative path", "/users", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRedirectURL("user-management-url", tc.url)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateRedirectURL(%q) error = %v, wantErr %v", tc.url, err, tc.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "user-management-url") {
				t.Fatalf("validateRedirectURL(%q) error = %v, want it to mention the flag name", tc.url, err)
			}
		})
	}
}

// TestResolveLogoutURL 验证显式登出地址优先、兼容字段回退和两者为空三种结果，
// 同时检查是否需要记录兼容字段告警的返回标记。
func TestResolveLogoutURL(t *testing.T) {
	tests := []struct {
		name               string
		logoutURL          string
		deprecatedFlagVal  string
		wantResolved       string
		wantUsedDeprecated bool
	}{
		{
			name:         "new flag set, deprecated ignored",
			logoutURL:    "https://idp.example.com/logout",
			wantResolved: "https://idp.example.com/logout",
		},
		{
			name:               "only deprecated flag set",
			deprecatedFlagVal:  "https://idp.example.com/logout",
			wantResolved:       "https://idp.example.com/logout",
			wantUsedDeprecated: true,
		},
		{
			name:         "neither set",
			wantResolved: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved, usedDeprecated := resolveLogoutURL(tc.logoutURL, tc.deprecatedFlagVal)
			if resolved != tc.wantResolved {
				t.Fatalf("resolveLogoutURL() resolved = %q, want %q", resolved, tc.wantResolved)
			}
			if usedDeprecated != tc.wantUsedDeprecated {
				t.Fatalf("resolveLogoutURL() usedDeprecated = %v, want %v", usedDeprecated, tc.wantUsedDeprecated)
			}
		})
	}
}

// TestRootCommand_TrimsStringFlagValue 确认字符串参数两端空格在校验与使用前被去除。
func TestRootCommand_TrimsStringFlagValue(t *testing.T) {
	runRootCommand(t, []string{"--login-url= https://idp.example.com/login "}, func(cfg *serverConfig) {
		if got, want := cfg.proxy.loginURL, "https://idp.example.com/login"; got != want {
			t.Fatalf("login-url = %q, want %q", got, want)
		}
	})
}

// TestRootCommand_ResolutionPrecedence 分别验证环境覆盖默认、参数覆盖环境，
// 以及环境字符串裁剪，避免将来源优先级与值归一化混成同一个断言。
func TestRootCommand_ResolutionPrecedence(t *testing.T) {
	t.Run("environment variable overrides the default", func(t *testing.T) {
		t.Setenv("LEAFDOC_HOST", "0.0.0.0")
		runRootCommand(t, nil, func(cfg *serverConfig) {
			if got, want := cfg.server.host, "0.0.0.0"; got != want {
				t.Fatalf("host = %q, want %q", got, want)
			}
		})
	})

	t.Run("flag overrides the environment variable", func(t *testing.T) {
		t.Setenv("LEAFDOC_HOST", "0.0.0.0")
		runRootCommand(t, []string{"--host=192.168.0.1"}, func(cfg *serverConfig) {
			if got, want := cfg.server.host, "192.168.0.1"; got != want {
				t.Fatalf("host = %q, want %q", got, want)
			}
		})
	})

	t.Run("environment variable values are trimmed", func(t *testing.T) {
		t.Setenv("LEAFDOC_DATA_DIR", "  /srv/leafdoc  ")
		runRootCommand(t, nil, func(cfg *serverConfig) {
			if got, want := cfg.server.dataDir, "/srv/leafdoc"; got != want {
				t.Fatalf("data-dir = %q, want %q", got, want)
			}
		})
	})
}

// TestRootCommand_EmptyEnvVarIsUnset 要求空或纯空白环境值不覆盖默认配置，
// 因此回环主机、固定后端端口与默认快照启用状态保持有效。
func TestRootCommand_EmptyEnvVarIsUnset(t *testing.T) {
	t.Setenv("LEAFDOC_HOST", "")
	t.Setenv("LEAFDOC_PORT", "   ")
	t.Setenv("LEAFDOC_SNAPSHOT", "")

	runRootCommand(t, nil, func(cfg *serverConfig) {
		if got, want := cfg.server.host, "127.0.0.1"; got != want {
			t.Fatalf("host = %q, want %q", got, want)
		}
		if got, want := cfg.server.port, "24692"; got != want {
			t.Fatalf("port = %q, want %q", got, want)
		}
		if !cfg.backup.snapshot {
			t.Fatal("snapshot = false, want the default true")
		}
	})
}

// TestRootCommand_BoolEnvVarSpellings 逐项验证文字、数字和大小写布尔环境值，
// 检查归一化结果实际写入匿名阅读字段。
func TestRootCommand_BoolEnvVarSpellings(t *testing.T) {
	for _, tc := range []struct {
		env  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"ON", true},
		{"y", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"n", false},
	} {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("LEAFDOC_PUBLIC_ACCESS", tc.env)
			runRootCommand(t, nil, func(cfg *serverConfig) {
				if got := cfg.auth.publicAccess; got != tc.want {
					t.Fatalf("public-access with LEAFDOC_PUBLIC_ACCESS=%q = %v, want %v", tc.env, got, tc.want)
				}
			})
		})
	}
}

// TestRootCommand_UsageErrorIsReportedOnce 传入拼错的主机参数，核对一次错误输出、
// 相近参数建议和已报告标记，使 main 不会再重复报告同一解析失败。
func TestRootCommand_UsageErrorIsReportedOnce(t *testing.T) {
	var out bytes.Buffer

	cmd := newRootCommand()
	cmd.Writer = &out
	cmd.ErrWriter = &out

	err := cmd.Run(context.Background(), []string{"leafdoc", "--hosts=1.2.3.4"})
	if !errors.Is(err, errReported) {
		t.Fatalf("cmd.Run() error = %v, want errReported so main stays quiet", err)
	}
	if got := strings.Count(out.String(), "not defined"); got != 1 {
		t.Fatalf("expected the usage error to be printed exactly once, got %d times in %q", got, out.String())
	}
	if !strings.Contains(out.String(), `Did you mean "--host"?`) {
		t.Fatalf("expected a suggestion for the mistyped flag, got %q", out.String())
	}
}

// TestRootCommand_UnknownCommandIsRejected 验证未知位置参数携带失败退出码与原名称。
func TestRootCommand_UnknownCommandIsRejected(t *testing.T) {
	var out bytes.Buffer

	cmd := newRootCommand()
	cmd.Writer = &out
	cmd.ErrWriter = &out
	// 替换退出处理器以检查错误对象，避免 cli 默认处理直接终止测试进程。
	cmd.ExitErrHandler = func(context.Context, *cli.Command, error) {}

	err := cmd.Run(context.Background(), []string{"leafdoc", "reset-admin-passwort"})
	if err == nil {
		t.Fatal("expected an error for an unknown command, got nil")
	}
	var exitErr cli.ExitCoder
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected a cli.ExitCoder, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("exit code = %d, want 1", exitErr.ExitCode())
	}
	if !strings.Contains(err.Error(), "reset-admin-passwort") {
		t.Fatalf("expected the error to name the unknown command, got %q", err.Error())
	}
}

// TestParseLogFormat_Precedence 经真实参数解析后再规范化格式，
// 验证 text 默认、环境 json、大小写处理和命令行优先级。
func TestParseLogFormat_Precedence(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		envVal   string
		wantForm string
	}{
		{
			name:     "neither set falls back to default",
			wantForm: "text",
		},
		{
			name:     "env var sets json",
			envVal:   "json",
			wantForm: "json",
		},
		{
			name:     "env var is case-insensitive",
			envVal:   "JSON",
			wantForm: "json",
		},
		{
			name:     "cli flag overrides env var",
			args:     []string{"--log-format=text"},
			envVal:   "json",
			wantForm: "text",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LEAFDOC_LOG_FORMAT", tc.envVal)
			runRootCommand(t, tc.args, func(cfg *serverConfig) {
				got, ok := parseLogFormat(cfg.server.logFormat)
				if !ok {
					t.Fatalf("parseLogFormat(%q) reported an invalid format", cfg.server.logFormat)
				}
				if got != tc.wantForm {
					t.Fatalf("parseLogFormat() = %q, want %q", got, tc.wantForm)
				}
			})
		})
	}
}

// TestSetupLogger_SelectsHandlerByFormat 向内存缓冲写入真实日志，
// 通过 JSON 合法性与文本消息字段检查处理器选择。
func TestSetupLogger_SelectsHandlerByFormat(t *testing.T) {
	t.Run("text format writes non-JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		setupLogger(&buf, "text")
		slog.Default().Info("hello")

		if json.Valid(buf.Bytes()) {
			t.Fatalf("expected non-JSON text output, got %q", buf.String())
		}
		if !strings.Contains(buf.String(), "msg=hello") {
			t.Fatalf("expected text output to contain msg=hello, got %q", buf.String())
		}
	})

	t.Run("json format writes valid JSON output", func(t *testing.T) {
		var buf bytes.Buffer
		setupLogger(&buf, "json")
		slog.Default().Info("hello")

		if !json.Valid(buf.Bytes()) {
			t.Fatalf("expected valid JSON output, got %q", buf.String())
		}
	})
}

// TestRunRestoreSnapshotCommand_MissingArg_ReturnsUsageError 验证缺少路径时返回专用错误。
func TestRunRestoreSnapshotCommand_MissingArg_ReturnsUsageError(t *testing.T) {
	err := runRestoreSnapshotCommand(t.TempDir(), "")
	if !errors.Is(err, errRestoreSnapshotUsage) {
		t.Fatalf("runRestoreSnapshotCommand() error = %v, want errRestoreSnapshotUsage", err)
	}
}

// TestRunRestoreSnapshotCommand_InvalidZipPath_PropagatesError 使用不存在的归档，
// 确认实际恢复错误不被混淆为缺少命令参数的用法错误。
func TestRunRestoreSnapshotCommand_InvalidZipPath_PropagatesError(t *testing.T) {
	dataDir := t.TempDir()
	zipPath := filepath.Join(dataDir, "does-not-exist.zip")

	err := runRestoreSnapshotCommand(dataDir, zipPath)
	if err == nil {
		t.Fatal("runRestoreSnapshotCommand() expected an error for a non-existent snapshot zip, got nil")
	}
	if errors.Is(err, errRestoreSnapshotUsage) {
		t.Fatalf("runRestoreSnapshotCommand() error = %v, want a restore error, not the usage error", err)
	}
}

// TestValidateListenConfig 覆盖套接字与主机端口显式设置的组合，
// 仅空套接字配置通过，任一非空套接字路径均被拒绝。
func TestValidateListenConfig(t *testing.T) {
	tests := []struct {
		name       string
		unixSocket string
		hostSet    bool
		portSet    bool
		wantErr    bool
	}{
		{
			name:       "tcp only is allowed",
			unixSocket: "",
			hostSet:    true,
			portSet:    true,
			wantErr:    false,
		},
		{
			name:       "unix socket only is rejected",
			unixSocket: "/tmp/leafdoc.sock",
			wantErr:    true,
		},
		{
			name:       "unix socket with host is rejected",
			unixSocket: "/tmp/leafdoc.sock",
			hostSet:    true,
			wantErr:    true,
		},
		{
			name:       "unix socket with port is rejected",
			unixSocket: "/tmp/leafdoc.sock",
			portSet:    true,
			wantErr:    true,
		},
		{
			name:       "unix socket with host and port is rejected",
			unixSocket: "/tmp/leafdoc.sock",
			hostSet:    true,
			portSet:    true,
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateListenConfig(tc.unixSocket, tc.hostSet, tc.portSet)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateListenConfig(%q, host=%v, port=%v) error = %v, wantErr %v", tc.unixSocket, tc.hostSet, tc.portSet, err, tc.wantErr)
			}
		})
	}
}

// TestRootCommand_AcceptsDoubleDashLongFlags 验证双短横线参数绑定到相同配置字段，
// 保持认证与指标参数的常规调用形式可用。
func TestRootCommand_AcceptsDoubleDashLongFlags(t *testing.T) {
	runRootCommand(t, []string{
		"--jwt-secret=test-secret",
		"--admin-password=test-password",
		"--admin-username=test-admin",
		"--admin-email=test-admin@example.com",
		"--allow-insecure=true",
		"--enable-metrics=true",
		"--metrics-host=127.0.0.2",
		"--metrics-port=24694",
	}, func(cfg *serverConfig) {
		for _, tc := range []struct {
			flag string
			got  string
			want string
		}{
			{"jwt-secret", cfg.auth.jwtSecret, "test-secret"},
			{"admin-password", cfg.auth.adminPassword, "test-password"},
			{"admin-username", cfg.auth.adminUsername, "test-admin"},
			{"admin-email", cfg.auth.adminEmail, "test-admin@example.com"},
			{"metrics-host", cfg.metrics.metricsHost, "127.0.0.2"},
			{"metrics-port", cfg.metrics.metricsPort, "24694"},
		} {
			if tc.got != tc.want {
				t.Fatalf("expected --%s %q, got %q", tc.flag, tc.want, tc.got)
			}
		}
		if !cfg.server.allowInsecure {
			t.Fatal("expected --allow-insecure to be true")
		}
		if !cfg.metrics.enableMetrics {
			t.Fatal("expected --enable-metrics to be true")
		}
	})
}

// TestStartMetricsServer_ServesOnlyMetricsEndpoint 建立真实指标监听，
// 读取指标文字并检查业务健康路径为 404，处理结束时关闭响应体与监听。
func TestStartMetricsServer_ServesOnlyMetricsEndpoint(t *testing.T) {
	metrics := httpmetrics.NewHTTPMetrics("test")
	stopServer, addr, err := startMetricsServer(metrics, "127.0.0.1", serviceports.Port("metricsTest"))
	if err != nil {
		t.Fatalf("startMetricsServer() error = %v", err)
	}
	defer stopServer()

	resp, err := http.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics failed: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading /metrics response failed: %v", err)
	}
	if !strings.Contains(string(body), "leafdoc_http_requests_in_flight") {
		t.Fatalf("expected metrics output, got %q", string(body))
	}

	notFoundResp, err := http.Get("http://" + addr + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health failed: %v", err)
	}
	defer func() {
		_ = notFoundResp.Body.Close()
	}()

	if notFoundResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 from non-metrics path, got %d", notFoundResp.StatusCode)
	}
}

// TestRemoveStaleUnixSocket_RemovesExistingSocket 在支持平台创建真实套接字，
// 确认清理函数删除路径，Windows 与路径过长环境跳过这一平台测试。
func TestRemoveStaleUnixSocket_RemovesExistingSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are not supported on windows")
	}
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "leafdoc.sock")
	if len(socketPath) >= 100 {
		t.Skipf("socket path too long (%d chars) for this platform", len(socketPath))
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	}()

	if err := removeStaleUnixSocket(socketPath); err != nil {
		t.Fatalf("removeStaleUnixSocket() error = %v", err)
	}
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("expected socket to be removed, stat err = %v", err)
	}
}

// TestRemoveStaleUnixSocket_RejectsNonSocketPath 在目标位置放普通文件，
// 验证清理函数拒绝将其作为套接字删除并给出明确错误。
func TestRemoveStaleUnixSocket_RejectsNonSocketPath(t *testing.T) {
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "leafdoc.sock")
	if err := os.WriteFile(socketPath, []byte("not a socket"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := removeStaleUnixSocket(socketPath)
	if err == nil {
		t.Fatalf("expected error for non-socket path")
	}
	if !strings.Contains(err.Error(), "is not a socket") {
		t.Fatalf("expected non-socket error, got %v", err)
	}
}

// TestListenOnUnixSocket_CreatesSocketWithExpectedPermissions 核对实际文件类型与 0660 权限，
// 通过 defer 释放监听并清理临时路径。
func TestListenOnUnixSocket_CreatesSocketWithExpectedPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are not supported on windows")
	}
	dir := t.TempDir()
	socketPath := filepath.Join(dir, "leafdoc.sock")
	if len(socketPath) >= 100 {
		t.Skipf("socket path too long (%d chars) for this platform", len(socketPath))
	}
	listener, err := listenOnUnixSocket(socketPath)
	if err != nil {
		t.Fatalf("listenOnUnixSocket() error = %v", err)
	}
	defer func() {
		_ = listener.Close()
		_ = os.Remove(socketPath)
	}()

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("expected %s to be a socket, mode = %v", socketPath, info.Mode())
	}
	if got := info.Mode().Perm(); got != 0660 {
		t.Fatalf("expected socket permissions 0660, got %#o", got)
	}
}

// TestListenOnUnixSocket_WindowsReturnsHelpfulError 仅在 Windows 验证明确的平台拒绝文字。
func TestListenOnUnixSocket_WindowsReturnsHelpfulError(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-specific behavior")
	}
	_, err := listenOnUnixSocket(`C:\leafdoc.sock`)
	if err == nil {
		t.Fatalf("expected error on windows")
	}
	if !strings.Contains(err.Error(), "not supported on windows") {
		t.Fatalf("expected windows support error, got %v", err)
	}
}

// testSignal 实现 os.Signal，测试只向内存通道发送，不向操作系统进程发真实信号。
type testSignal string

// String 返回场景名称，供生命周期日志读取。
func (s testSignal) String() string { return string(s) }

// Signal 是满足 os.Signal 接口的空标记方法。
func (testSignal) Signal() {}

// TestServeWithLifecycle_GracefulShutdownWaitsForInFlightRequest 用通道停住已进入的请求，
// 确认关停不会提前返回，放行后响应完整送达并干净退出。
func TestServeWithLifecycle_GracefulShutdownWaitsForInFlightRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-release
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}),
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+serviceports.Port("httpTest"))
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	reloadSignals := make(chan os.Signal)
	shutdownSignals := make(chan os.Signal, 1)

	runErr := make(chan error, 1)
	go func() {
		runErr <- serveWithLifecycle(server, listener, nil, func() {}, reloadSignals, shutdownSignals)
	}()

	respCh := make(chan *http.Response, 1)
	reqErrCh := make(chan error, 1)
	go func() {
		resp, err := http.Get("http://" + listener.Addr().String())
		if err != nil {
			reqErrCh <- err
			return
		}
		respCh <- resp
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach handler")
	}

	// 请求已进入处理器后才发送关停，避免把连接尚未建立误认为正在处理。
	shutdownSignals <- testSignal("shutdown")

	select {
	case err := <-runErr:
		t.Fatalf("server exited before request completed: %v", err)
	case <-time.After(150 * time.Millisecond):
	}

	close(release)

	var resp *http.Response
	select {
	case err := <-reqErrCh:
		t.Fatalf("request failed: %v", err)
	case resp = <-respCh:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not complete")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("closing response body failed: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if string(body) != "ok" {
		t.Fatalf("expected body ok, got %q", string(body))
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not exit after request completed")
	}
}

// TestServeWithLifecycle_ReloadSignalTriggersCallbackWithoutStoppingServer 等待重载回调通知，
// 再发真实请求证明监听继续可用，最终显式发送关停并等待退出。
func TestServeWithLifecycle_ReloadSignalTriggersCallbackWithoutStoppingServer(t *testing.T) {
	var reloadCalls atomic.Int32

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+serviceports.Port("httpTest"))
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	reloadDone := make(chan struct{}, 1)
	reloadSignals := make(chan os.Signal, 1)
	shutdownSignals := make(chan os.Signal, 1)

	runErr := make(chan error, 1)
	go func() {
		runErr <- serveWithLifecycle(server, listener, nil, func() {
			reloadCalls.Add(1)
			reloadDone <- struct{}{}
		}, reloadSignals, shutdownSignals)
	}()

	reloadSignals <- testSignal("reload")

	select {
	case <-reloadDone:
	case <-time.After(2 * time.Second):
		t.Fatal("reload callback was not triggered")
	}

	resp, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("request after reload failed: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("closing response body failed: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", resp.StatusCode)
	}

	if reloadCalls.Load() != 1 {
		t.Fatalf("expected one reload call, got %d", reloadCalls.Load())
	}

	shutdownSignals <- testSignal("shutdown")

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after shutdown signal")
	}
}

// TestServeWithLifecycle_ShutdownTimeoutStillRunsCleanup 临时缩短关停期限，
// 让处理器等到连接被强制关闭，核对期限错误、清理回调和请求上下文取消。
func TestServeWithLifecycle_ShutdownTimeoutStillRunsCleanup(t *testing.T) {
	previousTimeout := gracefulShutdownTimeout
	gracefulShutdownTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		gracefulShutdownTimeout = previousTimeout
	})

	started := make(chan struct{})
	handlerCanceled := make(chan struct{}, 1)
	cleanupCalled := make(chan struct{}, 1)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-r.Context().Done()
			handlerCanceled <- struct{}{}
		}),
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+serviceports.Port("httpTest"))
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	reloadSignals := make(chan os.Signal)
	shutdownSignals := make(chan os.Signal, 1)
	runErr := make(chan error, 1)
	go func() {
		runErr <- serveWithLifecycle(server, listener, func() {
			select {
			case cleanupCalled <- struct{}{}:
			default:
			}
		}, func() {}, reloadSignals, shutdownSignals)
	}()

	go func() {
		resp, err := http.Get("http://" + listener.Addr().String())
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach handler")
	}

	shutdownSignals <- testSignal("shutdown")

	select {
	case err := <-runErr:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected context deadline exceeded, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not exit after shutdown timeout")
	}

	select {
	case <-cleanupCalled:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup was not called on shutdown timeout")
	}

	select {
	case <-handlerCanceled:
	case <-time.After(2 * time.Second):
		t.Fatal("handler was not canceled after shutdown timeout")
	}
}

// TestServeWithLifecycle_ShutdownDoesNotWaitForInFlightReload 让重载回调启动独立后台工作后返回，
// 验证服务关停不等待该后台工作，再显式放行并等待测试工作结束。
// 此场景不表示关停能越过一个尚未返回的同步重载回调。
func TestServeWithLifecycle_ShutdownDoesNotWaitForInFlightReload(t *testing.T) {
	reloadStarted := make(chan struct{})
	releaseReload := make(chan struct{})
	reloadFinished := make(chan struct{})

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
	}

	listener, err := net.Listen("tcp", "127.0.0.1:"+serviceports.Port("httpTest"))
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	reloadSignals := make(chan os.Signal, 1)
	shutdownSignals := make(chan os.Signal, 1)
	runErr := make(chan error, 1)
	go func() {
		runErr <- serveWithLifecycle(server, listener, nil, func() {
			close(reloadStarted)
			go func() {
				<-releaseReload
				close(reloadFinished)
			}()
		}, reloadSignals, shutdownSignals)
	}()

	reloadSignals <- testSignal("reload")

	select {
	case <-reloadStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("reload did not start")
	}

	shutdownSignals <- testSignal("shutdown")

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop while reload was still running")
	}

	close(releaseReload)

	select {
	case <-reloadFinished:
	case <-time.After(2 * time.Second):
		t.Fatal("reload did not finish after release")
	}
}

// goBuildLdflagsLine 返回 Dockerfile 中首个包含 -ldflags= 的行，
// 未找到时使测试失败，不解析或执行 Dockerfile 指令。
func goBuildLdflagsLine(t *testing.T, dockerfilePath string) string {
	t.Helper()

	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", dockerfilePath, err)
	}

	for _, line := range strings.Split(string(content), "\n") {
		if strings.Contains(line, "-ldflags=") {
			return line
		}
	}

	t.Fatalf("no -ldflags line found in %s", dockerfilePath)
	return ""
}

// TestDockerfile_GoBuildLdflags_InjectsAppVersion 检查应用镜像的 Go 构建行
// 显式把 APP_VERSION 注入 main.Version，使后端版本与构建输入对应。
func TestDockerfile_GoBuildLdflags_InjectsAppVersion(t *testing.T) {
	line := goBuildLdflagsLine(t, filepath.Join("..", "..", "..", "other", "Dockerfile"))

	if !strings.Contains(line, "-X main.Version=${APP_VERSION}") {
		t.Fatalf("expected Dockerfile go build ldflags to inject main.Version from APP_VERSION, got: %s", line)
	}
}

// TestDockerfileBuilder_GoBuildLdflags_InjectsAppVersion 对二进制构建镜像执行相同版本注入检查。
func TestDockerfileBuilder_GoBuildLdflags_InjectsAppVersion(t *testing.T) {
	line := goBuildLdflagsLine(t, filepath.Join("..", "..", "..", "other", "Dockerfile.builder"))

	if !strings.Contains(line, "-X main.Version=${APP_VERSION}") {
		t.Fatalf("expected Dockerfile.builder go build ldflags to inject main.Version from APP_VERSION, got: %s", line)
	}
}

// dockerBuildStageDeclaresArg 找到首个标记行，向前定位所属 FROM 阶段，
// 只在该阶段起点到标记之间检查独立 ARG 声明，避免把其他阶段的声明当作有效来源。
func dockerBuildStageDeclaresArg(t *testing.T, dockerfilePath, marker, argName string) bool {
	t.Helper()

	content, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("failed to read %s: %v", dockerfilePath, err)
	}

	lines := strings.Split(string(content), "\n")

	markerIdx := -1
	for i, line := range lines {
		if strings.Contains(line, marker) {
			markerIdx = i
			break
		}
	}
	if markerIdx == -1 {
		t.Fatalf("no line containing %q found in %s", marker, dockerfilePath)
	}

	stageStart := 0
	for i := markerIdx; i >= 0; i-- {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "FROM ") {
			stageStart = i
			break
		}
	}

	for _, line := range lines[stageStart:markerIdx] {
		if strings.TrimSpace(line) == "ARG "+argName {
			return true
		}
	}
	return false
}

// TestDockerfile_GoBuildStage_DeclaresAppVersionArg 要求应用后端构建阶段
// 自身声明 APP_VERSION，使版本注入的来源处于同一 Docker 构建阶段。
func TestDockerfile_GoBuildStage_DeclaresAppVersionArg(t *testing.T) {
	if !dockerBuildStageDeclaresArg(t, filepath.Join("..", "..", "..", "other", "Dockerfile"), "go build", "APP_VERSION") {
		t.Fatal("Dockerfile's go build stage uses ${APP_VERSION} without declaring ARG APP_VERSION in that stage")
	}
}

// TestDockerfileBuilder_GoBuildStage_DeclaresAppVersionArg 检查二进制构建阶段的版本参数作用域。
func TestDockerfileBuilder_GoBuildStage_DeclaresAppVersionArg(t *testing.T) {
	if !dockerBuildStageDeclaresArg(t, filepath.Join("..", "..", "..", "other", "Dockerfile.builder"), "go build", "APP_VERSION") {
		t.Fatal("Dockerfile.builder's builder stage uses ${APP_VERSION} without declaring ARG APP_VERSION in that stage")
	}
}

// TestResolveVersionScript_AppVersionEnvOverride_ReturnsEnvValue 实际运行版本脚本，
// 确认显式 APP_VERSION 原样输出。Windows 使用 Node 入口，其他平台使用 Bash 入口，
// 此测试不核对仓库标签回退分支。
func TestResolveVersionScript_AppVersionEnvOverride_ReturnsEnvValue(t *testing.T) {
	scriptPath := filepath.Join("..", "..", "..", "other", "scripts", "resolve-version.sh")
	cmdName, cmdArgs := "bash", []string{scriptPath}
	if runtime.GOOS == "windows" {
		cmdName, cmdArgs = "node", []string{filepath.Join("..", "..", "..", "other", "scripts", "resolve-version.mjs")}
	}
	cmd := exec.Command(cmdName, cmdArgs...)
	cmd.Env = append(os.Environ(), "APP_VERSION=v9.9.9-test")

	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve-version.sh failed: %v", err)
	}

	if got := strings.TrimSpace(string(out)); got != "v9.9.9-test" {
		t.Fatalf("expected resolve-version.sh to echo APP_VERSION override, got: %q", got)
	}
}

// TestContainerBuildUsesLinuxAndSharedVersion 检查容器入口包含共享版本脚本、
// 版本参数与 Linux 构建标记，测试仅核对源码文字，不运行镜像构建。
func TestContainerBuildUsesLinuxAndSharedVersion(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "other", "scripts", "container.mjs"))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"resolve-version.mjs", "APP_VERSION=", "GOOS=linux", "'linux'"} {
		if !strings.Contains(string(raw), required) {
			t.Fatalf("container launcher missing %q", required)
		}
	}
}

// TestViteConfig_ResolvesVersionViaSharedScript 检查前端构建配置引用共享版本脚本，
// 使前后端版本来源集中到同一入口。
func TestViteConfig_ResolvesVersionViaSharedScript(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "frontend", "vite.config.ts"))
	if err != nil {
		t.Fatalf("failed to read vite.config.ts: %v", err)
	}

	if !strings.Contains(string(content), "scripts/resolve-version.mjs") {
		t.Fatalf("expected vite.config.ts to resolve its version via scripts/resolve-version.mjs, got:\n%s", content)
	}
}
