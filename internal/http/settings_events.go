package http

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/perber/wiki/internal/core/settings"
)

// settingsEvents broadcasts committed writes without disclosing account or credential data.
type settingsEvents struct {
	mu      sync.Mutex
	version uint64
	clients map[chan uint64]bool
}

func newSettingsEvents() *settingsEvents {
	return &settingsEvents{version: 1, clients: make(map[chan uint64]bool)}
}
func (h *settingsEvents) publish() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.version++
	for ch := range h.clients {
		select {
		case ch <- h.version:
		default:
		}
	}
}
func (h *settingsEvents) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() < 300 && c.Writer.Status() >= 200 && c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" {
			h.publish()
		}
	}
}

// stream sends an initial snapshot version, reconnect notifications, and keepalive comments.
func (h *settingsEvents) stream(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	ch := make(chan uint64, 1)
	h.mu.Lock()
	h.clients[ch] = true
	ch <- h.version
	h.mu.Unlock()
	defer func() { h.mu.Lock(); delete(h.clients, ch); h.mu.Unlock() }()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	committed, unsubscribe := settings.Subscribe()
	defer unsubscribe()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case version := <-ch:
			version += settings.Version()
			if _, err := fmt.Fprintf(c.Writer, "id: %d\ndata: %d\n\n", version, version); err != nil {
				return
			}
			c.Writer.Flush()
		case <-committed:
			version := h.current()
			if _, err := fmt.Fprintf(c.Writer, "id: %d\ndata: %d\n\n", version, version); err != nil {
				return
			}
			c.Writer.Flush()
		case <-ticker.C:
			if _, err := fmt.Fprint(c.Writer, ": keepalive\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

// committedWrite marks mutation responses; streaming responses are never buffered.
func isSettingsStream(path string) bool { return strings.HasSuffix(path, "/api/settings/events") }

func (h *settingsEvents) current() uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.version + settings.Version()
}
