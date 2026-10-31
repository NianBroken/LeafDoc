// 本文件通过公开入口验证代理来源白名单，覆盖 IPv4、IPv6、网段和带端口连接地址。
// 测试直接解析地址文本，不启动代理服务器，也不向网络发送请求。
package auth_test

import (
	"testing"

	authmw "github.com/NianBroken/LeafDoc/internal/http/middleware/auth"
)

// TestParseTrustedProxies_Empty 确认空配置合法且不会自动信任回环地址。
func TestParseTrustedProxies_Empty(t *testing.T) {
	tp, err := authmw.ParseTrustedProxies("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp.IsTrusted("127.0.0.1") {
		t.Error("empty list should trust nobody")
	}
}

// TestParseTrustedProxies_SingleIP 对比命中地址与无关地址，验证精确 IP 匹配。
func TestParseTrustedProxies_SingleIP(t *testing.T) {
	tp, err := authmw.ParseTrustedProxies("127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tp.IsTrusted("127.0.0.1") {
		t.Error("127.0.0.1 should be trusted")
	}
	if tp.IsTrusted("192.168.1.1") {
		t.Error("192.168.1.1 should not be trusted")
	}
}

// TestParseTrustedProxies_MultipleIPs 覆盖空白分隔、IPv4 与 IPv6 混合配置及未列出地址。
func TestParseTrustedProxies_MultipleIPs(t *testing.T) {
	tp, err := authmw.ParseTrustedProxies("127.0.0.1, ::1, 10.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, addr := range []string{"127.0.0.1", "::1", "10.0.0.1"} {
		if !tp.IsTrusted(addr) {
			t.Errorf("%s should be trusted", addr)
		}
	}
	if tp.IsTrusted("10.0.0.2") {
		t.Error("10.0.0.2 should not be trusted")
	}
}

// TestParseTrustedProxies_CIDR 检查同网段内不同位置的地址与相邻网段的拒绝结果。
func TestParseTrustedProxies_CIDR(t *testing.T) {
	tp, err := authmw.ParseTrustedProxies("172.18.0.0/16")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tp.IsTrusted("172.18.0.1") {
		t.Error("172.18.0.1 should be trusted (within CIDR)")
	}
	if !tp.IsTrusted("172.18.255.254") {
		t.Error("172.18.255.254 should be trusted (within CIDR)")
	}
	if tp.IsTrusted("172.19.0.1") {
		t.Error("172.19.0.1 should not be trusted (outside CIDR)")
	}
}

// TestParseTrustedProxies_Mixed 确认单个地址集合和网段集合可同时参与信任判断。
func TestParseTrustedProxies_Mixed(t *testing.T) {
	tp, err := authmw.ParseTrustedProxies("127.0.0.1, ::1, 172.18.0.0/16")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tp.IsTrusted("127.0.0.1") {
		t.Error("127.0.0.1 should be trusted")
	}
	if !tp.IsTrusted("::1") {
		t.Error("::1 should be trusted")
	}
	if !tp.IsTrusted("172.18.42.10") {
		t.Error("172.18.42.10 should be trusted (CIDR)")
	}
}

// TestParseTrustedProxies_InvalidIP 确认普通文字不会被当作可解析的可信主机名。
func TestParseTrustedProxies_InvalidIP(t *testing.T) {
	_, err := authmw.ParseTrustedProxies("not-an-ip")
	if err == nil {
		t.Error("expected error for invalid IP")
	}
}

// TestParseTrustedProxies_InvalidCIDR 确认无效网段使配置解析返回错误。
func TestParseTrustedProxies_InvalidCIDR(t *testing.T) {
	_, err := authmw.ParseTrustedProxies("999.0.0.0/8")
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

// TestIsTrusted_WithPort 确认连接端口从地址中分离，不参与受信 IP 比较。
func TestIsTrusted_WithPort(t *testing.T) {
	tp, err := authmw.ParseTrustedProxies("127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 使用 HTTP 连接常见的主机与端口形式，而非裸 IP。
	if !tp.IsTrusted("127.0.0.1:54321") {
		t.Error("127.0.0.1:54321 should be trusted (port stripped)")
	}
}

// TestIsTrusted_InvalidAddr 确认无效连接地址直接拒绝，不命中已有受信地址。
func TestIsTrusted_InvalidAddr(t *testing.T) {
	tp, err := authmw.ParseTrustedProxies("127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tp.IsTrusted("not-valid") {
		t.Error("invalid address should not be trusted")
	}
}
