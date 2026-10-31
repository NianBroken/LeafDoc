package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestRateLimiter_NewKey 确认首次出现的连接地址能够登记并正常进入路由。
func TestRateLimiter_NewKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(3, time.Minute, false)

	// 将限制器放在路由之前，路由返回 200 作为请求已到达的证据。
	router := gin.New()
	router.Use(limiter)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 显式设置连接地址，使测试键与内存请求默认地址无关。
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestRateLimiter_ExceedsLimit 确认同一地址在一分钟内的前三次通过、第四次返回 429。
func TestRateLimiter_ExceedsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(3, time.Minute, false)

	router := gin.New()
	router.Use(limiter)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 成功清空关闭，连续三次成功响应仍保留在请求计数中。
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.2:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// 第四次请求使用同一地址，不能因路由自身返回成功而逃过上限。
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:1234"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", w.Code)
	}
}

// TestRateLimiter_ReleasesLockAfterLimit 触发超限后从另一地址请求，验证拒绝路径释放互斥锁。
func TestRateLimiter_ReleasesLockAfterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(1, time.Minute, false)

	router := gin.New()
	router.Use(limiter)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.4:1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.4:1234"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Expected status 429, got %d", w.Code)
	}

	// 缓冲通道使测试超时结束时，工作协程仍能提交结果而不阻塞在发送操作上。
	done := make(chan int, 1)

	go func() {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.5:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		done <- w.Code
	}()

	// 等待不超过两秒，测试整体截止时间更近时使用其剩余时间的一半。
	timeout := 2 * time.Second
	if deadline, ok := t.Deadline(); ok {
		if remaining := time.Until(deadline) / 2; remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case code := <-done:
		if code != http.StatusOK {
			t.Fatalf("Expected status 200 for different key after limit hit, got %d", code)
		}
	case <-timer.C:
		t.Fatal("Request blocked after limit hit; mutex was not released")
	}
}

// TestKeyedLimiter_AllowAndDeny 直接测试每个键独立计数，同键第三次拒绝不影响另一键。
func TestKeyedLimiter_AllowAndDeny(t *testing.T) {
	kl := NewKeyedLimiter(2, time.Minute, false)

	if !kl.Allow("k1") {
		t.Fatalf("expected first hit to be allowed")
	}
	if !kl.Allow("k1") {
		t.Fatalf("expected second hit to be allowed")
	}
	if kl.Allow("k1") {
		t.Fatalf("expected third hit within the limit window to be denied")
	}
	if !kl.Allow("k2") {
		t.Fatalf("expected a different key to be unaffected by k1's count")
	}
}

// TestKeyedLimiter_NotifyResultResetsOnSuccess 确认成功通知清空历史并允许下一次登记。
func TestKeyedLimiter_NotifyResultResetsOnSuccess(t *testing.T) {
	kl := NewKeyedLimiter(1, time.Minute, true)

	if !kl.Allow("k1") {
		t.Fatalf("expected first hit to be allowed")
	}
	kl.NotifyResult("k1", true)

	if !kl.Allow("k1") {
		t.Fatalf("expected hit to be allowed again after a successful result reset the count")
	}
}

// TestKeyedLimiter_NotifyResultDoesNotResetOnFailure 确认失败通知保留已满计数。
func TestKeyedLimiter_NotifyResultDoesNotResetOnFailure(t *testing.T) {
	kl := NewKeyedLimiter(1, time.Minute, true)

	if !kl.Allow("k1") {
		t.Fatalf("expected first hit to be allowed")
	}
	kl.NotifyResult("k1", false)

	if kl.Allow("k1") {
		t.Fatalf("expected the count to remain after a failed result")
	}
}

// TestRateLimiter_WindowExpires 用短窗口验证历史过期后同一地址重新获得请求额度。
func TestRateLimiter_WindowExpires(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 使用 100 毫秒窗口，使真实时间等待能够覆盖过期清理而不延长到分钟级。
	limiter := NewRateLimiter(2, 100*time.Millisecond, false)

	router := gin.New()
	router.Use(limiter)
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 两次请求占满该地址的窗口额度。
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.3:1234"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	// 等待 150 毫秒越过窗口，再通过 Allow 触发历史清理。
	time.Sleep(150 * time.Millisecond)

	// 使用同一连接地址重新请求，成功响应证明旧记录已不参与计数。
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.3:1234"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 after window expired, got %d", w.Code)
	}
}
