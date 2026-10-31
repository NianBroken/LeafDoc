// 本文件向客户端发送设置变化版本，不携带设置值、账号或令牌。
// HTTP 成功写响应与通用设置提交各有版本计数，事件输出使用两者之和。
package http

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/settings"
	"github.com/gin-gonic/gin"
)

// settingsEvents 保存本实例的版本和容量为一的订阅通道，
// 实例时间标识用于长轮询游标区分进程重建，不作为跨进程共享状态。
type settingsEvents struct {
	mu       sync.Mutex
	version  uint64
	clients  map[chan uint64]bool
	instance string
}

// newSettingsEvents 从版本一开始，以当前 UTC 纳秒格式文本标记本对象实例。
func newSettingsEvents() *settingsEvents {
	return &settingsEvents{version: 1, clients: make(map[chan uint64]bool), instance: time.Now().UTC().Format(time.RFC3339Nano)}
}

// publish 增加本地版本并非阻塞通知每个订阅者，通道已满则合并通知。
// 不保证每一次写响应都对应客户端收到的一条独立事件。
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

// middleware 在处理链返回后检查成功的非 GET、HEAD、OPTIONS 响应并发布通知。
// 此判断只根据状态和方法，不自行验证数据库是否真的发生变化或事务已经提交。
func (h *settingsEvents) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() < 300 && c.Writer.Status() >= 200 && c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.Request.Method != "OPTIONS" {
			h.publish()
		}
	}
}

// stream 在 poll=true 时使用长轮询，否则建立 SSE 流。注册后发送初始版本，
// 同时监听本地写响应和通用设置提交，每二十秒写一次保活注释。
// 请求取消或写入失败结束，退出时删除订阅、停止计时器并取消全局设置订阅。
func (h *settingsEvents) stream(c *gin.Context) {
	if c.Query("poll") == "true" {
		h.poll(c, 15*time.Second)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	// 初始版本在同一锁内入队，随后局部更新通道满时会合并为一次待处理事件。
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
			// 使用排队时的本地版本加读取时的设置版本，不携带变动内容。
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

// poll 先注册两类通知再比较游标，游标不同立即返回，同值最多等待指定时长。
// 通知只促使重新比较，超时返回当前游标，取消请求则不主动生成 JSON。
// 实例标识和组合版本构成游标，响应不允许缓存。
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

// isSettingsStream 仅按路径后缀识别设置事件入口，供响应翻译跳过流式缓冲。
// 不检查 HTTP 方法、查询参数或路由是否真实注册。
func isSettingsStream(path string) bool { return strings.HasSuffix(path, "/api/settings/events") }

// current 在本地锁内读取本地版本与进程设置版本之和，不修改任何计数。
func (h *settingsEvents) current() uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.version + settings.Version()
}
