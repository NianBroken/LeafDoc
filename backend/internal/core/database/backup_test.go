// 本文件从公开接口验证非法备份载荷，不连接数据库或执行实际导入。
package database_test

import (
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// 缺少格式、未知版本及含 SQL 结构字符的表名都必须被 ValidateDump 拒绝。
// 用例只断言存在错误，不依赖具体错误文本或先触发哪项校验。
func TestMalformedDumpRejected(t *testing.T) {
	for _, raw := range []string{`{}`, `{"format":2}`, `{"format":1,"tables":[{"name":"users;DROP","columns":["id"]}]}`} {
		if _, err := database.ValidateDump([]byte(raw)); err == nil {
			t.Fatal("malformed dump accepted")
		}
	}
}
