// 本文件验证密钥记录自身的有效期判断，不访问 API 密钥数据库。
package auth

import (
	"testing"
	"time"
)

// 到期时间明确早于当前时间时，未设置撤销标记也必须返回不可用。
func TestAPIKey_IsActive_ExpiredIsInactive(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	key := &APIKey{ExpiresAt: &past}
	if key.IsActive(time.Now()) {
		t.Fatalf("expected expired key to be inactive")
	}
}

// 零值记录同时没有到期和撤销时间，单独的状态检查返回可用。
// 此断言不表示空记录能通过服务层的密钥认证。
func TestAPIKey_IsActive_NoExpiryIsActive(t *testing.T) {
	key := &APIKey{}
	if !key.IsActive(time.Now()) {
		t.Fatalf("expected key with no expiry and no revocation to be active")
	}
}
