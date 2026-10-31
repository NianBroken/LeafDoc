// 本文件维护进程内按用户 ID 隔离的登录尝试计数与临时锁定状态。
package auth

import (
	"sync"
	"time"
)

// 连续计数达到五次时设置十五分钟锁定，状态不持久化到数据库。
const (
	loginMaxFailures  = 5
	loginLockDuration = 15 * time.Minute
)

// loginAttemptEntry 保存未被成功登录清除的尝试数及锁定截止时间。
type loginAttemptEntry struct {
	failures    int
	lockedUntil time.Time
}

// loginAttemptTracker 用同一互斥锁保护所有用户记录，读取检查与计数写入不可分割。
type loginAttemptTracker struct {
	mu      sync.Mutex
	entries map[string]*loginAttemptEntry
}

// newLoginAttemptTracker 创建空计数表，不启动定时清理任务。
func newLoginAttemptTracker() *loginAttemptTracker {
	return &loginAttemptTracker{
		entries: make(map[string]*loginAttemptEntry),
	}
}

// recordAttempt 在验证密码之前登记一次尝试，当前仍在锁定期则返回 false。
// 第五次登记设置锁定但仍返回 true，当前尝试继续校验，后续尝试才被锁定拒绝。
// AuthService 在成功校验密码后调用 reset，因此本计数器本身不判断密码是否错误。
func (t *loginAttemptTracker) recordAttempt(userID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	e, ok := t.entries[userID]
	if !ok {
		e = &loginAttemptEntry{}
		t.entries[userID] = e
	}

	if now.Before(e.lockedUntil) {
		return false
	}

	// 存在已到期锁定时清空旧计数和截止时间，再登记当前尝试。
	if !e.lockedUntil.IsZero() {
		e.failures = 0
		e.lockedUntil = time.Time{}
	}

	e.failures++
	if e.failures >= loginMaxFailures {
		e.lockedUntil = now.Add(loginLockDuration)
		e.failures = 0
	}

	return true
}

// reset 删除指定用户的计数和锁定状态，其他用户的记录保持原样。
func (t *loginAttemptTracker) reset(userID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, userID)
}
