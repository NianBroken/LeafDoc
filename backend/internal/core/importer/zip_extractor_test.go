package importer

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// TestZipExtractor_ValidateExtractedFiles 解压仓库中的固定归档并检查预期文件路径。
// 测试结束清理整个工作区；这里检查文件存在性，不比较解压后的正文内容。
func TestZipExtractor_ValidateExtractedFiles(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	zipPath := "fixtures/fixture-1.zip"

	extractor := NewZipExtractor()
	ws, err := extractor.ExtractToTemp(filepath.Join(currentDir, zipPath))
	if err != nil {
		t.Fatalf("ExtractToTemp failed: %v", err)
	}
	defer func() {
		if err := ws.Cleanup(); err != nil {
			t.Fatalf("Cleanup failed: %v", err)
		}
	}()

	// 同时覆盖根目录文件和分区目录中的文件，确认相对层级在解压后保留。
	expectedFiles := []string{
		"home.md",
		"features/index.md",
		"features/mermaind.md",
	}

	for _, relPath := range expectedFiles {
		fullPath := filepath.Join(ws.Root, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s does not exist", relPath)
		}
	}
}

// TestZipExtractor_Cleanup 验证工作区清理会使整个解压根目录不再存在。
func TestZipExtractor_Cleanup(t *testing.T) {
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	zipPath := "fixtures/fixture-1.zip"

	extractor := NewZipExtractor()
	ws, err := extractor.ExtractToTemp(filepath.Join(currentDir, zipPath))
	if err != nil {
		t.Fatalf("ExtractToTemp failed: %v", err)
	}

	workspaceRoot := ws.Root

	// 显式执行清理，再用文件系统查询确认根目录已被删除。
	if err := ws.Cleanup(); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// 必须得到不存在结果，其他查询错误也不能满足此断言。
	if _, err := os.Stat(workspaceRoot); !os.IsNotExist(err) {
		t.Errorf("Workspace root %s still exists after cleanup", workspaceRoot)
	}
}

// writeRawZip 根据条目名与内容映射建立临时 ZIP，供测试精确控制解压大小。
// 映射遍历不保证条目顺序，所有条目写入后依次关闭归档写入器与磁盘文件。
func writeRawZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "fixture.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip: %v", err)
	}
	w := zip.NewWriter(f)
	for name, content := range entries {
		zw, err := w.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		if _, err := zw.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write zip entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close zip file: %v", err)
	}
	return zipPath
}

// TestZipExtractor_ExtractToDirWithLimits_RejectsEntryLargerThanPerFileLimit
// 使用 200 字节内容和 100 字节单文件上限，隔离验证单文件大小错误分类。
func TestZipExtractor_ExtractToDirWithLimits_RejectsEntryLargerThanPerFileLimit(t *testing.T) {
	zipPath := writeRawZip(t, map[string]string{
		"big.txt": strings.Repeat("A", 200),
	})
	limits := shared.ExtractionLimits{
		MaxEntryBytes:   100,
		MaxTotalBytes:   1 << 30,
		MaxRatio:        1 << 30,
		RatioFloorBytes: 1 << 30,
	}

	extractor := NewZipExtractor()
	ws, err := extractor.extractToDirWithLimits(zipPath, t.TempDir(), limits)
	if ws != nil {
		defer func() { _ = ws.Cleanup() }()
	}
	if !errors.Is(err, shared.ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got: %v", err)
	}
}

// TestZipExtractor_ExtractToDirWithLimits_RejectsManyFilesSummingPastCumulativeLimit
// 每个文件都低于单文件限制，但多个 60 字节文件共用 100 字节预算时必须失败。
func TestZipExtractor_ExtractToDirWithLimits_RejectsManyFilesSummingPastCumulativeLimit(t *testing.T) {
	zipPath := writeRawZip(t, map[string]string{
		"a.txt": strings.Repeat("A", 60),
		"b.txt": strings.Repeat("B", 60),
		"c.txt": strings.Repeat("C", 60),
	})
	limits := shared.ExtractionLimits{
		MaxEntryBytes:   1 << 30,
		MaxTotalBytes:   100,
		MaxRatio:        1 << 30,
		RatioFloorBytes: 1 << 30,
	}

	extractor := NewZipExtractor()
	ws, err := extractor.extractToDirWithLimits(zipPath, t.TempDir(), limits)
	if ws != nil {
		defer func() { _ = ws.Cleanup() }()
	}
	if !errors.Is(err, shared.ErrCumulativeSizeTooLarge) {
		t.Fatalf("expected ErrCumulativeSizeTooLarge, got: %v", err)
	}
}

// TestZipExtractor_ExtractToDirWithLimits_RejectsHighDecompressionRatio
// 使用重复字节产生高压缩率内容，大小限制放宽，只检验解压比例错误。
func TestZipExtractor_ExtractToDirWithLimits_RejectsHighDecompressionRatio(t *testing.T) {
	zipPath := writeRawZip(t, map[string]string{
		"bomb.txt": strings.Repeat("A", 5000),
	})
	limits := shared.ExtractionLimits{
		MaxEntryBytes:   1 << 30,
		MaxTotalBytes:   1 << 30,
		MaxRatio:        3,
		RatioFloorBytes: 50,
	}

	extractor := NewZipExtractor()
	ws, err := extractor.extractToDirWithLimits(zipPath, t.TempDir(), limits)
	if ws != nil {
		defer func() { _ = ws.Cleanup() }()
	}
	if !errors.Is(err, shared.ErrDecompressionRatioTooHigh) {
		t.Fatalf("expected ErrDecompressionRatioTooHigh, got: %v", err)
	}
}

// TestZipExtractor_ExtractToDirWithLimits_AllowsLowRatioContentUnderFloor
// 40 字节内容未达到 50 字节比例检查门槛，验证小文件不因比例阈值被拒绝。
func TestZipExtractor_ExtractToDirWithLimits_AllowsLowRatioContentUnderFloor(t *testing.T) {
	zipPath := writeRawZip(t, map[string]string{
		"small.txt": strings.Repeat("A", 40),
	})
	limits := shared.ExtractionLimits{
		MaxEntryBytes:   1 << 30,
		MaxTotalBytes:   1 << 30,
		MaxRatio:        3,
		RatioFloorBytes: 50,
	}

	extractor := NewZipExtractor()
	ws, err := extractor.extractToDirWithLimits(zipPath, t.TempDir(), limits)
	if err != nil {
		t.Fatalf("expected success for small under-floor content, got: %v", err)
	}
	defer func() { _ = ws.Cleanup() }()
}
