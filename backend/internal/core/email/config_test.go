// 本文件核验启用判断只依赖主机，以及超时设置的零值和正数分支。
package email

import (
	"testing"
	"time"
)

// 零值配置没有主机，因此邮件集成未启用。
func TestConfig_Enabled_HostEmpty_ReturnsFalse(t *testing.T) {
	cfg := Config{}
	if cfg.Enabled() {
		t.Fatal("expected Enabled() to be false for zero-value Config")
	}
}

// 即使其他字段为空，只要主机非空，Enabled 仍返回 true。
func TestConfig_Enabled_HostSet_ReturnsTrue(t *testing.T) {
	cfg := Config{Host: "smtp.example.com"}
	if !cfg.Enabled() {
		t.Fatal("expected Enabled() to be true when Host is set")
	}
}

// 未设置超时使用集中声明的默认值。
func TestConfig_TimeoutOrDefault_Unset_ReturnsDefault(t *testing.T) {
	cfg := Config{}
	if got := cfg.TimeoutOrDefault(); got != defaultTimeout {
		t.Fatalf("expected default timeout %v, got %v", defaultTimeout, got)
	}
}

// 正数超时原样保留，不被默认值覆盖。
func TestConfig_TimeoutOrDefault_Set_ReturnsConfigured(t *testing.T) {
	cfg := Config{Timeout: 30 * time.Second}
	if got := cfg.TimeoutOrDefault(); got != 30*time.Second {
		t.Fatalf("expected configured timeout, got %v", got)
	}
}
