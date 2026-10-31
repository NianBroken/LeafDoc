// 本包提供测试用文件和 multipart 上传流构造工具，不参与业务请求处理。
// 写入路径和输入内容由各测试决定，返回的文件句柄需要调用方关闭。
package testutil

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"
)

// WriteFile 将正斜杠形式的相对路径转换为宿主路径，创建父目录后写入字符串字节。
// 不校验内容编码或路径是否留在 base 内，调用测试负责提供合法样本路径。
// 失败通过测试框架立即结束当前用例，成功返回拼接路径，不额外转换成绝对路径。
func WriteFile(t *testing.T, base, rel, content string) string {
	t.Helper()
	abs := filepath.Join(base, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return abs
}

// CreateMultipartFile 先编码包含 file 字段的 multipart 请求体，再解析同一请求体，
// 返回第一项的读取句柄和解析后的文件名。每步错误直接返回，不以空流代替失败。
// ReadForm 使用十 MiB 内存额度，超出额度时遵循标准库的临时文件行为；
// 此工具只返回文件句柄，不返回表单清理对象，当前小样本调用方负责关闭句柄。
func CreateMultipartFile(filename string, content []byte) (multipart.File, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(content); err != nil {
		return nil, "", err
	}
	// 关闭编码器写出结束边界，随后读取器才能消费完整 multipart 格式。
	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20)
	if err != nil {
		return nil, "", err
	}

	// 字段名与上面的编码字段一致，不把普通字段当作上传文件。
	files := form.File["file"]
	if len(files) == 0 {
		return nil, "", fmt.Errorf("no file found in form")
	}

	f, err := files[0].Open()
	return f, files[0].Filename, err
}
