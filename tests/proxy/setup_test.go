// Package e2eproxy contains integration tests that verify reverse-proxy
// authentication works end-to-end with a real nginx container in front of
// LeafDoc.  The tests expect the Docker Compose stack in this directory to
// be running before they are executed.
//
// Run with:
//
//	node run.mjs
package e2eproxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// proxyURL is the nginx entry point for the default-config stack (auto-create
// off — all requests go through the proxy).
var proxyURL string

// autoCreateProxyURL is the nginx entry point for the second stack, which runs
// LeafDoc with --enable-http-remote-user-auto-create=true. Kept as a separate
// stack/port so the default stack's "unknown user -> 401" behavior stays
// covered end-to-end, rather than being replaced by the auto-create behavior.
var autoCreateProxyURL string

// directURL is the LeafDoc port exposed directly, used to verify that the
// same Remote-User header is rejected when it does not come from nginx.
// NOTE: the direct port is intentionally not exposed in docker-compose so
// that test #4 (bypass attempt) must go through the proxy.  We test the
// untrusted-IP scenario by sending to the proxy without X-Test-User instead.
var directURL string

func TestMain(m *testing.M) {
	if os.Getenv("E2E_PROXY_RUN_ID") == "" {
		fmt.Fprintln(os.Stderr, "Run proxy tests with node run.mjs to ensure database isolation")
		os.Exit(1)
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "internal", "core", "serviceports", "ports.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var ports map[string]int
	if err = json.Unmarshal(raw, &ports); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	proxyURL = fmt.Sprintf("http://127.0.0.1:%d", ports["proxyTest"])
	autoCreateProxyURL = fmt.Sprintf("http://127.0.0.1:%d", ports["proxyAutoCreateTest"])
	directURL = envOr("E2E_DIRECT_URL", "")

	if err := waitReachable(proxyURL+"/api/config", 60*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "LeafDoc proxy stack not reachable at %s: %v\n", proxyURL, err)
		fmt.Fprintln(os.Stderr, "Run the isolated stack with: node run.mjs")
		os.Exit(1)
	}
	if err := waitReachable(autoCreateProxyURL+"/api/config", 60*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "LeafDoc auto-create proxy stack not reachable at %s: %v\n", autoCreateProxyURL, err)
		fmt.Fprintln(os.Stderr, "Run the isolated stack with: node run.mjs")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func waitReachable(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) //nolint:gosec,noctx
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("timed out after %s", timeout)
}
