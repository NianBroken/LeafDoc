// 本文件检查固定端口表的取值范围、互不重复约束和端口参数校验结果。
package serviceports

import (
	"sort"
	"strconv"
	"testing"
)

// TestAllocation 检查前后端的既定端口，并验证表内所有端口均为非特权有效端口且互不重复。
func TestAllocation(t *testing.T) {
	if Port("frontend") != "24691" || Port("backend") != "24692" {
		t.Fatal("application listeners changed")
	}
	var values []int
	for _, name := range Names() {
		value, err := strconv.Atoi(Port(name))
		if err != nil {
			t.Fatal(err)
		}
		values = append(values, value)
	}
	// 排序后重复值必定相邻，以同一轮循环同时核验范围和唯一性。
	sort.Ints(values)
	for i, value := range values {
		if value < 1024 || value > 65535 || (i > 0 && value == values[i-1]) {
			t.Fatalf("invalid or duplicate allocation: %v", values)
		}
	}
}

// TestValidator 检查零端口、其他服务端口、空文本及非法文本均被后端校验器拒绝。
// 后端自身的固定端口必须通过，避免校验器将所有输入都判为错误。
func TestValidator(t *testing.T) {
	for _, value := range []string{"0", "24693", "", "random", "-1"} {
		if Validator("backend")(value) == nil {
			t.Fatalf("accepted override %q", value)
		}
	}
	if err := Validator("backend")(Port("backend")); err != nil {
		t.Fatal(err)
	}
}
