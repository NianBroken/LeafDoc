// 本文件验证同端口协议识别、真实 TLS 状态和并发请求。
package dualhttp

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/serviceports"
)

// TestBothProtocols 验证明文和 TLS 请求不会互相污染连接状态。
func TestBothProtocols(t *testing.T) {
	fixture := httptest.NewTLSServer(http.NotFoundHandler())
	certificate := fixture.TLS.Certificates[0]
	fixture.Close()
	directory := t.TempDir()
	certFile, keyFile := filepath.Join(directory, "cert.pem"), filepath.Join(directory, "key.pem")
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Certificate[0]})
	keyDER, err := x509.MarshalPKCS8PrivateKey(certificate.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(certFile, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}
	base, err := net.Listen("tcp", "127.0.0.1:"+serviceports.Port("httpTest"))
	if err != nil {
		t.Fatal(err)
	}
	listener, err := Wrap(base, certFile, keyFile)
	if err != nil {
		base.Close()
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, r.TLS != nil) }), ReadHeaderTimeout: time.Second}
	defer server.Close()
	go server.Serve(listener)
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(certPEM)
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	var workers sync.WaitGroup
	for i := 0; i < 12; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			scheme, want := "http", "false"
			if index%2 == 1 {
				scheme, want = "https", "true"
			}
			response, err := client.Get(scheme + "://" + listener.Addr().String())
			if err != nil {
				t.Error(err)
				return
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil || string(body) != want {
				t.Errorf("protocol=%s body=%s error=%v", scheme, body, err)
			}
		}(i)
	}
	workers.Wait()
}
