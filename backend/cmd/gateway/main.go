// 网关与业务服务共用协议识别模块和固定端口分配。
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/dualhttp"
	"github.com/NianBroken/LeafDoc/internal/core/serviceports"
)

// main 执行网关并报告启动错误。
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run 通过真实后端 TLS 连接保留入站协议，并验证后端证书。
func run() error {
	host := os.Getenv("LEAFDOC_GATEWAY_HOST")
	if host == "" {
		host = os.Getenv("LEAFDOC_HOST")
	}
	if host == "" {
		host = "0.0.0.0"
	}
	target := os.Getenv("LEAFDOC_BACKEND_URL")
	if target == "" {
		target = "http://127.0.0.1:" + serviceports.Port("backend")
	}
	upstream, err := url.Parse(target)
	if err != nil {
		return err
	}
	if upstream.Hostname() == "" || upstream.Port() != serviceports.Port("backend") {
		return fmt.Errorf("backend must use fixed port %s", serviceports.Port("backend"))
	}
	certFile, keyFile := os.Getenv("LEAFDOC_TLS_CERT"), os.Getenv("LEAFDOC_TLS_KEY")
	roots, err := x509.SystemCertPool()
	if err != nil {
		return err
	}
	pem, err := os.ReadFile(certFile)
	if err != nil {
		return err
	}
	if !roots.AppendCertsFromPEM(pem) {
		return fmt.Errorf("invalid gateway certificate")
	}
	serverName := os.Getenv("LEAFDOC_BACKEND_TLS_NAME")
	if serverName == "" {
		serverName = "localhost"
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{RootCAs: roots, ServerName: serverName, MinVersion: tls.VersionTLS12}
	defer transport.CloseIdleConnections()
	proxy := &httputil.ReverseProxy{Transport: transport, Rewrite: func(r *httputil.ProxyRequest) {
		r.SetURL(upstream)
		r.Out.Host = r.In.Host
		// 协议通过连接传递，不采用客户端传入的协议头。
		r.Out.URL.Scheme = "http"
		if r.In.TLS != nil {
			r.Out.URL.Scheme = "https"
		}
		r.Out.Header.Del("X-Forwarded-Proto")
		r.Out.Header.Del("X-Forwarded-Ssl")
		r.Out.Header.Del("Front-End-Https")
	}}
	base, err := net.Listen("tcp", net.JoinHostPort(host, serviceports.Port("frontend")))
	if err != nil {
		return err
	}
	listener, err := dualhttp.Wrap(base, certFile, keyFile)
	if err != nil {
		base.Close()
		return err
	}
	defer listener.Close()
	server := &http.Server{Handler: proxy, ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("HTTP and HTTPS gateway listening on %s", listener.Addr())
	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
