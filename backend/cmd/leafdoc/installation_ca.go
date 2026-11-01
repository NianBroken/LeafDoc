// 公开证书下载只接受单个有效 CA 证书，拒绝私钥、证书拼接和配置错误。
package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"net/http"
	"os"
)

// serveInstallationCA 仅由首次安装服务调用，不向正式业务路由注册下载入口。
func serveInstallationCA(w http.ResponseWriter, r *http.Request, file string) {
	if file == "" {
		http.NotFound(w, r)
		return
	}
	content, err := os.ReadFile(file)
	if err == nil {
		block, rest := pem.Decode(content)
		if block != nil && block.Type == "CERTIFICATE" && len(bytes.TrimSpace(rest)) == 0 && bytes.Equal(bytes.TrimSpace(content), bytes.TrimSpace(pem.EncodeToMemory(block))) {
			certificate, parseErr := x509.ParseCertificate(block.Bytes)
			if parseErr == nil && certificate.IsCA && certificate.CheckSignatureFrom(certificate) == nil {
				w.Header().Set("Content-Type", "application/x-pem-file")
				w.Header().Set("Content-Disposition", `attachment; filename="leafdoc-local-ca.crt"`)
				if _, writeErr := w.Write(content); writeErr != nil {
					slog.Error("公开 CA 响应失败", "error", writeErr)
				}
				return
			}
		}
	}
	http.Error(w, "公开 CA 暂不可用，请检查初始化服务", http.StatusServiceUnavailable)
}
