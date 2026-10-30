package auth_test

import (
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
	"github.com/gin-gonic/gin"
)

type authFixture struct {
	auth  *coreauth.AuthService
	close func() error
}

func createTestAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	testutil.ResetMySQL(t)

	storageDir := t.TempDir()
	userStore, err := coreauth.NewUserStore(storageDir)
	if err != nil {
		t.Fatalf("Failed to create user store: %v", err)
	}
	t.Cleanup(func() {
		if err := userStore.Close(); err != nil {
			t.Error(err)
		}
	})
	sessionStore, err := coreauth.NewSessionStore(storageDir)
	if err != nil {
		_ = userStore.Close()
		t.Fatalf("Failed to create session store: %v", err)
	}
	t.Cleanup(func() {
		if err := sessionStore.Close(); err != nil {
			t.Error(err)
		}
	})

	userService := coreauth.NewUserService(userStore)
	if err := userService.InitDefaultAdmin("", "", "adminpassword"); err != nil {
		_ = sessionStore.Close()
		_ = userStore.Close()
		t.Fatalf("Failed to init default admin: %v", err)
	}

	sessions := coreauth.NewSessionManager(sessionStore, "test-secret-key-for-unit-tests-1", 15*time.Minute, 7*24*time.Hour)
	return &authFixture{
		auth: coreauth.NewAuthService(userService, sessions),
		close: func() error {
			if err := sessionStore.Close(); err != nil {
				_ = userStore.Close()
				return err
			}
			return userStore.Close()
		},
	}
}

func TestRequireAuth_WithInvalidUserContext_NilUser(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", (*coreauth.User)(nil))
		c.Next()
	})
	router.Use(authmw.RequireAuth(nil, authCookies))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for nil user context, got %d", w.Code)
	}

	expectedBody := `{"error":"Invalid user context"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestRequireAuth_WithInvalidUserContext_WrongType(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", "not-a-user")
		c.Next()
	})
	router.Use(authmw.RequireAuth(nil, authCookies))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for invalid user type, got %d", w.Code)
	}

	expectedBody := `{"error":"Invalid user context"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestRequireAuth_WithAuthEnabled_ValidToken(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	fixture := createTestAuthFixture(t)
	defer func() {
		if err := fixture.close(); err != nil {
			t.Fatalf("close auth fixture: %v", err)
		}
	}()

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

	// Login to get a valid token
	authToken, err := fixture.auth.Login("admin", "adminpassword")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	router := gin.New()

	// Require a validated session or an authenticated upstream identity.
	router.Use(authmw.RequireAuth(fixture.auth, authCookies))

	router.GET("/test", func(c *gin.Context) {
		userValue, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
			return
		}

		u, ok := userValue.(*coreauth.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user type"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"username": u.Username,
			"role":     u.Role,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "leafdoc_at",
		Value: authToken.Token,
	})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d - %s", w.Code, w.Body.String())
	}

	expectedBody := `{"role":"admin","username":"admin"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestRequireAuth_WithAuthEnabled_MissingToken(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	fixture := createTestAuthFixture(t)
	defer func() {
		if err := fixture.close(); err != nil {
			t.Fatalf("close auth fixture: %v", err)
		}
	}()

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

	router := gin.New()

	// Require a validated session or an authenticated upstream identity.
	router.Use(authmw.RequireAuth(fixture.auth, authCookies))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 when no token provided, got %d", w.Code)
	}

	expectedBody := `{"error":"Missing or invalid access token"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestRequireAuth_WithAuthEnabled_NilAuthService(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

	router := gin.New()
	router.Use(authmw.RequireAuth(nil, authCookies))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "leafdoc_at",
		Value: "some-token",
	})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when auth service is nil, got %d", w.Code)
	}

	expectedBody := `{"error":"Authentication service unavailable"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestRequireAuth_WithAuthEnabled_InvalidToken(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	fixture := createTestAuthFixture(t)
	defer func() {
		if err := fixture.close(); err != nil {
			t.Fatalf("close auth fixture: %v", err)
		}
	}()

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

	router := gin.New()

	// Require a validated session or an authenticated upstream identity.
	router.Use(authmw.RequireAuth(fixture.auth, authCookies))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "leafdoc_at",
		Value: "invalid-token-123",
	})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 when invalid token provided, got %d", w.Code)
	}

	expectedBody := `{"error":"Invalid or expired token"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestRequireAuth_WithAuthEnabled_UserSetInContext(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	fixture := createTestAuthFixture(t)
	defer func() {
		if err := fixture.close(); err != nil {
			t.Fatalf("close auth fixture: %v", err)
		}
	}()

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

	// Login to get a valid token
	authToken, err := fixture.auth.Login("admin", "adminpassword")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	router := gin.New()

	// Require a validated session or an authenticated upstream identity.
	router.Use(authmw.RequireAuth(fixture.auth, authCookies))

	userSetInContext := false

	router.GET("/test", func(c *gin.Context) {
		_, exists := c.Get("user")
		if exists {
			userSetInContext = true
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{
		Name:  "leafdoc_at",
		Value: authToken.Token,
	})
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !userSetInContext {
		t.Error("Expected user to be set in context")
	}
}

func TestRequireAuth_NextNotCalledOnFailure(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	fixture := createTestAuthFixture(t)
	defer func() {
		if err := fixture.close(); err != nil {
			t.Fatalf("close auth fixture: %v", err)
		}
	}()

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

	router := gin.New()

	// Require a validated session or an authenticated upstream identity.
	router.Use(authmw.RequireAuth(fixture.auth, authCookies))

	nextCalled := false

	router.Use(func(c *gin.Context) {
		nextCalled = true
		c.Next()
	})

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	if nextCalled {
		t.Error("Expected Next() not to be called when authentication fails")
	}
}

func TestRequireAuth_ComprehensiveScenarios(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		provideToken   bool
		validToken     bool
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "valid token - should pass",
			provideToken:   true,
			validToken:     true,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "no token - should fail",
			provideToken:   false,
			validToken:     false,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Missing or invalid access token",
		},
		{
			name:           "invalid token - should fail",
			provideToken:   true,
			validToken:     false,
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Invalid or expired token",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fixture := createTestAuthFixture(t)
			defer func() {
				if err := fixture.close(); err != nil {
					t.Fatalf("close auth fixture: %v", err)
				}
			}()

			authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)

			router := gin.New()

			// Apply RequireAuth
			router.Use(authmw.RequireAuth(fixture.auth, authCookies))

			router.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			req := httptest.NewRequest("GET", "/test", nil)

			// Add token if needed
			if tc.provideToken {
				var token string
				if tc.validToken {
					authToken, err := fixture.auth.Login("admin", "adminpassword")
					if err != nil {
						t.Fatalf("Failed to login: %v", err)
					}
					token = authToken.Token
				} else {
					token = "invalid-token"
				}
				req.AddCookie(&http.Cookie{
					Name:  "leafdoc_at",
					Value: token,
				})
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d - %s", tc.expectedStatus, w.Code, w.Body.String())
			}

			if tc.expectedError != "" {
				expectedBody := `{"error":"` + tc.expectedError + `"}`
				if w.Body.String() != expectedBody {
					t.Errorf("Expected error %s, got %s", expectedBody, w.Body.String())
				}
			}
		})
	}
}

func TestOptionalAuth_NoToken_PassesThrough(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)
	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)
	router := gin.New()
	router.Use(authmw.OptionalAuth(nil, authCookies))
	router.GET("/test", func(c *gin.Context) {
		_, exists := c.Get("user")
		if exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected user in context"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": nil})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for no token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOptionalAuth_ValidToken_SetsUser(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)
	fixture := createTestAuthFixture(t)
	defer func() {
		if err := fixture.close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	}()

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)
	authToken, err := fixture.auth.Login("admin", "adminpassword")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	router := gin.New()
	router.Use(authmw.OptionalAuth(fixture.auth, authCookies))
	router.GET("/test", func(c *gin.Context) {
		v, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not in context"})
			return
		}
		u, ok := v.(*coreauth.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "wrong type"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"username": u.Username})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "leafdoc_at", Value: authToken.Token})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if w.Body.String() != `{"username":"admin"}` {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestOptionalAuth_InvalidToken_PassesThrough(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)
	fixture := createTestAuthFixture(t)
	defer func() {
		if err := fixture.close(); err != nil {
			t.Fatalf("close: %v", err)
		}
	}()

	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)
	router := gin.New()
	router.Use(authmw.OptionalAuth(fixture.auth, authCookies))
	router.GET("/test", func(c *gin.Context) {
		_, exists := c.Get("user")
		if exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "unexpected user for invalid token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"user": nil})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "leafdoc_at", Value: "invalid-token"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for invalid token, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOptionalAuth_NilAuthService_WithToken_Returns500(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)
	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)
	router := gin.New()
	router.Use(authmw.OptionalAuth(nil, authCookies))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "leafdoc_at", Value: "some-token"})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for nil authService with token, got %d: %s", w.Code, w.Body.String())
	}
	expected := `{"error":"Authentication service unavailable"}`
	if w.Body.String() != expected {
		t.Errorf("expected %s, got %s", expected, w.Body.String())
	}
}

func TestOptionalAuth_UserAlreadyInContext_ShortCircuits(t *testing.T) {
	testutil.ResetMySQL(t)
	gin.SetMode(gin.TestMode)
	authCookies := authmw.NewAuthCookies(true, time.Hour, time.Hour*24)
	injected := &coreauth.User{ID: "proxy-user", Username: "proxy", Role: coreauth.RoleViewer}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user", injected)
		c.Next()
	})
	router.Use(authmw.OptionalAuth(nil, authCookies))
	router.GET("/test", func(c *gin.Context) {
		v, _ := c.Get("user")
		u := v.(*coreauth.User)
		c.JSON(http.StatusOK, gin.H{"username": u.Username})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != `{"username":"proxy"}` {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}
