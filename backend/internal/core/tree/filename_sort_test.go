package tree

import (
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// namedSortEntry 用文件名实现目录项接口，排序测试无需实际建立文件。
// 除名称外均返回固定值，Info 返回无效操作错误以避免把替身当作真实磁盘条目。
type namedSortEntry string

func (e namedSortEntry) Name() string               { return string(e) }
func (e namedSortEntry) IsDir() bool                { return false }
func (e namedSortEntry) Type() fs.FileMode          { return 0 }
func (e namedSortEntry) Info() (fs.FileInfo, error) { return nil, os.ErrInvalid }

// TestSortReconstructionEntries_MixedCaseAndUnicode_PreservesOrder
// 明确检查大小写名称与非 ASCII 字母的完整排序结果，包括同小写名称的次级顺序。
func TestSortReconstructionEntries_MixedCaseAndUnicode_PreservesOrder(t *testing.T) {
	entries := []os.DirEntry{namedSortEntry("z.md"), namedSortEntry("ä.md"), namedSortEntry("b.md"), namedSortEntry("A.md"), namedSortEntry("B.md"), namedSortEntry("a.md"), namedSortEntry("Ä.md")}
	sortReconstructionEntries(entries)
	got := []string{}
	for _, e := range entries {
		got = append(got, e.Name())
	}
	want := []string{"A.md", "a.md", "B.md", "b.md", "z.md", "Ä.md", "ä.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

// BenchmarkSortReconstructionEntries 分别测量三个规模的逆序文件名排序。
// 每轮先复制原输入再排序，并报告分配次数，避免只测已经排序好的数组。
func BenchmarkSortReconstructionEntries(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			original := make([]os.DirEntry, count)
			for i := range original {
				original[i] = namedSortEntry(fmt.Sprintf("Page-%05d.md", count-i))
			}
			entries := make([]os.DirEntry, count)
			b.ReportAllocs()
			for b.Loop() {
				copy(entries, original)
				sortReconstructionEntries(entries)
			}
		})
	}
}

// TestSortReconstructionEntries_MatchesLegacyComparator 遍历名称对，独立计算期望首项。
// 测试名称沿用既有标识，比较依据是 ToLower 后的字符串次序与原名次级次序。
func TestSortReconstructionEntries_MatchesLegacyComparator(t *testing.T) {
	// 两两组合覆盖 Unicode 小写转换和原字符串比较，不以语言字典顺序排序。
	names := []string{"a", "A", "İ", "i", "Ä", "ä", "ß", "ss", "Z"}
	for _, a := range names {
		for _, z := range names {
			entries := []os.DirEntry{namedSortEntry(a), namedSortEntry(z)}
			sortReconstructionEntries(entries)
			wantFirst := a
			// 期望值按 ToLower 明确计算，不能用 EqualFold 替代小写字符串的大小比较。
			foldedA, foldedZ := strings.ToLower(a), strings.ToLower(z)
			if foldedZ < foldedA || (foldedZ == foldedA && z < a) {
				wantFirst = z
			}
			if entries[0].Name() != wantFirst {
				t.Fatalf("%q,%q: got %q want %q", a, z, entries[0].Name(), wantFirst)
			}
		}
	}
}

// BenchmarkSortReconstructionReadDirOrder 先建立原文件名文字排序的输入，再测重建排序。
// 分别覆盖全小写与交替大小写名称，复制操作包含在每轮测量中。
func BenchmarkSortReconstructionReadDirOrder(b *testing.B) {
	for _, mixed := range []bool{false, true} {
		name := "Lowercase"
		if mixed {
			name = "MixedCase"
		}
		b.Run(name, func(b *testing.B) {
			original := make([]os.DirEntry, 10000)
			for i := range original {
				prefix := "page"
				if mixed && i%2 == 0 {
					prefix = "Page"
				}
				original[i] = namedSortEntry(fmt.Sprintf("%s-%05d.md", prefix, i))
			}
			sort.Slice(original, func(i, j int) bool { return original[i].Name() < original[j].Name() })
			entries := make([]os.DirEntry, len(original))
			b.ReportAllocs()
			for b.Loop() {
				copy(entries, original)
				sortReconstructionEntries(entries)
			}
		})
	}
}
