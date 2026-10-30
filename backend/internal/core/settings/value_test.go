package settings

import (
	"errors"
	"testing"
)

type testConfig struct {
	Name     string `json:"name"`
	Injected bool   `json:"-"`
}

func TestFixed_Get_ReturnsPinnedValue(t *testing.T) {
	f := NewFixed(testConfig{Name: "pinned"})
	if got := f.Get(); got.Name != "pinned" {
		t.Fatalf("Get() = %+v, want Name %q", got, "pinned")
	}
}

func TestFixed_Put_AlwaysFailsWithErrEnvManaged(t *testing.T) {
	f := NewFixed(testConfig{Name: "pinned"})
	if err := f.Put(testConfig{Name: "changed"}); !errors.Is(err, ErrEnvManaged) {
		t.Fatalf("Put() err = %v, want ErrEnvManaged", err)
	}
	if got := f.Get(); got.Name != "pinned" {
		t.Fatalf("Get() after rejected Put = %+v, want unchanged Name %q", got, "pinned")
	}
}

func TestFixed_Reload_IsANoOp(t *testing.T) {
	f := NewFixed(testConfig{Name: "pinned"})
	if err := f.Reload(); err != nil {
		t.Fatalf("Reload() err = %v, want nil", err)
	}
	if got := f.Get(); got.Name != "pinned" {
		t.Fatalf("Get() after Reload = %+v, want unchanged Name %q", got, "pinned")
	}
}

func TestFixed_EnvManaged_IsTrue(t *testing.T) {
	if !NewFixed(testConfig{}).EnvManaged() {
		t.Fatal("Fixed.EnvManaged() = false, want true")
	}
}
