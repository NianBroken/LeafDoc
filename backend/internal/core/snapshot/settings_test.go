// 本文件用有效的每日配置作为基准，逐项构造必须被拒绝的配置。
package snapshot

import (
	"testing"
)

// 每个样本只改变保留数、时区或小时中的一项，要求校验返回错误。
// 子测试不依赖 map 的遍历顺序，也不断言错误文本或代码。
func TestValidateSettingsRejectsInvalidValues(t *testing.T) {
	base := Settings{Enabled: true, Mode: scheduleModeDaily, Hour: 2, TimeZone: "UTC", RetentionCount: 0}
	for name, value := range map[string]Settings{
		"negative retention":  func() Settings { v := base; v.RetentionCount = -1; return v }(),
		"excessive retention": func() Settings { v := base; v.RetentionCount = maxRetentionCount + 1; return v }(),
		"invalid timezone":    func() Settings { v := base; v.TimeZone = "Mars/Olympus"; return v }(),
		"local timezone":      func() Settings { v := base; v.TimeZone = "Local"; return v }(),
		"invalid daily hour":  func() Settings { v := base; v.Hour = 24; return v }(),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateSettings(value); err == nil {
				t.Fatal("ValidateSettings() accepted invalid settings")
			}
		})
	}
}
