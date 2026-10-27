package email

import (
	"strings"
	"testing"

	"github.com/perber/wiki/internal/core/localization"
)

// Notification bodies retain links, escape HTML, and use the global language.
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
