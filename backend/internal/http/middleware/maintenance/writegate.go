package maintenance

import (
	"net/http"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/restore"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/gin-gonic/gin"
)

// exemptPathSegmentSequences 列出维护期间仍可请求的恢复及登录会话路径片段。
// 匹配允许前面存在部署基础路径，也允许序列后面继续存在子路径。
var exemptPathSegmentSequences = [][]string{
	{"api", "admin", "restore"},
	{"api", "auth", "login"},
	{"api", "auth", "refresh-token"},
	{"api", "auth", "logout"},
}

// pathContainsSegments 按斜杠拆分路径，查找连续且逐段完全相等的目标序列。
// 搜索从每个可容纳目标序列的位置开始，因而支持前缀部署路径。
// restore-policy 不等于 restore，单个路径段的文字前缀不构成命中。
func pathContainsSegments(path string, want []string) bool {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	for start := 0; start+len(want) <= len(segments); start++ {
		match := true
		for i, w := range want {
			if segments[start+i] != w {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// isExemptPath 在任一豁免序列命中时返回 true，不要求目标序列占满整个路径。
func isExemptPath(path string) bool {
	for _, want := range exemptPathSegmentSequences {
		if pathContainsSegments(path, want) {
			return true
		}
	}
	return false
}

// WriteGateMiddleware 将普通写请求登记到恢复门禁，门禁关闭时返回 503。
// GET、HEAD、OPTIONS 及豁免路径直接通过，不登记为活动写请求。
// 成功进入的请求在后续处理结束时退出计数，使恢复流程能够等待这些请求排空。
func WriteGateMiddleware(gate *restore.WriteGate) gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
			c.Next()
			return
		}

		if isExemptPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// 判断与登记由门禁统一完成，避免在恢复开始与请求进入之间分开检查状态。
		leave, ok := gate.TryEnter()
		if !ok {
			loc, _ := sharederrors.AsLocalizedError(restore.ErrWritesDisabled)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": gin.H{
					"code":     loc.Code,
					"message":  loc.Message,
					"template": loc.Template,
				},
			})
			return
		}
		// 延迟退出也覆盖后续处理提前返回的路径，不将完成请求留在活动计数内。
		defer leave()

		c.Next()
	}
}
