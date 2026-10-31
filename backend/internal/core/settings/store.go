// Package settings 将带类型的实例配置保存到 MySQL，并提供进程内缓存和变化通知。
// 键名中保留的 .json 表示配置记录名称，不表示在数据目录中创建同名文件。
package settings

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// Reloadable 统一暴露重新加载操作，供恢复流程刷新依赖数据库状态的服务。
type Reloadable interface{ Reload() error }

// Store 缓存 settings 命名空间中的一项配置，directory 用于定位数据库上下文。
// val 为当前缓存，zero 为缺失字段的初始值，onLoad 在加载和写入缓存后处理值。
// T 按值复制，其中的 map、切片和指针不会自动深拷贝。
type Store[T any] struct {
	directory, key string
	onLoad         func(*T)
	mu             sync.RWMutex
	val, zero      T
}

// New 构造存储并同步读取配置，记录不存在时将默认值写入数据库。
// 文件权限参数为兼容调用形式保留，当前实现不使用它创建磁盘文件。
// 构造过程调用内部加载方法，不发送变化通知；失败时不返回可用 Store。
func New[T any](dir, key string, _ os.FileMode, zero T, onLoad func(*T)) (*Store[T], error) {
	s := &Store[T]{directory: dir, key: key, onLoad: onLoad, val: zero, zero: zero}
	if err := s.reloadLocked(); err != nil {
		return nil, err
	}
	return s, nil
}

// Get 在读锁内复制当前值，不主动重新查询数据库。
func (s *Store[T]) Get() T { s.mu.RLock(); defer s.mu.RUnlock(); return s.val }

// Put 先编码输入，再在写锁内写数据库、替换缓存、调用 onLoad 并通知订阅者。
// onLoad 对缓存的改变不会在本方法中再次写回数据库，回调不应重入同一 Store。
func (s *Store[T]) Put(v T) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err = database.WriteRecord(s.directory, "settings", s.key, raw); err != nil {
		return err
	}
	s.val = v
	if s.onLoad != nil {
		s.onLoad(&s.val)
	}
	notifyCommitted()
	return nil
}

// Reload 持写锁重新加载配置，成功后通知订阅者，即使最终值与先前相同也发通知。
// 加载过程不是事务式缓存替换，内部后续步骤失败时缓存已被部分赋值。
func (s *Store[T]) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.reloadLocked(); err != nil {
		return err
	}
	notifyCommitted()
	return nil
}

// reloadLocked 由构造过程或已持写锁的方法调用，读取、校验并物化配置字段。
// 顶层必须为非 null 对象，顶层属性不允许显式 null，嵌套属性不在此循环逐层检查。
func (s *Store[T]) reloadLocked() error {
	raw, err := database.ReadRecord(s.directory, "settings", s.key)
	if errors.Is(err, sql.ErrNoRows) {
		s.val = s.zero
		raw, err = json.Marshal(s.zero)
		if err != nil {
			return err
		}
		if err = database.WriteRecord(s.directory, "settings", s.key, raw); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("read setting %s: %w", s.key, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("invalid setting %s: %w", s.key, err)
	}
	if fields == nil {
		return fmt.Errorf("setting %s must be an object", s.key)
	}
	for field, value := range fields {
		if string(value) == "null" {
			return fmt.Errorf("setting %s field %s must not be null", s.key, field)
		}
	}
	// 从默认结构开始反序列化，缺失字段沿用默认值，显式 false 和零值由输入覆盖。
	s.val = s.zero
	if err = json.Unmarshal(raw, &s.val); err != nil {
		return err
	}
	if s.onLoad != nil {
		s.onLoad(&s.val)
	}
	// 回调处理后的值重新编码，与原 JSON 文本不同就写回，包括字段顺序或排版差异。
	// 这一步会固化缺失字段的默认值，也只保留 T 能够重新编码出的字段。
	completed, err := json.Marshal(s.val)
	if err != nil {
		return err
	}
	if string(completed) != string(raw) {
		if err := database.WriteRecord(s.directory, "settings", s.key, completed); err != nil {
			return err
		}
	}
	return nil
}
