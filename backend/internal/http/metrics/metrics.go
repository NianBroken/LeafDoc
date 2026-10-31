package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// unmatchedRoute 将未命中路由的请求汇总到固定标签，避免每个未知路径产生独立时间序列。
const unmatchedRoute = "unmatched"

// HTTPMetrics 持有独立的 Prometheus 注册表、导出处理器与各业务流程的观测对象。
// HTTP 请求由中间件记录，页面保存、重构、同步及鉴权事件由业务调用方主动提交。
// 每个实例使用自己的注册表，不向 Prometheus 全局默认注册表注册指标。
type HTTPMetrics struct {
	registry              *prometheus.Registry
	buildInfo             *prometheus.GaugeVec
	requestsTotal         *prometheus.CounterVec
	requestDuration       *prometheus.HistogramVec
	requestsInFlight      prometheus.Gauge
	pagesaveDuration      *prometheus.HistogramVec
	pagesaveOpsTotal      *prometheus.CounterVec
	pagesaveFailures      *prometheus.CounterVec
	sideEffectDuration    *prometheus.HistogramVec
	sideEffectFailures    *prometheus.CounterVec
	refactorAffectedPages *prometheus.HistogramVec
	refactorMatchedLinks  *prometheus.HistogramVec
	refactorDuration      *prometheus.HistogramVec
	refactorStepDuration  *prometheus.HistogramVec
	resyncDuration        *prometheus.HistogramVec
	resyncRuns            *prometheus.CounterVec
	resyncFailures        *prometheus.CounterVec
	authLoginAttempts     *prometheus.CounterVec
	authSessions          *prometheus.CounterVec
	handler               http.Handler
}

// NewHTTPMetrics 创建并注册所有指标，将 version 作为恒为 1 的构建信息标签。
// 直方图使用各处明确列出的秒数或数量边界，向量按声明的标签组合分别累计。
// 指标名称与 Help 文本属于对外导出数据，构造过程原样交给 Prometheus。
func NewHTTPMetrics(version string) *HTTPMetrics {
	registry := prometheus.NewRegistry()

	buildInfo := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "leafdoc",
			Name:      "build_info",
			Help:      "Build information, constant 1, labeled by version.",
		},
		[]string{"version"},
	)
	buildInfo.WithLabelValues(version).Set(1)

	// 请求总数与耗时共用方法、路由模式和状态标签，便于将吞吐与延迟对应查看。
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "leafdoc",
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests by method, route, and status code.",
		},
		[]string{"method", "route", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "leafdoc",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds by method, route, and status code.",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"method", "route", "status"},
	)

	// 活动请求使用可增减的瞬时值，不按路由拆分。
	requestsInFlight := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "leafdoc",
			Name:      "http_requests_in_flight",
			Help:      "Current number of in-flight HTTP requests.",
		},
	)

	// 页面保存按操作和成功或失败分别统计次数与总耗时。
	pagesaveDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "leafdoc",
			Name:      "pagesave_duration_seconds",
			Help:      "Page save workflow duration in seconds by operation and result.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"operation", "result"},
	)

	pagesaveOpsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "leafdoc",
			Name:      "pagesave_operations_total",
			Help:      "Total number of page save workflows by operation and result.",
		},
		[]string{"operation", "result"},
	)

	pagesaveFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "leafdoc",
			Name:      "pagesave_failures_total",
			Help:      "Total number of failed page save workflows by operation and result.",
		},
		[]string{"operation", "result"},
	)

	// 保存附带处理按名称单独记录，耗时观测与失败计数由两个入口分别提交。
	sideEffectDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "leafdoc",
			Name:      "pagesave_sideeffect_duration_seconds",
			Help:      "Page save side-effect duration in seconds by operation and side effect.",
			Buckets:   []float64{0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"operation", "side_effect"},
	)

	sideEffectFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "leafdoc",
			Name:      "pagesave_sideeffect_failures_total",
			Help:      "Total number of swallowed page save side-effect failures by operation and side effect.",
		},
		[]string{"operation", "side_effect"},
	)

	// 重命名和移动同时观测涉及页面数、命中链接数与耗时，保留是否重写链接的标签。
	refactorAffectedPages := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "leafdoc",
			Name:      "refactor_affected_pages",
			Help:      "Distribution of affected page counts for rename and move refactors.",
			Buckets:   []float64{0, 1, 2, 5, 10, 25, 50, 100, 250},
		},
		[]string{"kind", "rewrite_links"},
	)

	refactorMatchedLinks := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "leafdoc",
			Name:      "refactor_matched_links",
			Help:      "Distribution of matched link counts for rename and move refactors.",
			Buckets:   []float64{0, 1, 2, 5, 10, 25, 50, 100, 250, 500},
		},
		[]string{"kind", "rewrite_links"},
	)

	refactorDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "leafdoc",
			Name:      "refactor_duration_seconds",
			Help:      "Rename and move refactor duration in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"kind", "rewrite_links"},
	)

	refactorStepDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "leafdoc",
			Name:      "refactor_step_duration_seconds",
			Help:      "Rename and move refactor duration in seconds, broken down by internal step.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"kind", "step"},
	)

	// 文件系统同步按整体结果记录运行次数、耗时和失败次数。
	resyncDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "leafdoc",
			Name:      "resync_duration_seconds",
			Help:      "Filesystem resync duration in seconds by result.",
			Buckets:   []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"result"},
	)

	resyncRuns := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "leafdoc",
			Name:      "resync_runs_total",
			Help:      "Total number of filesystem resync runs by result.",
		},
		[]string{"result"},
	)

	resyncFailures := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "leafdoc",
			Name:      "resync_failures_total",
			Help:      "Total number of failed filesystem resync runs by result.",
		},
		[]string{"result"},
	)

	// 鉴权标签直接采用调用方传入的结果或事件名称，这里不解释或验证这些值。
	authLoginAttempts := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "leafdoc",
			Name:      "auth_login_attempts_total",
			Help:      "Total number of login attempts by outcome.",
		},
		[]string{"outcome"},
	)

	authSessions := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "leafdoc",
			Name:      "auth_sessions_total",
			Help:      "Total number of session lifecycle events by event type.",
		},
		[]string{"event"},
	)

	// 标准采集器提供 go_* 与 process_* 指标，具体样本由库及运行平台提供。
	// 它们与业务指标注册在同一个独立注册表中，由同一导出入口采集。
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	// MustRegister 在描述冲突等注册错误时触发 panic，不静默丢弃无效指标。
	registry.MustRegister(
		buildInfo,
		requestsTotal,
		requestDuration,
		requestsInFlight,
		pagesaveDuration,
		pagesaveOpsTotal,
		pagesaveFailures,
		sideEffectDuration,
		sideEffectFailures,
		refactorAffectedPages,
		refactorMatchedLinks,
		refactorDuration,
		refactorStepDuration,
		resyncDuration,
		resyncRuns,
		resyncFailures,
		authLoginAttempts,
		authSessions,
	)

	return &HTTPMetrics{
		registry:              registry,
		buildInfo:             buildInfo,
		requestsTotal:         requestsTotal,
		requestDuration:       requestDuration,
		requestsInFlight:      requestsInFlight,
		pagesaveDuration:      pagesaveDuration,
		pagesaveOpsTotal:      pagesaveOpsTotal,
		pagesaveFailures:      pagesaveFailures,
		sideEffectDuration:    sideEffectDuration,
		sideEffectFailures:    sideEffectFailures,
		refactorAffectedPages: refactorAffectedPages,
		refactorMatchedLinks:  refactorMatchedLinks,
		refactorDuration:      refactorDuration,
		refactorStepDuration:  refactorStepDuration,
		resyncDuration:        resyncDuration,
		resyncRuns:            resyncRuns,
		resyncFailures:        resyncFailures,
		authLoginAttempts:     authLoginAttempts,
		authSessions:          authSessions,
		handler:               promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
	}
}

// Middleware 在进入时增加活动请求数，退出时通过 defer 减少，覆盖异常退出路径。
// 总数和耗时在 c.Next 正常返回后记录，使用路由模式而非带具体参数的请求路径。
// 若后续处理向外触发 panic，这两项正常返回后的观测不会执行。
func (m *HTTPMetrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		m.requestsInFlight.Inc()
		defer m.requestsInFlight.Dec()

		start := time.Now()
		c.Next()

		// 未匹配到路由时使用固定标签，不把任意输入路径带入指标。
		route := c.FullPath()
		if route == "" {
			route = unmatchedRoute
		}

		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method
		durationSeconds := time.Since(start).Seconds()

		m.requestsTotal.WithLabelValues(method, route, status).Inc()
		m.requestDuration.WithLabelValues(method, route, status).Observe(durationSeconds)
	}
}

// HTTPHandler 返回构造时绑定注册表的导出处理器，由路由或独立指标服务挂载。
func (m *HTTPMetrics) HTTPHandler() http.Handler {
	return m.handler
}

// ObservePageSaveWorkflow 记录一次保存流程及从 started 到调用时刻的秒数。
// 非空错误还增加失败计数；空接收者直接返回，允许业务在关闭指标时共用调用路径。
func (m *HTTPMetrics) ObservePageSaveWorkflow(operation string, err error, started time.Time) {
	if m == nil {
		return
	}
	result := resultLabel(err)
	m.pagesaveOpsTotal.WithLabelValues(operation, result).Inc()
	m.pagesaveDuration.WithLabelValues(operation, result).Observe(time.Since(started).Seconds())
	if err != nil {
		m.pagesaveFailures.WithLabelValues(operation, result).Inc()
	}
}

// ObservePageSaveSideEffect 只记录指定附带处理的耗时，不推断执行结果。
func (m *HTTPMetrics) ObservePageSaveSideEffect(operation, sideEffect string, started time.Time) {
	if m == nil {
		return
	}
	m.sideEffectDuration.WithLabelValues(operation, sideEffect).Observe(time.Since(started).Seconds())
}

// IncPageSaveSideEffectFailure 单独累计附带处理失败，空接收者不执行记录。
func (m *HTTPMetrics) IncPageSaveSideEffectFailure(operation, sideEffect string) {
	if m == nil {
		return
	}
	m.sideEffectFailures.WithLabelValues(operation, sideEffect).Inc()
}

// ObserveRefactor 在同一标签组合下记录重构影响数量和整体耗时。
// 布尔值编码为 true 或 false，数量直接转换为浮点观测值，不在这里重新计算。
func (m *HTTPMetrics) ObserveRefactor(kind string, rewriteLinks bool, affectedPages, matchedLinks int, started time.Time) {
	if m == nil {
		return
	}
	rewrite := strconv.FormatBool(rewriteLinks)
	m.refactorAffectedPages.WithLabelValues(kind, rewrite).Observe(float64(affectedPages))
	m.refactorMatchedLinks.WithLabelValues(kind, rewrite).Observe(float64(matchedLinks))
	m.refactorDuration.WithLabelValues(kind, rewrite).Observe(time.Since(started).Seconds())
}

// ObserveRefactorStep 按重构种类和内部步骤记录从 started 开始的耗时。
// 该入口与整体重构耗时独立，使调用方能够分别观测页面重写、链接刷新等阶段。
func (m *HTTPMetrics) ObserveRefactorStep(kind, step string, started time.Time) {
	if m == nil {
		return
	}
	m.refactorStepDuration.WithLabelValues(kind, step).Observe(time.Since(started).Seconds())
}

// ObserveResyncRun 累计一次同步运行及耗时，错误非空时同时累计失败。
func (m *HTTPMetrics) ObserveResyncRun(err error, started time.Time) {
	if m == nil {
		return
	}
	result := resultLabel(err)
	m.resyncRuns.WithLabelValues(result).Inc()
	m.resyncDuration.WithLabelValues(result).Observe(time.Since(started).Seconds())
	if err != nil {
		m.resyncFailures.WithLabelValues(result).Inc()
	}
}

// IncAuthLoginAttempt 将调用方给出的登录结果标签累计一次，不记录账号或令牌。
func (m *HTTPMetrics) IncAuthLoginAttempt(outcome string) {
	if m == nil {
		return
	}
	m.authLoginAttempts.WithLabelValues(outcome).Inc()
}

// IncAuthSession 累计会话生命周期事件，这是事件总数而非当前活动会话数量。
func (m *HTTPMetrics) IncAuthSession(event string) {
	if m == nil {
		return
	}
	m.authSessions.WithLabelValues(event).Inc()
}

// resultLabel 仅按错误是否为空划分 success 与 error，不暴露错误内容为标签。
func resultLabel(err error) string {
	if err != nil {
		return "error"
	}
	return "success"
}
