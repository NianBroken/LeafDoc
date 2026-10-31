package http

import (
	"embed"
	"encoding/json"
	"html"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/assets"
	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/brandassets"
	"github.com/NianBroken/LeafDoc/internal/core/preferences"
	"github.com/NianBroken/LeafDoc/internal/core/publicaccess"
	"github.com/NianBroken/LeafDoc/internal/core/restore"
	"github.com/NianBroken/LeafDoc/internal/core/tocdisplay"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	auth_middleware "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/maintenance"
	"github.com/NianBroken/LeafDoc/internal/http/middleware/security"
	"github.com/gin-gonic/gin"
)

// frontend 将 dist 中可匹配的构建资源包含在程序中，是否对外提供页面另由开关控制。
//
//go:embed dist/**
var frontend embed.FS

// EmbedFrontend 控制是否挂载已嵌入的单页应用资源与未知路由回退。
// 只有字符串严格等于 true 时启用，构建入口通过链接参数赋值。
var EmbedFrontend = "false"

// slogWriter 将 Gin 普通输出去除两端空白后交给结构化日志的 Debug 级别。
type slogWriter struct{ logger *slog.Logger }

// Write 把每次写入作为一条日志，返回原始字节数以满足 Writer 契约。
func (sw *slogWriter) Write(p []byte) (n int, err error) {
	sw.logger.Debug(strings.TrimSpace(string(p)))
	return len(p), nil
}

// slogErrorWriter 将 Gin 错误输出转给结构化日志的 Error 级别。
type slogErrorWriter struct{ logger *slog.Logger }

// Write 去除日志两端空白，但按原始输入长度报告写入完成。
func (sew *slogErrorWriter) Write(p []byte) (n int, err error) {
	sew.logger.Error(strings.TrimSpace(string(p)))
	return len(p), nil
}

// slogRequestLogger 在后续处理正常返回后记录方法、含原始查询串的路径、状态和耗时。
// 客户端地址使用 Gin 的 ClientIP 规则，日志中的路径保留请求查询参数。
func slogRequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		c.Next()

		slog.Default().Info("http request",
			"method", c.Request.Method,
			"path", path,
			"status", c.Writer.Status(),
			"latency", time.Since(start),
			"ip", c.ClientIP(),
		)
	}
}

// disableClientCache 通过禁止存储、兼容旧缓存规则和过去的过期时间禁用客户端缓存。
func disableClientCache(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", time.Unix(0, 0).UTC().Format(http.TimeFormat))
}

// HTTPRemoteUserConfig 将路由装配所需的代理身份配置传给对应鉴权中间件。
type HTTPRemoteUserConfig struct {
	Enabled         bool
	HeaderName      string
	AutoCreate      bool   // 未知身份是否交给用户服务创建账号。
	EmailHeaderName string // 自动创建时读取邮箱的可选请求头名称。
	DefaultRole     string // 自动创建入口接收的默认角色。
	TrustedProxies  *auth_middleware.TrustedProxies
	// UserService 按请求取得当前服务，跟随在线恢复中的用户存储替换。
	UserService func() *coreauth.UserService
}

// RouterOptions 汇集路由注册器共享的服务入口、功能标记与前端配置。
// 部分字段由 NewRouter 建立当前实例后写入副本，再通过 RouterContext 交给各模块。
type RouterOptions struct {
	SettingsInstance string
	SettingsVersion  func() uint64
	Preferences      *preferences.Service
	AuthService      *coreauth.AuthService
	// PublicAccess 提供请求时读取的公开访问状态，空值在路由装配时替换为固定关闭。
	PublicAccess publicaccess.Provider
	// AlwaysShowToc 提供前端目录持续显示设置，空值替换为固定关闭的提供器。
	AlwaysShowToc           tocdisplay.Provider
	EditorLimit             int                      // 管理员与编辑者总数限制，供配置接口向前端提供。
	InjectCodeInHeader      string                   // 直接插入页面 head 的原始 HTML 或脚本，不执行转义。
	CustomStylesheet        string                   // 自定义 CSS 路径，缺少已解析路径时相对存储目录规范化。
	AllowInsecure           bool                     // Cookie 策略是否接受普通 HTTP 连接。
	AccessTokenTimeout      time.Duration            // 访问令牌 Cookie 的有效期。
	RefreshTokenTimeout     time.Duration            // 刷新令牌 Cookie 的有效期。
	HideLinkMetadataSection bool                     // 交给前端的链接元数据区域隐藏设置。
	BasePath                string                   // 路由共同前缀及前端资源地址前缀。
	MaxAssetUploadSizeBytes int64                    // 上传请求限制的字节数，非正数时采用资源模块默认值。
	EnableRevision          bool                     // 交给路由及配置接口的页面历史功能开关。
	EnableLinkRefactor      bool                     // 交给前端的链接重构功能开关。
	EnableAPIKeyManagement  bool                     // 交给路由及前端的 API 密钥管理开关。
	Metrics                 *httpmetrics.HTTPMetrics // 非空时挂载 HTTP 请求指标中间件。
	SnapshotEnabled         bool                     // 配置接口提供的完整快照功能状态。
	SMTPEnabled             bool                     // 配置接口提供的邮件服务配置状态。
	HTTPRemoteUser          HTTPRemoteUserConfig     // 受信代理通过请求头声明身份的配置。
	APIKeyService           *coreauth.APIKeyService  // 非空时挂载全局 Bearer 密钥解析。
	DisableRequestLog       bool                     // 是否跳过逐请求访问日志中间件。
	UserManagementURL       string                   // 前端外部用户管理入口地址。
	DefaultLanguage         string                   // 供前端匹配已支持语言的默认语言代码。
	LoginURL                string                   // 供前端替代内置登录表单的跳转地址。
	LogoutURL               string                   // 供前端登出后使用的跳转地址。
	WriteGate               *restore.WriteGate       // 非空时挂载恢复期间的普通写请求门禁。
}

// FrontendConfig 提供首屏 HTML 品牌数据及自定义样式文件的位置。
// 品牌回调在每个 HTML 请求中调用，模板本身只在路由构造时读取一次。
type FrontendConfig struct {
	// GetSiteName 返回当前站点名，空回调或空结果使用 LeafDoc。
	GetSiteName func() string
	// GetFaviconFile 返回品牌图标文件名，空结果使用内置默认图标。
	GetFaviconFile func() string
	// GetLogoFile 提供首屏导航标识文件名，作为品牌元数据注入页面。
	GetLogoFile func() string
	// CustomStylesheetPath 非空时直接采用，调用方负责事先解析和验证。
	// 为空时还会尝试 RouterOptions.CustomStylesheet，不是无条件关闭样式。
	CustomStylesheetPath string
	// StorageDir 是规范化 RouterOptions.CustomStylesheet 时使用的存储根目录。
	StorageDir string
}

// NewRouter 依次装配全局日志与恢复处理、内置资源、设置事件、鉴权及业务注册器。
// Gin 按注册路由时已有的处理链保存中间件，较早注册的默认资源和事件流不继承后挂载项。
// 可选自定义样式与嵌入前端在业务路由之后挂载，未知页面回退由引擎 NoRoute 处理。
func NewRouter(registrars []RouteRegistrar, frontendCfg FrontendConfig, opts RouterOptions) *gin.Engine {
	if opts.MaxAssetUploadSizeBytes <= 0 {
		opts.MaxAssetUploadSizeBytes = assets.DefaultMaxUploadSizeBytes
	}

	if opts.PublicAccess == nil {
		opts.PublicAccess = publicaccess.Fixed(false)
	}

	if opts.AlwaysShowToc == nil {
		opts.AlwaysShowToc = tocdisplay.Fixed(false)
	}

	// Gin 模式及默认日志 Writer 属于包级设置，构造路由会更新这些全局值。
	gin.SetMode(gin.ReleaseMode)

	gin.DefaultWriter = &slogWriter{logger: slog.Default().With("component", "gin")}
	gin.DefaultErrorWriter = &slogErrorWriter{logger: slog.Default().With("component", "gin")}

	authCookies := auth_middleware.NewAuthCookies(opts.AllowInsecure, opts.AccessTokenTimeout, opts.RefreshTokenTimeout)
	csrfCookie := security.NewCSRFCookie(opts.AllowInsecure, 3*24*time.Hour)

	engine := gin.New()
	if !opts.DisableRequestLog {
		engine.Use(slogRequestLogger())
	}
	if opts.Metrics != nil {
		engine.Use(opts.Metrics.Middleware())
	}
	engine.Use(gin.RecoveryWithWriter(gin.DefaultErrorWriter))
	base := engine.Group(opts.BasePath)
	// 默认品牌资源先于基础路由的鉴权中间件注册，不要求前端构建或登录。
	builtinFS, err := fs.Sub(brandassets.Files, "files")
	if err != nil {
		panic(err)
	}
	defaults := base.Group("/defaults")
	defaults.Use(func(c *gin.Context) { disableClientCache(c); c.Next() })
	defaults.StaticFS("", http.FS(builtinFS))
	// 每个路由实例生成自己的设置事件标识与版本函数，覆盖选项副本中的对应值。
	events := newSettingsEvents()
	opts.SettingsInstance = events.instance
	opts.SettingsVersion = events.current
	if opts.Preferences != nil {
		base.Use(localizeJSON())
	}
	base.Use(events.middleware())
	if opts.AuthService != nil {
		base.Use(auth_middleware.PasswordChangeGate(opts.AuthService, authCookies))
	}
	// 事件流在写门禁与代理、密钥身份注入之前注册，沿用此前的处理链。
	base.GET("/api/settings/events", events.stream)

	if opts.WriteGate != nil {
		base.Use(maintenance.WriteGateMiddleware(opts.WriteGate))
	}

	if opts.HTTPRemoteUser.Enabled {
		base.Use(auth_middleware.InjectRemoteUser(auth_middleware.RemoteUserConfig{
			Enabled:         opts.HTTPRemoteUser.Enabled,
			HeaderName:      opts.HTTPRemoteUser.HeaderName,
			AutoCreate:      opts.HTTPRemoteUser.AutoCreate,
			EmailHeaderName: opts.HTTPRemoteUser.EmailHeaderName,
			DefaultRole:     opts.HTTPRemoteUser.DefaultRole,
			TrustedProxies:  opts.HTTPRemoteUser.TrustedProxies,
			UserService:     opts.HTTPRemoteUser.UserService,
		}))
	}

	if opts.APIKeyService != nil {
		// 密钥解析覆盖后续注册的业务路由，实际权限由有效用户角色及路由守卫判断。
		// 成功密钥标记允许 CSRF 豁免，凭据管理路由通过 RequireCookieSession 拒绝该标记。
		base.Use(auth_middleware.InjectAPIKeyUser(auth_middleware.APIKeyConfig{
			Service: opts.APIKeyService,
			// 每次密钥尝试先登记，成功或存储暂停时清空该地址全部记录。
			// 连续失败达到二十次后，在解析密钥前返回 429，因此同地址的有效密钥也受限。
			RateLimiter: security.NewKeyedLimiter(20, 5*time.Minute, true),
		}))
	}

	ctx := RouterContext{
		Engine:      engine,
		Base:        base,
		AuthCookies: authCookies,
		CSRFCookie:  csrfCookie,
		Opts:        opts,
	}

	// 注册器按传入顺序共享上下文，分别声明自己的业务路由与权限要求。
	for _, r := range registrars {
		r.RegisterRoutes(ctx)
	}

	// 优先采用调用方已验证的样式路径；缺失时才规范化选项中的路径。
	// 规范化失败记录错误并继续构造路由，不挂载失败的样式入口。
	customStylesheetPath := frontendCfg.CustomStylesheetPath
	if customStylesheetPath == "" && opts.CustomStylesheet != "" {
		resolved, err := NormalizeCustomStylesheetPath(frontendCfg.StorageDir, opts.CustomStylesheet)
		if err != nil {
			slog.Default().Error("custom stylesheet disabled", "error", err)
		} else {
			customStylesheetPath = resolved
		}
	}

	// 路径非空时挂载样式路由，每次请求重新检查文件，区分缺失和其他系统错误。
	if customStylesheetPath != "" {
		cssPath := customStylesheetPath
		base.GET("/custom.css", func(c *gin.Context) {
			if _, err := os.Stat(cssPath); os.IsNotExist(err) {
				c.Status(http.StatusNotFound)
				return
			} else if err != nil {
				slog.Default().Error("error checking custom stylesheet existence", "error", err, "path", cssPath)
				c.Status(http.StatusInternalServerError)
				return
			}
			c.Header("Content-Type", "text/css; charset=utf-8")
			c.File(cssPath)
		})
	}

	// 开关开启时挂载静态资源，并为满足路径与方法规则的未知请求返回单页应用。
	if EmbedFrontend == "true" {
		fsys, err := fs.Sub(frontend, "dist")
		if err != nil {
			panic("failed to create sub FS: " + err.Error())
		}
		staticFS, err := fs.Sub(frontend, "dist/static")
		if err != nil {
			panic("failed to create sub FS: " + err.Error())
		}

		base.StaticFS("/static", http.FS(staticFS))

		// 模板与读取错误均保留到请求处理阶段，后续请求不重复读取嵌入文件。
		indexData, indexErr := fs.ReadFile(fsys, "index.html")
		indexTemplate := string(indexData)

		engine.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if opts.BasePath != "" {
				if path != opts.BasePath && !strings.HasPrefix(path, opts.BasePath+"/") {
					c.String(http.StatusNotFound, "Page not found")
					return
				}
				path = strings.TrimPrefix(path, opts.BasePath)
				if path == "" {
					path = "/"
				}
			}

			// 排除规则按文本前缀检查，因此以 /api 等文本开头的未知页面也不进入回退。
			if c.Request.Method == http.MethodGet &&
				!strings.HasPrefix(path, "/api") &&
				!strings.HasPrefix(path, "/assets") &&
				!strings.HasPrefix(path, "/static") &&
				!strings.HasPrefix(path, "/branding") {

				c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
				if indexErr != nil {
					c.Status(http.StatusNotFound)
					return
				}

				siteName := "LeafDoc"
				if frontendCfg.GetSiteName != nil {
					if name := frontendCfg.GetSiteName(); name != "" {
						siteName = name
					}
				}
				faviconFile := ""
				if frontendCfg.GetFaviconFile != nil {
					faviconFile = frontendCfg.GetFaviconFile()
				}

				logoFile := ""
				if frontendCfg.GetLogoFile != nil {
					logoFile = frontendCfg.GetLogoFile()
				}
				// 字符串映射序列化后再作 HTML 属性转义，站点名与路径占位符也逐项转义。
				brandingData, _ := json.Marshal(map[string]string{"siteName": siteName, "logoFile": logoFile, "faviconFile": faviconFile})
				doc := indexTemplate
				escapedBasePath := html.EscapeString(opts.BasePath)
				doc = strings.ReplaceAll(doc, "{{__SITE_NAME__}}", html.EscapeString(siteName))
				doc = strings.ReplaceAll(doc, "{{__BASE_PATH__}}", escapedBasePath)
				doc = strings.ReplaceAll(doc, "{{__FAVICON_HREF__}}", html.EscapeString(BuildFrontendFaviconHref(opts.BasePath, faviconFile)))
				// 将模板内双引号开头的 ./static/ 引用转为基础路径下的绝对地址，
				// 深层页面因此不会将首屏资源解释为当前页面目录下的相对路径。
				doc = strings.ReplaceAll(doc, `"./static/`, `"`+escapedBasePath+`/static/`)

				doc = injectIntoHead(doc, `<meta name="leafdoc-branding" content="`+html.EscapeString(string(brandingData))+`">`)
				disableClientCache(c)
				doc = injectIntoHead(doc, buildCustomStylesheetTag(opts.BasePath, customStylesheetPath))

				// 配置中的头部代码原样插入，属于受配置方控制的 HTML，不进行内容清洗。
				if opts.InjectCodeInHeader != "" {
					doc = injectIntoHead(doc, opts.InjectCodeInHeader)
				}
				c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(doc))
			} else {
				c.String(http.StatusNotFound, "Page not found")
			}
		})
	}

	return engine
}

// BuildFrontendFaviconHref 按是否提供品牌文件名拼接地址，并保留传入的部署前缀。
// 函数不检查文件存在性，也不在这里转义路径，HTML 输出方负责属性转义。
func BuildFrontendFaviconHref(basePath, faviconFile string) string {
	if faviconFile != "" {
		return basePath + "/branding/" + faviconFile
	}

	return basePath + "/defaults/" + brandassets.FaviconFile
}

// NormalizeCustomStylesheetPath 去除路径空白，要求扩展名为 CSS 并按存储目录解析相对路径。
// 使用清理后的相对关系拒绝向父目录越界，不检查文件存在性，也不解析符号链接目标。
// 空路径返回空值且无错误，其他有效路径返回清理后的路径，不保证它是绝对路径。
func NormalizeCustomStylesheetPath(storageDir, customStylesheet string) (string, error) {
	cssPath := strings.TrimSpace(customStylesheet)
	if cssPath == "" {
		return "", nil
	}

	if strings.ToLower(filepath.Ext(cssPath)) != ".css" {
		return "", os.ErrPermission
	}

	if !filepath.IsAbs(cssPath) {
		cssPath = filepath.Join(storageDir, cssPath)
	}

	cleanStorageDir := filepath.Clean(storageDir)
	cleanCSSPath := filepath.Clean(cssPath)

	relPath, err := filepath.Rel(cleanStorageDir, cleanCSSPath)
	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return "", os.ErrPermission
	}

	return cleanCSSPath, nil
}

// buildCustomStylesheetTag 在样式路径非空时引用固定的 custom.css 路由，而非暴露磁盘路径。
// 基础路径原样拼接，不在这个助手内部转义。
func buildCustomStylesheetTag(basePath, customStylesheet string) string {
	if strings.TrimSpace(customStylesheet) == "" {
		return ""
	}
	return `<link rel="stylesheet" href="` + basePath + `/custom.css">`
}

// injectIntoHead 在第一个大小写完全一致的 </head> 前插入原始片段及缩进换行。
// 空白片段不处理，替换后文本未改变时记录警告并返回原文本。
func injectIntoHead(html, snippet string) string {
	if strings.TrimSpace(snippet) == "" {
		return html
	}
	newHTML := strings.Replace(html, "</head>", "  "+snippet+"\n  </head>", 1)
	if newHTML == html {
		slog.Default().Warn("could not inject code into header", "reason", "</head> tag not found")
	}
	return newHTML
}
