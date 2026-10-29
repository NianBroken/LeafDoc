package http

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// Poll responses are complete, uncached, and release their subscriptions.
func TestSettingsPollInitialAndIdle(t *testing.T) {
	testutil.ResetMySQL(t)
	h := newSettingsEvents()
	for _, cursor := range []string{"", fmt.Sprintf("%s/%d", h.instance, h.current())} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/?cursor="+url.QueryEscape(cursor), nil)
		h.poll(c, time.Millisecond)
		var body struct{ Cursor string }
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Cursor != fmt.Sprintf("%s/%d", h.instance, h.current()) || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("invalid poll response: %s", w.Body)
		}
		if len(h.clients) != 0 {
			t.Fatal("poll subscription leaked")
		}
	}
}

func TestSettingsPollCommitAndCancellation(t *testing.T) {
	testutil.ResetMySQL(t)
	for _, cancelRequest := range []bool{false, true} {
		h := newSettingsEvents()
		previous := fmt.Sprintf("%s/%d", h.instance, h.current())
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		ctx, cancel := context.WithCancel(context.Background())
		c.Request = httptest.NewRequest("GET", "/?cursor="+url.QueryEscape(previous), nil).WithContext(ctx)
		done := make(chan struct{})
		go func() { defer close(done); h.poll(c, time.Second) }()
		deadline := time.Now().Add(time.Second)
		for {
			h.mu.Lock()
			subscribed := len(h.clients) == 1
			h.mu.Unlock()
			if subscribed {
				break
			}
			if time.Now().After(deadline) {
				cancel()
				t.Fatal("poll did not subscribe")
			}
			runtime.Gosched()
		}
		if cancelRequest {
			cancel()
		} else {
			h.publish()
		}
		select {
		case <-done:
		case <-time.After(time.Second):
			cancel()
			t.Fatal("poll did not finish")
		}
		cancel()
		if len(h.clients) != 0 {
			t.Fatal("poll subscription leaked")
		}
		if cancelRequest {
			if w.Body.Len() != 0 {
				t.Fatal("canceled poll wrote a response")
			}
		} else if w.Body.String() == "" || w.Body.String() == fmt.Sprintf("{\"cursor\":%q}", previous) {
			t.Fatal("commit did not advance the cursor")
		}
	}
}
