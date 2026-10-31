// 本文件在临时目录中验证附件上传、列表、删除、重命名、文件名约束和体积限制。
// 上传流由测试创建并关闭，目录由测试框架清理，不接触运行实例的业务附件。
package assets

import (
	"errors"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/testutil"
)

// 通常用例给上传流一千零二十四字节额度，超限用例单独传入更小上限。
const testAssetMaxBytes int64 = 1024

// 上传一个附件后检查公开地址列表，确认地址中的页面 ID 与页面路径段分离。
func TestSaveAndListAsset(t *testing.T) {
	tmp := t.TempDir()
	page := &tree.PageNode{Slug: "lonely-page", ID: "a7b3"}
	// 准备页面目录与正文样本，附件仍由服务写入独立的 assets/页面ID 目录。
	pagePath := filepath.Join(tmp, "lonely-page")
	if err := os.MkdirAll(pagePath, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagePath, "index.md"), []byte("# Lonely Page"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	service := NewAssetService(tmp, tree.NewSlugService())

	file, name, err := testutil.CreateMultipartFile("my-image.png", []byte("hello image"))
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			t.Fatalf("Close() error: %v", err)
		}
	}()

	url, err := service.SaveAssetForPage(page, file, name, testAssetMaxBytes)
	if err != nil {
		t.Fatalf("SaveAsset failed: %v", err)
	}

	if url == "" {
		t.Fatalf("expected public URL, got empty string")
	}

	files, err := service.ListAssetsForPage(page)
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	if len(files) != 1 || files[0] != "/assets/a7b3/my-image.png" {
		t.Errorf("unexpected asset list: %v", files)
	}
}

// 先上传附件，再调用页面级附件删除入口，确认列表对已删除目录返回空结果。
// 本用例没有删除页面节点，也不验证页面树的删除流程。
func TestDeletePageAndEnsureAllAssetsAreDeleted(t *testing.T) {
	tmp := t.TempDir()
	page := &tree.PageNode{Slug: "lonely-page", ID: "a7b3"}
	// 创建页面正文作为场景数据，删除入口只针对附件目录。
	pagePath := filepath.Join(tmp, "lonely-page")
	if err := os.MkdirAll(pagePath, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagePath, "index.md"), []byte("# Lonely Page"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	service := NewAssetService(tmp, tree.NewSlugService())

	file, name, err := testutil.CreateMultipartFile("my-image.png", []byte("hello image"))
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			t.Fatalf("Close() error: %v", err)
		}
	}()

	_, err = service.SaveAssetForPage(page, file, name, testAssetMaxBytes)
	if err != nil {
		t.Fatalf("SaveAsset failed: %v", err)
	}

	err = service.DeleteAllAssetsForPage(page)
	if err != nil {
		t.Fatalf("DeletePage failed: %v", err)
	}

	files, err := service.ListAssetsForPage(page)
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected no assets, got %d", len(files))
	}
}

// 连续上传三份同名附件，验证名称冲突处理保留三条文件记录。
// 断言不限定重名后的具体序号文本；此样本的空页面 ID 使附件直接落在附件根目录。
func TestSlugCollision(t *testing.T) {
	tmp := t.TempDir()
	page := &tree.PageNode{Slug: "collision-page"}
	service := NewAssetService(tmp, tree.NewSlugService())

	// 每轮重新创建可读流，延迟关闭捕获该轮句柄，避免复用已读到末尾的上传流。
	for i := 0; i < 3; i++ {
		file, name, err := testutil.CreateMultipartFile("logo.png", []byte("image"))
		if err != nil {
			t.Fatalf("testutil.CreateMultipartFile failed: %v", err)
		}
		defer func(f multipart.File) {
			if err := f.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}
		}(file)

		_, err = service.SaveAssetForPage(page, file, name, testAssetMaxBytes)
		if err != nil {
			t.Fatalf("upload %d failed: %v", i, err)
		}
	}

	files, err := service.ListAssetsForPage(page)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 files, got %d", len(files))
	}
}

// 同扩展名的合法主体重命名后，列表应只包含新的公开地址。
func TestAssetRename(t *testing.T) {
	tmp := t.TempDir()
	page := &tree.PageNode{Slug: "rename-page", ID: "c3d4"}
	// 页面正文保持原路径，附件名称在独立的页面 ID 目录内变化。
	pagePath := filepath.Join(tmp, "rename-page")
	if err := os.MkdirAll(pagePath, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagePath, "index.md"), []byte("# Rename Page"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	service := NewAssetService(tmp, tree.NewSlugService())

	file, name, err := testutil.CreateMultipartFile("old-name.png", []byte("old image"))
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() {
		err := file.Close()
		if err != nil {
			t.Fatalf("Close() error: %v", err)
		}
	}()

	if _, err := service.SaveAssetForPage(page, file, name, testAssetMaxBytes); err != nil {
		t.Fatalf("SaveAsset failed: %v", err)
	}

	newName := "new-name.png"
	newUrl, err := service.RenameAsset(page, name, newName)
	if err != nil {
		t.Fatalf("RenameAsset failed: %v", err)
	}

	if newUrl != "" && strings.Contains(newUrl, newName) == false {
		t.Errorf("expected new URL to contain new name %s, got %s", newName, newUrl)
	}

	files, err := service.ListAssetsForPage(page)
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	expectedURL := "/assets/c3d4/new-name.png"
	if len(files) != 1 || files[0] != expectedURL {
		t.Errorf("unexpected asset list after rename: %v", files)
	}
}

// 附件目录存在而目标文件缺失时，删除必须返回 asset_not_found 本地化错误。
func TestDeleteMissingAssetReturnsNotFound(t *testing.T) {
	tmp := t.TempDir()
	page := &tree.PageNode{Slug: "delete-page", ID: "delete-page-id"}
	service := NewAssetService(tmp, tree.NewSlugService())

	pageAssetDir := filepath.Join(service.GetAssetsDir(), page.ID)
	if err := os.MkdirAll(pageAssetDir, 0755); err != nil {
		t.Fatalf("failed to create asset directory: %v", err)
	}

	err := service.DeleteAsset(page, "missing.png")
	if err == nil {
		t.Fatal("expected delete to fail for missing asset")
	}

	localized, ok := sharederrors.AsLocalizedError(err)
	if !ok {
		t.Fatalf("expected localized error, got %T", err)
	}

	if localized.Code != "asset_not_found" {
		t.Fatalf("expected asset_not_found, got %s", localized.Code)
	}
}

// 将 Windows 风格输入与输出都按正斜杠比较，验证两类路径拼接共用规范化处理。
func TestAssetDiskPaths_WindowsPath(t *testing.T) {
	if got, want := strings.ReplaceAll(assetPageDiskPath(`C:\wiki\data\assets`, "a7b3"), `\`, `/`), `C:/wiki/data/assets/a7b3`; got != want {
		t.Fatalf("asset page path = %q, want %q", got, want)
	}
	if got, want := strings.ReplaceAll(assetFileDiskPath(`C:\wiki\data\assets\a7b3`, "my-image.png"), `\`, `/`), `C:/wiki/data/assets/a7b3/my-image.png`; got != want {
		t.Fatalf("asset file path = %q, want %q", got, want)
	}
}

// 验证普通名称可通过，空值、点路径段和包含任一方向分隔符的名称被拒绝。
// 样本覆盖路径分隔限制，不代表覆盖操作系统的全部文件名约束。
func TestValidateFilename(t *testing.T) {
	good := []string{"my-image.png", "file.jpg", "a", "foo-bar.webp"}
	for _, name := range good {
		if err := validateFilename(name); err != nil {
			t.Errorf("validateFilename(%q) unexpected error: %v", name, err)
		}
	}

	bad := []string{
		"",
		".",
		"..",
		"../etc/passwd",
		"../../users.db",
		"foo/bar.png",
		`foo\bar.png`,
	}
	for _, name := range bad {
		if err := validateFilename(name); err == nil {
			t.Errorf("validateFilename(%q) should have returned an error", name)
		}
	}
}

// 在页面附件目录已经存在的前提下逐个提交路径穿越名称，
// 确认删除入口在接触目标文件前返回 asset_invalid_name。
func TestDeleteAsset_PathTraversal(t *testing.T) {
	tmp := t.TempDir()
	page := &tree.PageNode{Slug: "test-page", ID: "traversal-delete"}
	service := NewAssetService(tmp, tree.NewSlugService())

	pageAssetDir := filepath.Join(service.GetAssetsDir(), page.ID)
	if err := os.MkdirAll(pageAssetDir, 0755); err != nil {
		t.Fatalf("failed to create asset directory: %v", err)
	}

	traversalNames := []string{
		"../../users.db",
		"../other-page/secret.png",
		"..",
		".",
		"",
		"foo/bar.png",
		`foo\bar.png`,
	}
	for _, name := range traversalNames {
		err := service.DeleteAsset(page, name)
		if err == nil {
			t.Errorf("DeleteAsset(%q) should have returned an error", name)
			continue
		}
		localized, ok := sharederrors.AsLocalizedError(err)
		if !ok {
			t.Errorf("DeleteAsset(%q): expected localized error, got %T: %v", name, err, err)
			continue
		}
		if localized.Code != "asset_invalid_name" {
			t.Errorf("DeleteAsset(%q): expected asset_invalid_name, got %s", name, localized.Code)
		}
	}
}

// 重命名使用非法旧名称与合法新名称，确认旧名称检查拒绝目录逃逸样本。
// 本用例不验证非法新名称，那部分由生产代码的路径段校验负责。
func TestRenameAsset_OldFilenamePathTraversal(t *testing.T) {
	tmp := t.TempDir()
	page := &tree.PageNode{Slug: "test-page", ID: "traversal-rename"}
	service := NewAssetService(tmp, tree.NewSlugService())

	pageAssetDir := filepath.Join(service.GetAssetsDir(), page.ID)
	if err := os.MkdirAll(pageAssetDir, 0755); err != nil {
		t.Fatalf("failed to create asset directory: %v", err)
	}

	traversalNames := []string{
		"../../users.db",
		"../other-page/secret.png",
		"..",
		".",
		"",
		"foo/bar.png",
		`foo\bar.png`,
	}
	for _, name := range traversalNames {
		_, err := service.RenameAsset(page, name, "new-name.png")
		if err == nil {
			t.Errorf("RenameAsset(oldFilename=%q) should have returned an error", name)
			continue
		}
		localized, ok := sharederrors.AsLocalizedError(err)
		if !ok {
			t.Errorf("RenameAsset(oldFilename=%q): expected localized error, got %T: %v", name, err, err)
			continue
		}
		if localized.Code != "asset_invalid_name" {
			t.Errorf("RenameAsset(oldFilename=%q): expected asset_invalid_name, got %s", name, localized.Code)
		}
	}
}

// 公开地址必须始终使用 URL 正斜杠，不把宿主系统路径分隔符带入返回值。
func TestAssetPublicPath_UsesForwardSlashes(t *testing.T) {
	service := NewAssetService(t.TempDir(), tree.NewSlugService())
	page := &tree.PageNode{ID: "a7b3"}

	if got, want := service.buildPublicPath(page, "my-image.png"), "/assets/a7b3/my-image.png"; got != want {
		t.Fatalf("public path = %q, want %q", got, want)
	}
}

// 将三十二字节上传流限制为八字节，确认错误链保留 ErrFileTooLarge，
// 并检查页面附件目录内没有正式文件或遗留临时文件。
func TestSaveAssetForPage_TooLarge_DoesNotLeavePartialFile(t *testing.T) {
	tmp := t.TempDir()
	page := &tree.PageNode{Slug: "limit-page", ID: "limit-page-id"}
	service := NewAssetService(tmp, tree.NewSlugService())

	file, name, err := testutil.CreateMultipartFile("too-large.bin", []byte(strings.Repeat("a", 32)))
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Close() error: %v", err)
		}
	}()

	_, err = service.SaveAssetForPage(page, file, name, 8)
	if !errors.Is(err, shared.ErrFileTooLarge) {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}

	assetDir := filepath.Join(service.GetAssetsDir(), page.ID)
	entries, readErr := os.ReadDir(assetDir)
	if readErr != nil {
		t.Fatalf("failed to read asset directory: %v", readErr)
	}

	if len(entries) != 0 {
		t.Fatalf("expected no files after failed upload, got %d", len(entries))
	}
}
