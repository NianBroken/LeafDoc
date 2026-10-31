// 本文件验证临时文件提交、跨平台路径处理和解压复制的三类限额错误。
// 文件写入均使用测试临时目录，不触碰实际实例的正文和附件。
package shared

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestWriteFileAtomic_WritesToTargetFile 在目标不存在时写入字节，并从最终路径核对正文。
func TestWriteFileAtomic_WritesToTargetFile(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "page.md")

	if err := WriteFileAtomic(target, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic err: %v", err)
	}

	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile err: %v", err)
	}
	if string(raw) != "hello" {
		t.Fatalf("content = %q", string(raw))
	}
}

// TestAtomicWriteDir_WindowsPath 将 Windows 风格路径作为输入，在各平台统一比较斜线路径。
// 两个案例分别覆盖页面目录与多级附件目录，不要求这些示例路径实际存在。
func TestAtomicWriteDir_WindowsPath(t *testing.T) {
	testCases := []struct {
		name string
		path string
		want string
	}{
		{
			name: "markdown page",
			path: `C:\wiki\data\root\page.md`,
			want: `C:/wiki/data/root`,
		},
		{
			name: "asset file",
			path: `C:\wiki\data\assets\a7b3\image.png`,
			want: `C:/wiki/data/assets/a7b3`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.ReplaceAll(atomicWriteDir(tc.path), `\`, `/`)
			if got != tc.want {
				t.Fatalf("dir = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestWriteStreamAtomic_WritesToTargetFile 检查数据流提交后的正文和非 Windows 文件权限。
// Windows 不参与 Unix 权限位断言，但仍必须满足内容写入和目标存在要求。
func TestWriteStreamAtomic_WritesToTargetFile(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "asset.bin")

	if err := WriteStreamAtomic(target, bytes.NewBufferString("hello stream"), 1024, 0o644); err != nil {
		t.Fatalf("WriteStreamAtomic err: %v", err)
	}

	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile err: %v", err)
	}
	if string(raw) != "hello stream" {
		t.Fatalf("content = %q", string(raw))
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat err: %v", err)
	}
	if runtime.GOOS != "windows" {
		if got := info.Mode().Perm(); got != 0o644 {
			t.Fatalf("permissions = %04o, want 0644", got)
		}
	}
}

// 下列用例分别检查单项超限、总预算超限和压缩大小声明溢出时的错误类别。
// 每个测试都放宽与其目标无关的限制，使断言能够对应到指定的判断路径。

// TestCopyWithBudget_ReportsErrFileTooLarge_WhenEntryCapIsBinding 验证较小的单项上限决定错误类别。
func TestCopyWithBudget_ReportsErrFileTooLarge_WhenEntryCapIsBinding(t *testing.T) {
	limits := ExtractionLimits{MaxEntryBytes: 10, MaxTotalBytes: 1000, MaxRatio: 1 << 30, RatioFloorBytes: 1 << 30}
	budget := NewSizeBudget(limits.MaxTotalBytes)

	err := CopyWithBudget(&bytes.Buffer{}, strings.NewReader(strings.Repeat("A", 20)), 0, limits, budget)
	if !errors.Is(err, ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got: %v", err)
	}
}

// TestCopyWithBudget_ReportsErrCumulativeSizeTooLarge_WhenBudgetCapIsBinding 验证共享剩余额度更小时返回总量错误。
func TestCopyWithBudget_ReportsErrCumulativeSizeTooLarge_WhenBudgetCapIsBinding(t *testing.T) {
	limits := ExtractionLimits{MaxEntryBytes: 1000, MaxTotalBytes: 1000, MaxRatio: 1 << 30, RatioFloorBytes: 1 << 30}
	budget := NewSizeBudget(limits.MaxTotalBytes)
	budget.remaining = 10 // 直接设置剩余预算，表达前面条目已经消耗大部分额度的状态。

	err := CopyWithBudget(&bytes.Buffer{}, strings.NewReader(strings.Repeat("A", 20)), 0, limits, budget)
	if !errors.Is(err, ErrCumulativeSizeTooLarge) {
		t.Fatalf("expected ErrCumulativeSizeTooLarge, got: %v", err)
	}
}

// TestCopyWithBudget_ClampsOverflowingDeclaredCompressedSize_KeepsRatioCheckFailClosed 检查转换溢出不会停用比例限制。
func TestCopyWithBudget_ClampsOverflowingDeclaredCompressedSize_KeepsRatioCheckFailClosed(t *testing.T) {
	limits := ExtractionLimits{MaxEntryBytes: 1 << 30, MaxTotalBytes: 1 << 30, MaxRatio: 3, RatioFloorBytes: 50}
	budget := NewSizeBudget(limits.MaxTotalBytes)

	// 最大 uint64 转为 int64 后为负，复制函数会将压缩大小置为 1。
	// 五千字节输出超过五十字节起检值，必须触发比例错误，而非因负值跳过检查。
	declaredCompressedSize := uint64(math.MaxUint64)
	err := CopyWithBudget(&bytes.Buffer{}, strings.NewReader(strings.Repeat("A", 5000)), declaredCompressedSize, limits, budget)
	if !errors.Is(err, ErrDecompressionRatioTooHigh) {
		t.Fatalf("expected the ratio check to stay active (fail-closed) for an overflowing declared compressed size, got: %v", err)
	}
}
