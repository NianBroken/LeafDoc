// 会话测试验证逐步接口不能由一次性提交、跨站请求或过期状态绕过。
package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInstallationStepsEnforceOrder(t *testing.T) {
	called := false
	handler := installationSteps(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }), "/docs", "01234567890123456789012345678901")
	send := func(path, body string, cookie *http.Cookie, marker bool) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/docs/api/setup"+path, strings.NewReader(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		if marker {
			req.Header.Set("X-LeafDoc-Setup", "1")
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	if rec := send("", `{}`, nil, true); rec.Code != 401 {
		t.Fatal(rec.Code, rec.Body.String())
	}
	if rec := send("/token", `{"setupToken":"wrong"}`, nil, true); rec.Code != 403 {
		t.Fatal(rec.Code)
	}
	if rec := send("/token", `{"setupToken":"01234567890123456789012345678901"}`, nil, false); rec.Code != 403 {
		t.Fatal("missing origin marker accepted")
	}
	rec := send("/token", `{"setupToken":"01234567890123456789012345678901"}`, nil, true)
	if rec.Code != 200 || len(rec.Result().Cookies()) != 1 {
		t.Fatal(rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()[0]
	if !cookie.HttpOnly || cookie.Path != "/docs/" {
		t.Fatal("invalid session cookie")
	}
	for _, path := range []string{"/credentials", "/database", "/admin", ""} {
		if result := send(path, `{"revision":1}`, cookie, true); result.Code != 409 {
			t.Fatal(path, result.Code)
		}
	}
	if result := send("/address", `{"revision":0}`, cookie, true); result.Code != 409 {
		t.Fatal("stale revision accepted")
	}
	if result := send("/address", `{"revision":1,"host":"http://localhost","port":3306}`, cookie, true); result.Code != 400 {
		t.Fatal("invalid host accepted")
	}
	if called {
		t.Fatal("final handler invoked before validation")
	}
}
