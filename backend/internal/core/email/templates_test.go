// 本文件在多种界面语言下直接核验邮件正文，不连接 SMTP 服务器。
package email

import (
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/localization"
)

// 邀请和重置邮件均保留操作链接，并转义 HTML 特殊字符。
// 逐一设置语言提供器，检查对应操作文案存在，测试结束将提供器设置为中文。
func TestLocalizedMail(t *testing.T) {
	t.Cleanup(func() { localization.SetLanguageProvider(func() string { return "zh" }) })
	for _, language := range []string{"zh", "en", "de", "es", "fr"} {
		localization.SetLanguageProvider(func() string { return language })
		for _, invite := range []bool{false, true} {
			link := `https://wiki.example.com/action?token="><script>alert(1)</script>`
			html, text := localizedMail(link, invite)
			if !strings.Contains(text, link) || !strings.Contains(html, "https://wiki.example.com/action") {
				t.Fatal("missing action link")
			}
			if strings.Contains(html, "<script>") {
				t.Fatal("unescaped HTML")
			}
			mail := localization.Mail(invite)
			if !strings.Contains(text, mail.Action) || mail.Subject == "" {
				t.Fatal("missing localized mail")
			}
			if language == "zh" && !strings.Contains(text, "链接") {
				t.Fatal("Chinese mail not localized")
			}
		}
	}
}
