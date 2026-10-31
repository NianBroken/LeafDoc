// 本文件执行命令动作、组装 Wiki 与备份恢复服务，并管理业务和指标监听的生命周期。
// 参数声明位于 flags 文件，平台信号登记位于平台文件，领域处理由内部服务承担。
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/auth/maintenance"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/email"
	"github.com/NianBroken/LeafDoc/internal/core/ignore"
	"github.com/NianBroken/LeafDoc/internal/core/localization"
	"github.com/NianBroken/LeafDoc/internal/core/preferences"
	"github.com/NianBroken/LeafDoc/internal/core/publicaccess"
	"github.com/NianBroken/LeafDoc/internal/core/restore"
	"github.com/NianBroken/LeafDoc/internal/core/settings"
	"github.com/NianBroken/LeafDoc/internal/core/snapshot"
	"github.com/NianBroken/LeafDoc/internal/core/tocdisplay"
	httpinternal "github.com/NianBroken/LeafDoc/internal/http"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/NianBroken/LeafDoc/internal/wiki"
	wikiinstancesettings "github.com/NianBroken/LeafDoc/internal/wiki/instancesettings"
	wikirestore "github.com/NianBroken/LeafDoc/internal/wiki/restore"
	wikisnapshot "github.com/NianBroken/LeafDoc/internal/wiki/snapshot"
	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v3"
)

// Version 保存构建版本，未通过链接参数注入时使用 dev。
// 构建脚本可用 -X main.Version 指定版本，指标与快照元数据读取同一值。
var Version = "dev"

// setupLogger 根据环境中的精确小写级别配置默认日志器，未识别值使用 info。
// json 选择 JSON 输出，其余格式使用文本，所有记录均附带调用位置。
func setupLogger(w io.Writer, format string) {
	level := slog.LevelInfo
	switch os.Getenv("LEAFDOC_LOG_LEVEL") {
	case "debug":
		level = slog.LevelDebug
	case "error":
		level = slog.LevelError
	case "warn":
		level = slog.LevelWarn
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	slog.SetDefault(slog.New(handler))
}

// fail 记录错误并直接以状态一退出进程，此路径不执行调用栈中的 defer。
func fail(msg string, args ...any) {
	slog.Default().Error(msg, args...)
	os.Exit(1)
}

var errRestoreSnapshotUsage = errors.New("restore-snapshot requires a path to a snapshot zip: leafdoc [--data-dir <DIR>] restore-snapshot <path-to-zip>")

// runRestoreSnapshotCommand 要求提供 ZIP 路径并执行离线恢复，不启动服务。
// 恢复失败保留底层错误链，成功时提示后续启动操作。
func runRestoreSnapshotCommand(dataDir, snapshotPath string) error {
	if snapshotPath == "" {
		return errRestoreSnapshotUsage
	}
	if err := restore.RestoreOffline(dataDir, snapshotPath); err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}
	fmt.Println("Snapshot restored successfully. Start the server normally to pick up the restored data.")
	return nil
}

// gracefulShutdownTimeout 限制业务与指标服务等待活动 HTTP 请求结束的时间。
var gracefulShutdownTimeout = 10 * time.Second

// errLogged 和 errReported 标识已通过日志器或命令错误输出报告的失败，
// main 仍以失败状态结束，但不会再打印含这些标记的错误。
var (
	errLogged   = errors.New("logged error")
	errReported = errors.New("reported error")
)

// main 执行命令并在命令返回后关闭共享数据库连接池，合并两处错误。
func main() {
	// 尚未报告的错误写入标准错误，已有报告标记避免重复输出同一失败。
	runErr := newRootCommand().Run(context.Background(), os.Args)
	err := errors.Join(runErr, database.ClosePools())
	if err != nil {
		if !errors.Is(err, errLogged) && !errors.Is(err, errReported) {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}

// runResetAdminPasswordCommand 将目录、账号资料和进程环境中的显式密码交给维护服务。
// 只有环境密码为空时才打印生成密码，显式密码不写入成功输出。
func runResetAdminPasswordCommand(cfg *serverConfig) error {
	user, err := maintenance.ResetAdminPassword(cfg.server.dataDir, cfg.auth.adminUsername, cfg.auth.adminEmail, os.Getenv("LEAFDOC_ADMIN_PASSWORD"))
	if err != nil {
		return fmt.Errorf("password reset failed: %w", err)
	}

	fmt.Println("Admin password reset successfully.")
	if os.Getenv("LEAFDOC_ADMIN_PASSWORD") == "" {
		fmt.Printf("New password for user %s: %s\n", user.Username, user.Password)
	}
	return nil
}

// runServerCommand 在根命令没有子命令时组装并运行服务。
// 先校验配置，再加载设置、创建 Wiki、备份恢复与 HTTP 路由。
func runServerCommand(_ context.Context, cmd *cli.Command, cfg *serverConfig) error {
	// 大小参数已通过命令校验，这里取得数值供上传入口设置请求限额。
	basePath := normalizeBasePath(cfg.server.basePath)
	maxAssetUploadSize := mustParseByteSize(cfg.frontend.maxAssetUploadSize, "max asset upload size")
	restoreUploadMaxSize := mustParseByteSize(cfg.backup.restoreUploadMaxSize, "restore upload max size")

	logoutURL := cfg.proxy.logoutURL
	if resolved, usedDeprecated := resolveLogoutURL(logoutURL, cfg.proxy.httpRemoteUserLogoutURL); usedDeprecated {
		slog.Default().Warn("--http-remote-user-logout-url/LEAFDOC_HTTP_REMOTE_USER_LOGOUT_URL is deprecated, use --logout-url/LEAFDOC_LOGOUT_URL instead")
		logoutURL = resolved
	}

	smtpEnabled := cfg.email.smtpHost != ""

	trustedProxies, err := authmw.ParseTrustedProxies(cfg.proxy.trustedProxyIPs)
	if err != nil {
		fail("invalid --trusted-proxy-ips value", "error", err)
	}
	if err := validateListenConfig(cfg.server.unixSocket, cmd.IsSet("host"), cmd.IsSet("port")); err != nil {
		fail("Invalid listen configuration", "error", err)
	}

	if err := validateHTTPRemoteUserConfig(cfg.proxy.enableHTTPRemoteUser, cfg.proxy.trustedProxyIPs); err != nil {
		fail("Invalid HTTP remote user configuration", "error", err)
	}

	if err := validateHTTPRemoteUserAutoCreateConfig(cfg.proxy.enableHTTPRemoteUserAutoCreate, cfg.proxy.enableHTTPRemoteUser, cfg.proxy.httpRemoteUserDefaultRole); err != nil {
		fail("Invalid HTTP remote user auto-create configuration", "error", err)
	}

	if err := validateSMTPConfig(cfg.email.smtpHost, cfg.email.smtpFrom, cfg.email.publicURL, cfg.email.smtpSecurity); err != nil {
		fail("Invalid SMTP configuration", "error", err)
	}
	if smtpEnabled {
		// 启动时记录实际邮件地址配置，便于在发送邮件前核对链接来源。
		slog.Default().Info("SMTP email enabled", "host", cfg.email.smtpHost, "port", cfg.email.smtpPort, "security", cfg.email.smtpSecurity, "publicUrl", cfg.email.publicURL)
	}

	if cfg.proxy.enableHTTPRemoteUser {
		slog.Default().Info("Reverse-proxy authentication enabled",
			"header", cfg.proxy.httpRemoteUserHeader,
			"trusted_proxies", cfg.proxy.trustedProxyIPs,
			"auto_create", cfg.proxy.enableHTTPRemoteUserAutoCreate,
			"default_role", cfg.proxy.httpRemoteUserDefaultRole,
		)
	}
	if cfg.metrics.enableMetrics {
		slog.Default().Info("Prometheus metrics enabled",
			"metrics_host", cfg.metrics.metricsHost,
			"metrics_port", cfg.metrics.metricsPort,
		)
	}

	if cfg.server.allowInsecure {
		slog.Default().Warn("allow-insecure enabled. Auth cookies may be transmitted over plain HTTP (INSECURE).")
	}

	// 只在目录不存在时递归创建，其他 Stat 错误由后续实际存取暴露。
	if _, err := os.Stat(cfg.server.dataDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cfg.server.dataDir, 0755); err != nil {
			fail("Failed to create data directory", "error", err)
		}
		slog.Default().Info("Data directory created", "path", cfg.server.dataDir)
	}

	publicAccessService := buildPublicAccessService(cmd, cfg)
	globalPreferences, err := preferences.New(cfg.server.dataDir)
	if err != nil {
		return err
	}

	// 语言提供器每次读取当前偏好，让后续错误翻译使用运行时设置。
	localization.SetLanguageProvider(func() string { return globalPreferences.Get().Language })
	tocDisplayService, err := tocdisplay.New(cfg.server.dataDir)
	if err != nil {
		fail("Failed to load toc-display configuration", "error", err)
	}

	if cfg.auth.jwtSecret == "" {
		fail("JWT secret is required. Set it using --jwt-secret or LEAFDOC_JWT_SECRET environment variable.")
	}

	var metrics *httpmetrics.HTTPMetrics
	if cfg.metrics.enableMetrics {
		metrics = httpmetrics.NewHTTPMetrics(Version)
	}

	// 将认证、修订、用户配额、邮件和指标依赖交给 Wiki 统一组装。
	w, err := wiki.NewWiki(&wiki.WikiOptions{
		StorageDir:             cfg.server.dataDir,
		AdminUsername:          cfg.auth.adminUsername,
		AdminEmail:             cfg.auth.adminEmail,
		AdminPassword:          cfg.auth.adminPassword,
		JWTSecret:              cfg.auth.jwtSecret,
		AccessTokenTimeout:     cfg.auth.accessTokenTimeout,
		RefreshTokenTimeout:    cfg.auth.refreshTokenTimeout,
		EnableRevision:         cfg.frontend.enableRevision,
		EnableAPIKeyManagement: cfg.frontend.enableAPIKeyManagement,
		MaxRevisionHistory:     cfg.frontend.maxRevisionHistory,
		EditorLimit:            cfg.auth.editorLimit,
		RevisionCoalesceWindow: cfg.frontend.revisionCoalesceWindow,
		SMTP: email.Config{
			Host:               cfg.email.smtpHost,
			Port:               cfg.email.smtpPort,
			Username:           cfg.email.smtpUsername,
			Password:           cfg.email.smtpPassword,
			From:               cfg.email.smtpFrom,
			FromName:           cfg.email.smtpFromName,
			Security:           email.Security(cfg.email.smtpSecurity),
			InsecureSkipVerify: cfg.email.smtpInsecureSkipVerify,
			Timeout:            cfg.email.smtpTimeout,
			PublicURL:          cfg.email.publicURL,
		},
		Metrics: metrics,
	})
	if err != nil {

		fail("Failed to initialize Wiki", "error", err)
	}

	w.SetInstanceSettingsRoutes(wikiinstancesettings.NewRoutes(publicAccessService, tocDisplayService, w.AuthService(), slog.Default()))

	// 读取 root 下的忽略规则并报告条数，读取失败只告警，不在这里终止启动。
	rootDir := filepath.Join(cfg.server.dataDir, "root")
	if ignoreFile, err := ignore.LoadFromDir(rootDir); err != nil {
		slog.Default().Warn("invalid .leafdocignore", "error", err)
	} else if ignoreFile != nil {
		slog.Default().Info("loaded .leafdocignore", "patterns", ignoreFile.PatternCount())
	}

	defer func() {
		if err := w.Close(); err != nil {
			slog.Default().Error("Failed to close Wiki", "error", err)
		}
	}()

	// 总开关启用时同时创建备份和在线恢复，二者共享写入屏障。
	var writeGate *restore.WriteGate
	if cfg.backup.snapshot {
		snapshotsDir := cfg.backup.snapshotDir
		if snapshotsDir == "" {
			snapshotsDir = filepath.Join(cfg.server.dataDir, "snapshots")
		}
		snapshotSettings, err := snapshot.NewSettingsStore(cfg.server.dataDir, cfg.backup.snapshotInterval, cfg.backup.snapshotRetention)
		if err != nil {
			return fmt.Errorf("failed to initialize full backup settings: %w", err)
		}
		writeGate = restore.NewWriteGate()
		snapshotManager := snapshot.NewManager(snapshot.Config{
			DataDir:        cfg.server.dataDir,
			BackupsDir:     snapshotsDir,
			AssetsDir:      filepath.Join(cfg.server.dataDir, "assets"),
			BrandingDir:    filepath.Join(cfg.server.dataDir, "branding"),
			AvatarsDir:     filepath.Join(cfg.server.dataDir, "avatars"),
			WikiVersion:    Version,
			Interval:       cfg.backup.snapshotInterval,
			RetentionCount: cfg.backup.snapshotRetention,
			// 归档前阻止新写入并等待已进入的写请求结束，超时立即解除屏障。
			Quiesce: func() (func(), error) {
				writeGate.Engage()
				if !writeGate.WaitForDrain(10 * time.Second) {
					writeGate.Disengage()
					return nil, fmt.Errorf("snapshot maintenance drain timed out")
				}
				return writeGate.Disengage, nil
			},
		})
		snapshotManager.UpdateSettings(snapshotSettings.Get())
		snapshotScheduler := snapshot.NewScheduler(snapshotManager)
		// 设置存储应用回调同步管理器配置，并通知调度器重新计算等待时间。
		snapshotSettings.SetApply(func(value snapshot.Settings) {
			snapshotManager.UpdateSettings(value)
			snapshotScheduler.UpdateSettings()
		})
		defer snapshotScheduler.Stop()
		w.SetSnapshotRoutes(wikisnapshot.NewRoutes(snapshotManager, snapshotScheduler, w.AuthService(), cfg.backup.snapshotRetention, snapshotSettings))

		restoreManager := restore.NewManager(restore.Config{
			SnapshotManager:    snapshotManager,
			DataDir:            cfg.server.dataDir,
			WikiVersion:        Version,
			WriteGate:          writeGate,
			AuthService:        w.AuthService(),
			Reloadables:        []settings.Reloadable{w.BrandingService(), publicAccessService, tocDisplayService, globalPreferences, snapshotSettings},
			UserResolver:       w.UserResolver(),
			ReloadRuntime:      w.ReloadAfterRestore,
			MaxUploadSizeBytes: restoreUploadMaxSize,
		})
		// defer 按后登记先执行的顺序运行，先等恢复任务结束，再停止调度器，
		// 随后关闭 Wiki，避免恢复仍使用账号和会话存储时资源已经关闭。
		defer restoreManager.Wait()
		w.SetRestoreRoutes(wikirestore.NewRoutes(restoreManager, w.AuthService()))
	}

	// 将领域注册器与运行选项交给公共路由，实例设置服务供请求期间动态读取。
	router := httpinternal.NewRouter(w.Registrars(), w.FrontendConfig(), httpinternal.RouterOptions{
		PublicAccess:            publicAccessService,
		Preferences:             globalPreferences,
		AuthService:             w.AuthService(),
		AlwaysShowToc:           tocDisplayService,
		EditorLimit:             cfg.auth.editorLimit,
		InjectCodeInHeader:      cfg.frontend.injectCodeInHeader,
		CustomStylesheet:        cfg.frontend.customStylesheet,
		AllowInsecure:           cfg.server.allowInsecure,
		HideLinkMetadataSection: cfg.frontend.hideLinkMetadataSection,
		AccessTokenTimeout:      cfg.auth.accessTokenTimeout,
		RefreshTokenTimeout:     cfg.auth.refreshTokenTimeout,
		BasePath:                basePath,
		MaxAssetUploadSizeBytes: maxAssetUploadSize,
		EnableRevision:          cfg.frontend.enableRevision,
		EnableLinkRefactor:      cfg.frontend.enableLinkRefactor,
		EnableAPIKeyManagement:  cfg.frontend.enableAPIKeyManagement,
		Metrics:                 metrics,
		SnapshotEnabled:         cfg.backup.snapshot,
		SMTPEnabled:             smtpEnabled,
		HTTPRemoteUser: httpinternal.HTTPRemoteUserConfig{
			Enabled:         cfg.proxy.enableHTTPRemoteUser,
			HeaderName:      cfg.proxy.httpRemoteUserHeader,
			AutoCreate:      cfg.proxy.enableHTTPRemoteUserAutoCreate,
			EmailHeaderName: cfg.proxy.httpRemoteUserEmailHeader,
			DefaultRole:     cfg.proxy.httpRemoteUserDefaultRole,
			TrustedProxies:  trustedProxies,
			UserService:     w.UserService,
		},
		APIKeyService:     w.APIKeyService(),
		DisableRequestLog: cfg.server.disableRequestLog,
		UserManagementURL: cfg.proxy.userManagementURL,
		DefaultLanguage:   cfg.frontend.defaultLanguage,
		LoginURL:          cfg.proxy.loginURL,
		LogoutURL:         logoutURL,
		WriteGate:         writeGate,
	})

	// 缓冲通道接收平台重载与关停信号，函数退出时撤销信号登记。
	reloadSignals := make(chan os.Signal, 1)
	notifyReloadSignals(reloadSignals)
	defer signal.Stop(reloadSignals)

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownSignals)

	if err := runServer(router, cfg.server.host, cfg.server.port, cfg.server.unixSocket, cfg.server.dataDir, metrics, cfg.metrics.metricsHost, cfg.metrics.metricsPort, w.TriggerResyncAsync, reloadSignals, shutdownSignals); err != nil {
		slog.Default().Error("Failed to start server", "error", err)
		return errLogged
	}

	return nil
}

// buildPublicAccessService 加载匿名阅读设置，仅在配置文件不存在且参数显式设置时
// 写入启动值，已有持久化设置不被普通启动参数覆盖。
func buildPublicAccessService(cmd *cli.Command, cfg *serverConfig) *publicaccess.Service {

	svc, err := publicaccess.NewSettingsManaged(cfg.server.dataDir)
	if err != nil {
		fail("Failed to load public-access configuration", "error", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.server.dataDir, "public-access.json")); os.IsNotExist(err) && cmd.IsSet("public-access") {
		if err := svc.SetEnabled(cfg.auth.publicAccess); err != nil {
			fail("Failed to save public access", "error", err)
		}
	}
	return svc
}

// runServer 先按需启动指标监听，再建立业务监听并进入生命周期等待。
// 公共命令校验拒绝 Unix 套接字，内部函数仍保留独立的套接字处理分支。
func runServer(
	router *gin.Engine,
	host, port, unixSocket, dataDir string,
	metrics *httpmetrics.HTTPMetrics,
	metricsHost, metricsPort string,
	reload func(),
	reloadSignals, shutdownSignals <-chan os.Signal,
) error {
	server := &http.Server{
		Handler:           router.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	var shutdownMetricsServer func()
	if metrics != nil {
		stopMetricsServer, _, err := startMetricsServer(metrics, metricsHost, metricsPort)
		if err != nil {
			return err
		}
		shutdownMetricsServer = stopMetricsServer
		defer shutdownMetricsServer()
	}

	if unixSocket == "" {
		listenAddr := host + ":" + port
		slog.Default().Info("Starting LeafDoc", "address", listenAddr, "data_dir", dataDir)
		listener, err := net.Listen("tcp", listenAddr)
		if err != nil {
			return err
		}
		return serveWithLifecycle(server, listener, nil, reload, reloadSignals, shutdownSignals)
	}

	listener, err := listenOnUnixSocket(unixSocket)
	if err != nil {
		return err
	}

	cleanup := func() {
		_ = listener.Close()
		_ = os.Remove(unixSocket)
	}
	defer cleanup()

	slog.Default().Info(
		"Starting LeafDoc",
		"unix_socket", unixSocket,
		"data_dir", dataDir,
		"host_port_overridden", true,
	)

	return serveWithLifecycle(server, listener, cleanup, reload, reloadSignals, shutdownSignals)
}

// startMetricsServer 只在独立监听上暴露 /metrics，返回停止函数和实际监听地址。
// 异步 Serve 异常记录日志，不通过返回值通知已运行的业务服务。
func startMetricsServer(metrics *httpmetrics.HTTPMetrics, host, port string) (func(), string, error) {
	listenAddr := host + ":" + port
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, "", err
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.HTTPHandler())

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	slog.Default().Info("Starting metrics server", "address", listener.Addr().String())

	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		slog.Default().Error("metrics server stopped unexpectedly", "error", err)
	}()

	// 关闭时先限时等待活动请求，失败再强制关闭连接并记录实际错误。
	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
				slog.Default().Error("failed to close metrics server", "error", errors.Join(err, closeErr))
				return
			}
			slog.Default().Error("failed to shut down metrics server gracefully", "error", err)
		}
	}, listener.Addr().String(), nil
}

// listenOnUnixSocket 在非 Windows 平台清理同名套接字后建立监听并设置 0660 权限。
// 权限设置失败会关闭监听并删除路径，不保留未完成初始化的资源。
func listenOnUnixSocket(socketPath string) (net.Listener, error) {
	if runtime.GOOS == "windows" {
		return nil, fmt.Errorf("unix sockets are not supported on windows")
	}
	if err := removeStaleUnixSocket(socketPath); err != nil {
		return nil, err
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(socketPath, 0660); err != nil {
		_ = listener.Close()
		_ = os.Remove(socketPath)
		return nil, err
	}

	return listener, nil
}

// removeStaleUnixSocket 只删除 Lstat 判断为套接字的现有路径，
// 普通文件和符号链接不被覆盖，不存在的路径直接通过。
func removeStaleUnixSocket(socketPath string) error {
	info, err := os.Lstat(socketPath)
	if err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("unix socket path already exists and is not a socket: %s", socketPath)
		}
		return os.Remove(socketPath)
	}
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// parseLogFormat 去除两端空白并忽略大小写，只接受 text 与 json。
func parseLogFormat(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "text":
		return "text", true
	case "json":
		return "json", true
	}
	return "", false
}

// parseBool 识别明确的开关文字，第二个返回值区分有效 false 与无法识别的输入。
func parseBool(s string) (bool, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "true", "1", "yes", "y", "on":
		return true, true
	case "false", "0", "no", "n", "off":
		return false, true
	}

	return false, false
}

// parseByteSize 使用 humanize 解析带单位的无符号字节数，拒绝零和超出 int64 的值。
// 返回的正数可直接供请求体大小限制使用，错误包含原始输入便于定位配置。
func parseByteSize(raw string) (int64, error) {
	size, err := humanize.ParseBytes(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid byte size %q: %w", raw, err)
	}
	if size == 0 {
		return 0, fmt.Errorf("byte size must be greater than zero, got %q", raw)
	}
	if size > math.MaxInt64 {
		return 0, fmt.Errorf("byte size %q is too large", raw)
	}
	return int64(size), nil
}

// mustParseByteSize 解析已经通过参数校验的大小，异常时记录设置名并直接退出。
func mustParseByteSize(raw, label string) int64 {
	size, err := parseByteSize(raw)
	if err != nil {
		fail("Invalid byte size value", "setting", label, "error", err)
	}
	return size
}

// validateHTTPRemoteUserConfig 在代理认证启用时要求至少一个非空可信来源项。
// 这里不判断 IP 或网段合法性，地址解析由认证中间件提供的解析器完成。
func validateHTTPRemoteUserConfig(enabled bool, trustedProxyIPsRaw string) error {
	if !enabled {
		return nil
	}
	hasTrustedProxy := false
	for _, entry := range strings.Split(trustedProxyIPsRaw, ",") {
		if strings.TrimSpace(entry) != "" {
			hasTrustedProxy = true
			break
		}
	}
	if !hasTrustedProxy {
		return fmt.Errorf("--trusted-proxy-ips is required when --enable-http-remote-user is set. Set it using --trusted-proxy-ips or LEAFDOC_TRUSTED_PROXY_IPS")
	}
	return nil
}

// validateHTTPRemoteUserAutoCreateConfig 要求自动建号同时启用代理认证，
// 默认角色必须属于已定义角色且不能是管理员，防止代理头直接创建管理员账号。
func validateHTTPRemoteUserAutoCreateConfig(autoCreateEnabled, remoteUserEnabled bool, defaultRole string) error {
	if !autoCreateEnabled {
		return nil
	}
	if !remoteUserEnabled {
		return fmt.Errorf("--enable-http-remote-user-auto-create requires --enable-http-remote-user to also be set")
	}
	if !auth.IsValidRole(defaultRole) {
		return fmt.Errorf("--http-remote-user-default-role %q is not a valid role", defaultRole)
	}
	if defaultRole == auth.RoleAdmin {
		return fmt.Errorf("--http-remote-user-default-role must not be %q; promote auto-created users manually instead", auth.RoleAdmin)
	}
	return nil
}

// validateSMTPConfig 在主机非空时要求发件地址、绝对公开地址和已知安全模式。
// 这里只检查公开地址协议与主机，不连接 SMTP，也不验证发件地址的邮箱格式。
func validateSMTPConfig(host, from, publicURL, security string) error {
	if host == "" {
		return nil
	}
	if from == "" {
		return fmt.Errorf("--smtp-from is required when --smtp-host is set")
	}
	if publicURL == "" {
		return fmt.Errorf("--public-url is required when --smtp-host is set")
	}
	u, err := url.Parse(publicURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("--public-url must be an absolute http(s):// URL, got %q", publicURL)
	}
	switch email.Security(security) {
	case email.SecurityNone, email.SecurityStartTLS, email.SecurityTLS:
	default:
		return fmt.Errorf("--smtp-security must be one of none, starttls, tls, got %q", security)
	}
	return nil
}

// resolveLogoutURL 优先使用 logout-url，空值时采用兼容代理登出字段。
// 第二个返回值只在实际采用兼容字段时为真，供调用方记录弃用告警。
func resolveLogoutURL(logoutURL, deprecated string) (resolved string, usedDeprecated bool) {
	if logoutURL != "" {
		return logoutURL, false
	}
	if deprecated == "" {
		return "", false
	}
	return deprecated, true
}

// validateRedirectURL 允许空值，非空时只检查忽略大小写的 HTTP 或 HTTPS 前缀。
// 它不解析主机或验证地址完整性，错误文字带上对应参数名称。
func validateRedirectURL(flagName, url string) error {
	if url == "" {
		return nil
	}
	lower := strings.ToLower(url)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return fmt.Errorf("--%s must start with http:// or https://", flagName)
	}
	return nil
}

// validateListenConfig 允许空套接字字段，所有非空值都返回同一固定 TCP 策略错误。
// hostSet 与 portSet 不改变拒绝结果。
func validateListenConfig(unixSocket string, hostSet, portSet bool) error {
	if unixSocket == "" {
		return nil
	}
	if hostSet || portSet {
		return fmt.Errorf("unix sockets are disabled; use the fixed backend TCP port")
	}
	return fmt.Errorf("unix sockets are disabled; use the fixed backend TCP port")
}

// serveWithLifecycle 同时等待 HTTP 服务结果与关停信号，独立协程串行消费重载信号。
// reload 在该协程中直接调用，生产环境回调负责异步调度；关停会等回调消费协程退出。
func serveWithLifecycle(
	server *http.Server,
	listener net.Listener,
	cleanup func(),
	reload func(),
	reloadSignals, shutdownSignals <-chan os.Signal,
) error {
	// 缓冲允许 Serve 在主循环尚未接收时报告结果，正常 Shutdown 转换为 nil。
	serveErrCh := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErrCh <- err
	}()

	stopReloads := make(chan struct{})
	var shuttingDown atomic.Bool
	var reloadWG sync.WaitGroup
	reloadWG.Add(1)
	go func() {
		defer reloadWG.Done()
		for {
			select {
			case <-stopReloads:
				return
			case sig, ok := <-reloadSignals:
				if !ok {
					return
				}
				if shuttingDown.Load() {
					return
				}
				slog.Default().Info("reload signal received: reloading from filesystem", "signal", sig.String())
				reload()
			}
		}
	}()

	// 关停分支和 defer 共用一次性函数，避免重复关闭通道或重复等待。
	stopReloader := sync.OnceFunc(func() {
		close(stopReloads)
	})
	defer stopReloader()

	waitForReloader := sync.OnceFunc(func() {
		reloadWG.Wait()
	})
	defer waitForReloader()

	for {
		select {
		case err := <-serveErrCh:
			return err
		case sig, ok := <-shutdownSignals:
			if !ok {
				// 将已关闭通道置空，避免 select 持续选中而形成空转。
				shutdownSignals = nil
				continue
			}

			slog.Default().Info("shutdown signal received: shutting down server", "signal", sig.String())
			// 阻止后续重载，先等待请求正常结束，超时则强制关闭活动连接。
			shuttingDown.Store(true)
			stopReloader()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
			err := server.Shutdown(shutdownCtx)
			cancel()
			if err != nil {
				closeErr := server.Close()
				if closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
					err = errors.Join(err, closeErr)
				}
			}

			if cleanup != nil {
				cleanup()
			}

			waitForReloader()
			serveErr := <-serveErrCh
			if err != nil {
				return err
			}
			return serveErr
		}
	}
}

// normalizeBasePath 去除两端空白与两端斜线，非空结果只加一个前导斜线。
// 根路径返回空值，不折叠内部连续斜线，也不检查路径段是否合法。
func normalizeBasePath(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "/")
	if s == "" {
		return ""
	}
	return "/" + s
}
