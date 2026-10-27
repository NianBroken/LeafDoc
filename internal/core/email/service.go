package email

import (
	"context"
	"html"
	"strings"

	"github.com/perber/wiki/internal/core/localization"
)

// Service is the mail-sending façade used by internal/core/auth's
// EmailTokenService. It is only constructed when Config.Enabled() and auth
// are both true (see wiki.initEmail) — nil-checked everywhere it's consumed,
// same as this codebase's other optional-feature services.
type Service struct {
	cfg    Config
	client *Client
}

func NewService(cfg Config) *Service {
	return &Service{cfg: cfg, client: NewClient(cfg)}
}

func (s *Service) SendPasswordResetEmail(ctx context.Context, to, link string) error {
	html, text := localizedMail(link, false)
	return s.send(ctx, to, localization.Mail(false).Subject, html, text)
}

func (s *Service) SendInviteEmail(ctx context.Context, to, link string) error {
	html, text := localizedMail(link, true)
	return s.send(ctx, to, localization.Mail(true).Subject, html, text)
}

// send is not the swallow point for delivery failures — callers (the
// forgot-password fire-and-forget goroutine, the synchronous invite-send
// handler) decide whether/how to log or surface the error, per ADR-0008's
// "log once, at the swallow point" — so this deliberately does not log.
func (s *Service) send(ctx context.Context, to, subject, html, text string) error {
	sendCtx, cancel := context.WithTimeout(ctx, s.cfg.TimeoutOrDefault())
	defer cancel()

	return s.client.Send(sendCtx, Message{To: to, Subject: subject, HTML: html, Text: text})
}

// localizedMail escapes every HTML field and keeps the plain-text alternative readable.
func localizedMail(link string, invite bool) (string, string) {
	value := localization.Mail(invite)
	text := strings.Join([]string{value.Intro, value.Action, link, value.Expiry}, "\n\n")
	body := "<p>" + html.EscapeString(value.Intro) + "</p><p>" + html.EscapeString(value.Action) + "</p><p><a href=\"" + html.EscapeString(link) + "\">" + html.EscapeString(link) + "</a></p><p>" + html.EscapeString(value.Expiry) + "</p>"
	return body, text
}
