// 本文件只验证固定来源的配置值，不连接数据库或构造持久化 Store。
package settings

import (
	"errors"
	"testing"
)

// testConfig 提供普通字段和不参与 JSON 的字段，作为泛型配置测试载体。
type testConfig struct {
	Name     string `json:"name"`
	Injected bool   `json:"-"`
}

// TestFixed_Get_ReturnsPinnedValue 确认构造时传入的名称能够原样读取。
func TestFixed_Get_ReturnsPinnedValue(t *testing.T) {
	f := NewFixed(testConfig{Name: "pinned"})
	if got := f.Get(); got.Name != "pinned" {
		t.Fatalf("Get() = %+v, want Name %q", got, "pinned")
	}
}

// TestFixed_Put_AlwaysFailsWithErrEnvManaged 同时验证拒绝写入的错误类别和原值未改变。
func TestFixed_Put_AlwaysFailsWithErrEnvManaged(t *testing.T) {
	f := NewFixed(testConfig{Name: "pinned"})
	if err := f.Put(testConfig{Name: "changed"}); !errors.Is(err, ErrEnvManaged) {
		t.Fatalf("Put() err = %v, want ErrEnvManaged", err)
	}
	if got := f.Get(); got.Name != "pinned" {
		t.Fatalf("Get() after rejected Put = %+v, want unchanged Name %q", got, "pinned")
	}
}

// TestFixed_Reload_IsANoOp 检查固定值重载成功返回，且不会清空或替换当前名称。
func TestFixed_Reload_IsANoOp(t *testing.T) {
	f := NewFixed(testConfig{Name: "pinned"})
	if err := f.Reload(); err != nil {
		t.Fatalf("Reload() err = %v, want nil", err)
	}
	if got := f.Get(); got.Name != "pinned" {
		t.Fatalf("Get() after Reload = %+v, want unchanged Name %q", got, "pinned")
	}
}

// TestFixed_EnvManaged_IsTrue 确认零内容固定配置也按启动来源只读值呈现。
func TestFixed_EnvManaged_IsTrue(t *testing.T) {
	if !NewFixed(testConfig{}).EnvManaged() {
		t.Fatal("Fixed.EnvManaged() = false, want true")
	}
}
