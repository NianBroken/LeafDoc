// 本文件使用本地 TCP 服务核验 SMTP 内容和 TLS 握手的期限，不访问外部邮件系统。
package email

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/serviceports"
)

// fakeSMTPServer 接受一次明文 SMTP 会话，只实现发送测试使用的指令。
// 信封和正文记录在字段中，供断言核对客户端实际传出的内容。
type fakeSMTPServer struct {
	addr string

	mailFrom string
	rcptTo   string
	data     string
}

// startFakeSMTPServer 监听测试端口并启动单连接处理，测试结束时关闭监听器。
func startFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:"+serviceports.Port("emailSMTPTest"))
	if err != nil {
		t.Fatalf("failed to start fake smtp listener: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	srv := &fakeSMTPServer{addr: ln.Addr().String()}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		srv.serve(conn)
	}()

	return srv
}

// serve 按行读取指令，DATA 期间收集正文，单独句点结束正文，QUIT 结束会话。
func (s *fakeSMTPServer) serve(conn net.Conn) {
	reader := bufio.NewReader(conn)
	writeLine(conn, "220 fake.smtp.test ESMTP")

	inData := false
	var dataLines []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				s.data = strings.Join(dataLines, "\r\n")
				writeLine(conn, "250 OK: message accepted")
				continue
			}
			dataLines = append(dataLines, line)
			continue
		}

		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"):
			writeLine(conn, "250-fake.smtp.test greets you")
			writeLine(conn, "250 8BITMIME")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			s.mailFrom = line
			writeLine(conn, "250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			s.rcptTo = line
			writeLine(conn, "250 OK")
		case upper == "DATA":
			inData = true
			writeLine(conn, "354 Start mail input; end with <CRLF>.<CRLF>")
		case upper == "QUIT":
			writeLine(conn, "221 Bye")
			return
		default:
			writeLine(conn, "500 unrecognized command")
		}
	}
}

// writeLine 以 SMTP 所用 CRLF 写入一行，模拟服务器不单独报告写入错误。
func writeLine(conn net.Conn, line string) {
	_, _ = fmt.Fprintf(conn, "%s\r\n", line)
}

// 明文往返核验发件人、收件人、主题及带显示名的 From 头确实到达服务器。
func TestClient_Send_PlaintextRoundTrip_DeliversMessage(t *testing.T) {
	srv := startFakeSMTPServer(t)
	host, port := splitHostPort(t, srv.addr)

	client := NewClient(Config{
		Host:     host,
		Port:     port,
		From:     "wiki@example.com",
		FromName: "LeafDoc",
		Security: SecurityNone,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Send(ctx, Message{
		To:      "user@example.com",
		Subject: "Reset your LeafDoc password",
		HTML:    "<p>link</p>",
		Text:    "link",
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	// 发送返回后保留短暂等待，再读取模拟服务器记录的会话字段。
	time.Sleep(50 * time.Millisecond)

	if !strings.Contains(srv.mailFrom, "wiki@example.com") {
		t.Errorf("expected MAIL FROM to contain sender, got %q", srv.mailFrom)
	}
	if !strings.Contains(srv.rcptTo, "user@example.com") {
		t.Errorf("expected RCPT TO to contain recipient, got %q", srv.rcptTo)
	}
	if !strings.Contains(srv.data, "Subject: Reset your LeafDoc password") {
		t.Errorf("expected message data to contain subject header, got: %s", srv.data)
	}
	if !strings.Contains(srv.data, "From: LeafDoc <wiki@example.com>") {
		t.Errorf("expected message data to contain From header with display name, got: %s", srv.data)
	}
}

// 包含 CRLF 的收件人与主题不得生成独立的 Bcc 或自定义头行。
func TestClient_BuildRawMessage_StripsCRLFFromHeaderValues(t *testing.T) {
	client := NewClient(Config{From: "wiki@example.com", FromName: "LeafDoc"})

	raw := client.buildRawMessage(Message{
		To:      "attacker@example.com\r\nBcc: victim@example.com",
		Subject: "hello\r\nX-Injected: true",
		Text:    "body",
	})

	if strings.Contains(raw, "\r\nBcc: victim@example.com") {
		t.Errorf("expected injected Bcc to not become its own header line, got: %s", raw)
	}
	if strings.Contains(raw, "\r\nX-Injected: true") {
		t.Errorf("expected injected header to not become its own header line, got: %s", raw)
	}
}

// splitHostPort 将监听器实际地址拆为客户端配置使用的主机和整数端口。
func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("failed to split addr %q: %v", addr, err)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		t.Fatalf("failed to parse port %q: %v", portStr, err)
	}
	return host, port
}

// TLS 对端只接收握手字节而不回复，用于验证期限和主动取消均能结束握手。
// 客户端失败后还须关闭 TCP 连接，使对端读取退出，避免留下悬挂连接。
func TestClient_Send_TLSHandshakeHonorsContext(t *testing.T) {
	const handshakeTimeout = 250 * time.Millisecond
	const completionTimeout = 3 * time.Second
	for _, cancelDuringHandshake := range []bool{false, true} {
		name := "deadline"
		if cancelDuringHandshake {
			name = "cancellation"
		}
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = listener.Close() })
			started := make(chan struct{})
			closed := make(chan error, 1)
			go func() {
				conn, acceptErr := listener.Accept()
				if acceptErr != nil {
					closed <- acceptErr
					return
				}
				defer func() { _ = conn.Close() }()
				if err := conn.SetDeadline(time.Now().Add(completionTimeout)); err != nil {
					closed <- err
					return
				}
				var firstByte [1]byte
				if _, err := conn.Read(firstByte[:]); err != nil {
					closed <- err
					return
				}
				close(started)
				_, err := io.Copy(io.Discard, conn)
				closed <- err
			}()
			host, port := splitHostPort(t, listener.Addr().String())
			client := NewClient(Config{Host: host, Port: port, Security: SecurityTLS})
			ctx, cancel := context.WithTimeout(context.Background(), completionTimeout)
			if !cancelDuringHandshake {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), handshakeTimeout)
			}
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- client.Send(ctx, Message{}) }()
			select {
			case <-started:
				if cancelDuringHandshake {
					cancel()
				}
			case <-time.After(completionTimeout):
				t.Fatal("TLS handshake did not start")
			}
			want := context.DeadlineExceeded
			if cancelDuringHandshake {
				want = context.Canceled
			}
			select {
			case err := <-result:
				if !errors.Is(err, want) {
					t.Fatalf("expected %v, got %v", want, err)
				}
			case <-time.After(completionTimeout):
				t.Fatal("TLS handshake did not stop with context")
			}
			select {
			case err := <-closed:
				if err != nil {
					t.Fatalf("client did not release connection cleanly: %v", err)
				}
			case <-time.After(completionTimeout):
				t.Fatal("TLS connection remained open")
			}
		})
	}
}
