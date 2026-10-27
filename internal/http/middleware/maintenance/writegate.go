package maintenance

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/perber/wiki/internal/core/restore"
	sharederrors "github.com/perber/wiki/internal/core/shared/errors"
)

var exemptPathSegmentSequences = [][]string{
	{"api", "admin", "restore"},
	{"api", "auth", "login"},
	{"api", "auth", "refresh-token"},
	{"api", "auth", "logout"},
}

// pathContainsSegments reports whether path contains the given sequence of
// segments consecutively, anywhere — so it works regardless of --base-path
// (the sequence just needs to appear somewhere after whatever prefix), and
// so "restore-policy" (a different segment) never matches a check for
// "restore".
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

func isExemptPath(path string) bool {
	for _, want := range exemptPathSegmentSequences {
		if pathContainsSegments(path, want) {
			return true
		}
	}
	return false
}

// WriteGateMiddleware returns 503 for any non-GET/HEAD/OPTIONS request while
// gate is engaged (a restore is swapping live files), so a write never lands
// in a directory that's about to be renamed away.
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
		defer leave()

		c.Next()
	}
}
