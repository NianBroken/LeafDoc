// 本文件把账号通知所需的本地化内容转换为邮件，并为发送建立超时上下文。
package email

import (
	"context"
	"html"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/localization"
)

// Service 为账号令牌流程提供邀请与密码重置邮件发送入口。
// 配置与客户端在构造时保存，每次发送按当前全局语言获取邮件文字。
type Service struct {
	cfg    Config
	client *Client
}

// NewService 创建邮件客户端但不建连，是否启用该服务由外层装配流程决定。
func NewService(cfg Config) *Service {
	return &Service{cfg: cfg, client: NewClient(cfg)}
}

// SendPasswordResetEmail 使用重置密码文案生成两种正文，再发送到指定邮箱。
// 正文与主题分别读取语言提供器，本方法不验证操作链接或记录发送日志。
func (s *Service) SendPasswordResetEmail(ctx context.Context, to, link string) error {
	html, text := localizedMail(link, false)
	return s.send(ctx, to, localization.Mail(false).Subject, html, text)
}

// SendInviteEmail 使用邀请文案和传入链接发送邮件，发送失败向调用方返回。
func (s *Service) SendInviteEmail(ctx context.Context, to, link string) error {
	html, text := localizedMail(link, true)
	return s.send(ctx, to, localization.Mail(true).Subject, html, text)
}

// send 从调用方上下文派生发送期限，原上下文期限更早时仍遵守更早期限。
// 客户端返回的错误直接向上传递，调用方负责决定日志和接口响应，
// 延迟取消释放上下文计时资源，不在此处重试发送。
func (s *Service) send(ctx context.Context, to, subject, html, text string) error {
	sendCtx, cancel := context.WithTimeout(ctx, s.cfg.TimeoutOrDefault())
	defer cancel()

	return s.client.Send(sendCtx, Message{To: to, Subject: subject, HTML: html, Text: text})
}

// localizedMail 按全局语言选择邀请或重置文案，纯文本各段使用空行分隔。
// HTML 中的文案、链接属性和值都经过转义，链接协议和目标地址不在此处校验。
func localizedMail(link string, invite bool) (string, string) {
	value := localization.Mail(invite)
	text := strings.Join([]string{value.Intro, value.Action, link, value.Expiry}, "\n\n")
	body := "<p>" + html.EscapeString(value.Intro) + "</p><p>" + html.EscapeString(value.Action) + "</p><p><a href=\"" + html.EscapeString(link) + "\">" + html.EscapeString(link) + "</a></p><p>" + html.EscapeString(value.Expiry) + "</p>"
	return body, text
}
