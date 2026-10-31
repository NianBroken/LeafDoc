// 本文件提供受字节预算约束的数据复制，供导入和恢复的解压流程使用。
// 限制在复制过程中检查，失败时目标中已经写入的数据由调用方负责清理。
package shared

import (
	"errors"
	"fmt"
	"io"
)

var (
	// 三种错误分别表示单项字节数、共享总预算和解压比例超限，可通过 errors.Is 区分。
	ErrFileTooLarge              = errors.New("file too large")
	ErrCumulativeSizeTooLarge    = errors.New("cumulative size too large")
	ErrDecompressionRatioTooHigh = errors.New("decompression ratio too high")
)

// ExtractionLimits 保存单项大小、总大小与解压比例参数。
// CopyWithBudget 直接读取单项和比例参数，总量限制由调用方用 MaxTotalBytes 创建共享预算。
type ExtractionLimits struct {
	MaxEntryBytes   int64 // 一次复制允许的字节数，超限返回 ErrFileTooLarge。
	MaxTotalBytes   int64 // 多次复制共用的初始预算，由 NewSizeBudget 接收。
	MaxRatio        int64 // 正数时参与比例检查，比较使用整数除法。
	RatioFloorBytes int64 // 已写字节严格超过此值后才检查比例。
}

// DefaultZipExtractionLimits 为常规解压提供单项 50 MiB、总量 1 GiB 的配置。
// 写出超过 1 MiB 后启用比例检查，MaxRatio 为 100，实际阈值按整数除法判断。
// 这是可修改的包变量，调用方接收结构值后使用该次复制获得的配置。
var DefaultZipExtractionLimits = ExtractionLimits{
	MaxEntryBytes:   50 * 1024 * 1024,
	MaxTotalBytes:   1024 * 1024 * 1024,
	MaxRatio:        100,
	RatioFloorBytes: 1024 * 1024,
}

// unrestrictedExtractionBytes 是放宽配置使用的有限上限，等于二的六十二次方。
// 它低于 int64 最大值，为复制时使用的 limit+1 留出空间，并不表示真正无限。
const unrestrictedExtractionBytes = int64(1) << 62

// UnrestrictedExtractionLimits 供离线恢复和既有快照恢复使用，放宽常规上传限制。
// 单项、总量、比例和比例起检字节数均使用同一大整数，大小上限仍然存在。
// 在正常预算范围内不会超过比例起检值，因此不会执行后续整数比例比较。
var UnrestrictedExtractionLimits = ExtractionLimits{
	MaxEntryBytes:   unrestrictedExtractionBytes,
	MaxTotalBytes:   unrestrictedExtractionBytes,
	MaxRatio:        unrestrictedExtractionBytes,
	RatioFloorBytes: unrestrictedExtractionBytes,
}

// SizeBudget 保存连续多次复制共享的剩余字节数。
// 成功复制才扣减预算，失败不会扣减已经写出的部分，调用方应中止并清理解压结果。
// 字段没有锁保护，同一预算不能由多个 goroutine 同时修改。
type SizeBudget struct {
	remaining int64
}

// NewSizeBudget 原样保存初始额度，不校验正负，也不与某个配置对象自动关联。
func NewSizeBudget(max int64) *SizeBudget {
	return &SizeBudget{remaining: max}
}

// ratioLimitedWriter 在底层写入完成后累计实际写出的字节，再判断解压比例。
// compressedSize 和 maxRatio 必须为正，written 必须超过 minFloor，才执行比较。
// 比例错误不能撤销已经写出的数据，也不使本次 Write 的返回字节数变为零。
type ratioLimitedWriter struct {
	dst            io.Writer
	compressedSize int64
	maxRatio       int64
	minFloor       int64
	written        int64
}

// Write 优先返回底层写入错误；底层成功后才检查当前累计值是否超过比例条件。
func (w *ratioLimitedWriter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	w.written += int64(n)
	if err != nil {
		return n, err
	}
	// 整数除法避免计算 compressedSize*maxRatio 时溢出，但阈值并非乘积比较的精确等价式。
	// 除法向零截断，written 至少达到 maxRatio*(compressedSize+1) 才满足此比较。
	// 压缩大小来自归档声明，函数不自行核实真实压缩字节数。
	if w.compressedSize > 0 && w.maxRatio > 0 && w.written > w.minFloor && w.written/w.maxRatio > w.compressedSize {
		return n, fmt.Errorf("%w: %d bytes decompressed from %d compressed bytes (exceeds %d:1)",
			ErrDecompressionRatioTooHigh, w.written, w.compressedSize, w.maxRatio)
	}
	return n, nil
}

// CopyWithBudget 在一次流式复制中检查单项大小、共享剩余额度和声明压缩大小对应的比例。
// 单项上限与剩余预算取较小者，并额外读取至多一字节识别超限；该字节也会写到 dst。
// 两个额度相等时超限归类为单项错误，只有预算严格更小时归类为总量错误。
// 声明压缩大小为零时跳过比例检查，比例错误由写入包装器在复制途中返回。
// budget 必须非 nil，配置需为 limit+1 留出整数范围；本函数不关闭 src 或 dst。
func CopyWithBudget(dst io.Writer, src io.Reader, declaredCompressedSize uint64, limits ExtractionLimits, budget *SizeBudget) error {
	limit := limits.MaxEntryBytes
	limitedByBudget := false
	if budget.remaining < limit {
		limit = budget.remaining
		limitedByBudget = true
	}
	// 负额度按零字节处理，仍读取一字节来区分空输入与超限输入。
	if limit < 0 {
		limit = 0
	}

	// uint64 中超过 int64 最大值的声明转换后为负数，将其置为 1。
	// 这样不会因 compressedSize 为负而跳过比例检查，零值则继续保留跳过语义。
	compressedSize := int64(declaredCompressedSize)
	if compressedSize < 0 {
		compressedSize = 1
	}

	rw := &ratioLimitedWriter{
		dst:            dst,
		compressedSize: compressedSize,
		maxRatio:       limits.MaxRatio,
		minFloor:       limits.RatioFloorBytes,
	}

	// 包装器的写入错误优先返回，只有复制无错才继续判断实际字节数和扣减预算。
	n, err := io.Copy(rw, io.LimitReader(src, limit+1))
	if err != nil {
		return err
	}
	if n > limit {
		if limitedByBudget {
			return fmt.Errorf("%w: %d bytes would exceed remaining budget of %d bytes", ErrCumulativeSizeTooLarge, n, budget.remaining)
		}
		return fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, n, limits.MaxEntryBytes)
	}
	// 仅完整通过本次检查后记账，后续文件共享扣减后的剩余额度。
	budget.remaining -= n
	return nil
}

// CopyWithLimit 最多复制 max 加一字节，超过 max 时返回可识别的 ErrFileTooLarge。
// 错误返回前超限字节已写到目标，调用方负责丢弃失败结果和关闭两端资源。
// 本函数不校验 max 的取值范围，调用方需避免 max+1 发生整数溢出。
func CopyWithLimit(dst io.Writer, src io.Reader, max int64) error {
	n, err := io.Copy(dst, io.LimitReader(src, max+1))
	if err != nil {
		return err
	}
	if n > max {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, n, max)
	}
	return nil
}
