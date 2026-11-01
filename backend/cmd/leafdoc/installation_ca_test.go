// 下载测试覆盖公开 CA、缺失文件和混入私钥的配置，确保响应不泄露私有内容。
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstallationCA(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	private := "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----\n"
	for _, tc := range []struct {
		name, content string
		status        int
	}{
		{"certificate", certificate, 200},
		{"private", private, 503},
		{"concatenated", certificate + private, 503},
		{"prefix", private + certificate, 503},
		{"invalid", "invalid", 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "public.crt")
			if err := os.WriteFile(file, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			serveInstallationCA(response, httptest.NewRequest("GET", "/api/setup/ca", nil), file)
			if response.Code != tc.status {
				t.Fatalf("status = %d", response.Code)
			}
			if strings.Contains(response.Body.String(), "PRIVATE KEY") {
				t.Fatal("private content disclosed")
			}
			if tc.status == 200 && response.Body.String() != certificate {
				t.Fatal("certificate differs")
			}
		})
	}
	for _, tc := range []struct {
		file   string
		status int
	}{{"", 404}, {filepath.Join(t.TempDir(), "missing"), 503}} {
		response := httptest.NewRecorder()
		serveInstallationCA(response, httptest.NewRequest("GET", "/api/setup/ca", nil), tc.file)
		if response.Code != tc.status {
			t.Fatalf("missing status = %d", response.Code)
		}
	}
}
