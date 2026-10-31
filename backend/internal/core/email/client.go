// 本文件负责 SMTP 建连、可选 TLS 与认证，以及 MIME 邮件的串行发送。
package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// Message 描述单封邮件的收件人、主题及两种正文形式。
// HTML 非空时同时发送纯文本与 HTML 部分，否则只发送纯文本。
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Client 使用标准库 SMTP 客户端发送邮件。隐式 TLS 在进入 SMTP 会话前完成握手，
// STARTTLS 先建立普通连接再升级，其他安全模式使用普通连接。
// 用户名非空时尝试 PLAIN 认证，不由 SecurityNone 单独禁止认证。
type Client struct {
	cfg Config
}

// NewClient 保存配置，不立即建立网络连接，也不在此处校验配置。
func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg}
}

// Send 依次建立连接、创建 SMTP 会话、升级加密、认证、提交信封和正文，再发送 QUIT。
// 建连和隐式 TLS 握手使用上下文；连接建立后将上下文截止时间设为连接读写期限。
// 本方法不创建默认超时，服务层负责提供带期限的上下文。仅取消但没有截止时间的
// 上下文不会在建连完成后单独触发 SMTP 会话中断。
func (c *Client) Send(ctx context.Context, msg Message) error {
	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)

	conn, err := c.dial(ctx, addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	// 后续 SMTP 方法不接收上下文，通过连接期限约束读写及 STARTTLS 握手。
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	defer func() { _ = conn.Close() }()

	client, err := smtp.NewClient(conn, c.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp handshake: %w", err)
	}
	defer func() { _ = client.Close() }()

	if c.cfg.Security == SecurityStartTLS {
		// 证书跳过校验只由显式配置启用，零值保持证书校验。
		tlsConfig := &tls.Config{ServerName: c.cfg.Host, InsecureSkipVerify: c.cfg.InsecureSkipVerify} //nolint:gosec // 跳过证书校验由显式配置控制，默认关闭。
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}

	// 认证触发条件仅为用户名非空，传输是否允许 PLAIN 由标准库认证实现检查。
	if c.cfg.Username != "" {
		auth := smtp.PlainAuth("", c.cfg.Username, c.cfg.Password, c.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(c.cfg.From); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	// DATA 写入器关闭时结束正文并读取服务器确认，成功后才结束整个 SMTP 会话。
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(c.buildRawMessage(msg))); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close body: %w", err)
	}

	return client.Quit()
}

// dial 让普通 TCP 与隐式 TLS 的连接阶段都响应上下文取消和截止时间。
// TLS Dialer 在返回连接前完成握手，握手失败时由标准库关闭底层连接。
func (c *Client) dial(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{}
	if c.cfg.Security == SecurityTLS {
		tlsConfig := &tls.Config{ServerName: c.cfg.Host, InsecureSkipVerify: c.cfg.InsecureSkipVerify} //nolint:gosec // 跳过证书校验由显式配置控制，默认关闭。
		tlsDialer := tls.Dialer{NetDialer: &d, Config: tlsConfig}
		return tlsDialer.DialContext(ctx, "tcp", addr)
	}
	return d.DialContext(ctx, "tcp", addr)
}

// mimeBoundary 分隔纯文本与 HTML 两部分，正文构造使用同一个边界值闭合消息。
const mimeBoundary = "leafdoc-boundary-9f3a2c1e"

// buildRawMessage 将配置中的发件人与消息字段拼为 CRLF 分行的邮件内容。
// 所有动态头字段都删除回车与换行，防止字段值插入额外头行。正文按原内容写入，
// HTML 非空时生成 multipart/alternative，并先写纯文本部分，再写 HTML 部分。
func (c *Client) buildRawMessage(msg Message) string {
	from := sanitizeHeader(c.cfg.From)
	if c.cfg.FromName != "" {
		from = fmt.Sprintf("%s <%s>", sanitizeHeader(c.cfg.FromName), from)
	}

	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + sanitizeHeader(msg.To) + "\r\n")
	b.WriteString("Subject: " + sanitizeHeader(msg.Subject) + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")

	if msg.HTML != "" {
		b.WriteString("Content-Type: multipart/alternative; boundary=" + mimeBoundary + "\r\n\r\n")
		b.WriteString("--" + mimeBoundary + "\r\n")
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.Text + "\r\n\r\n")
		b.WriteString("--" + mimeBoundary + "\r\n")
		b.WriteString("Content-Type: text/html; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.HTML + "\r\n\r\n")
		b.WriteString("--" + mimeBoundary + "--\r\n")
	} else {
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		b.WriteString(msg.Text + "\r\n")
	}

	return b.String()
}

// sanitizeHeader 删除回车和换行，不负责邮箱格式校验、显示名编码或 HTML 转义。
func sanitizeHeader(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}
