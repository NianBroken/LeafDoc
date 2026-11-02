// dualhttp 包在同一个 TCP 监听上提供 HTTP 与 HTTPS。
package dualhttp

import (
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
	"github.com/soheilhy/cmux"
)

// Listener 汇集两种协议连接，并保留 TLS 连接的具体类型。
type Listener struct {
	net.Listener
	connections chan net.Conn
	done        chan struct{}
	once        sync.Once
	mu          sync.RWMutex
	terminal    error
	mux         cmux.CMux
}

// Wrap 先验证证书，再通过连接内容分流，不创建额外端口。
func Wrap(base net.Listener, certFile, keyFile string) (*Listener, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	mux := cmux.New(base)
	mux.SetReadTimeout(5 * time.Second)
	plain := mux.Match(cmux.HTTP1Fast())
	encrypted := tls.NewListener(mux.Match(cmux.Any()), &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12, NextProtos: []string{"http/1.1"}})
	l := &Listener{Listener: base, connections: make(chan net.Conn), done: make(chan struct{}), mux: mux}
	pump := func(source net.Listener) {
		for {
			conn, err := source.Accept()
			if err != nil {
				return
			}
			select {
			case l.connections <- conn:
			case <-l.done:
				shared.LogNetworkClose(conn, "close pending dual protocol connection")
				return
			}
		}
	}
	go pump(plain)
	go pump(encrypted)
	// 分流服务异常透传给 HTTP 服务；主动关闭保持标准关闭错误。
	go func() {
		err := mux.Serve()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			l.mu.Lock()
			l.terminal = err
			l.mu.Unlock()
		}
		shared.LogNetworkClose(l, "close dual protocol listener")
	}()
	return l, nil
}

// Accept 保留 *tls.Conn，使 net/http 正确识别加密请求。
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.connections:
		return conn, nil
	case <-l.done:
		l.mu.RLock()
		err := l.terminal
		l.mu.RUnlock()
		if err != nil {
			return nil, err
		}
		return nil, net.ErrClosed
	}
}

// Close 释放底层监听并解除等待中的连接转交。
func (l *Listener) Close() error {
	var err error
	l.once.Do(func() { close(l.done); l.mux.Close(); err = l.Listener.Close() })
	return err
}
