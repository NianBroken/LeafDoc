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
	mu       sync.Mutex
	version  uint64
	clients  map[chan uint64]bool
	instance string
}

func newSettingsEvents() *settingsEvents {
	return &settingsEvents{version: 1, clients: make(map[chan uint64]bool), instance: time.Now().UTC().Format(time.RFC3339Nano)}
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
	if c.Query("poll") == "true" {
		h.poll(c, 15*time.Second)
		return
	}
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

// poll returns a complete response after a commit or a bounded idle interval.
// Subscriptions precede the cursor check so commits cannot fall between them.
func (h *settingsEvents) poll(c *gin.Context, timeout time.Duration) {
	ch := make(chan uint64, 1)
	h.mu.Lock()
	h.clients[ch] = true
	h.mu.Unlock()
	defer func() { h.mu.Lock(); delete(h.clients, ch); h.mu.Unlock() }()
	committed, unsubscribe := settings.Subscribe()
	defer unsubscribe()
	cursor := func() string { return fmt.Sprintf("%s/%d", h.instance, h.current()) }
	previous := c.Query("cursor")
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for previous == cursor() {
		select {
		case <-c.Request.Context().Done():
			return
		case <-timer.C:
			c.Header("Cache-Control", "no-store")
			c.JSON(200, gin.H{"cursor": cursor()})
			return
		case <-ch:
		case <-committed:
		}
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(200, gin.H{"cursor": cursor()})
}

// committedWrite marks mutation responses; streaming responses are never buffered.
func isSettingsStream(path string) bool { return strings.HasSuffix(path, "/api/settings/events") }

func (h *settingsEvents) current() uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.version + settings.Version()
}
