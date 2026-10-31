package avatar

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	coreavatar "github.com/NianBroken/LeafDoc/internal/core/avatar"
)

// fakeMultipartFile 为内存读取器提供空关闭方法，满足上传用例需要的 multipart.File 接口。
type fakeMultipartFile struct {
	*bytes.Reader
}

// Close 无实际文件句柄需要释放，始终成功。
func (f fakeMultipartFile) Close() error { return nil }

// newMultipartFile 将图片字节包装为支持读取、定位与关闭的测试上传文件。
func newMultipartFile(data []byte) multipart.File {
	return fakeMultipartFile{bytes.NewReader(data)}
}

// solidSquarePNG 绘制 64 像素正方形并编码为真实 PNG，使测试经过图片解码与保存路径。
func solidSquarePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.RGBA{50, 60, 70, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buf.Bytes()
}

// avatarFixture 让上传和删除用例共享临时目录下的同一个头像服务。
type avatarFixture struct {
	upload *UploadAvatarUseCase
	del    *DeleteAvatarUseCase
	dir    string
}

// newAvatarFixture 使用自动清理目录构造真实服务，初始化失败立即终止当前测试。
func newAvatarFixture(t *testing.T) *avatarFixture {
	t.Helper()
	dir := t.TempDir()
	svc, err := coreavatar.NewAvatarService(dir)
	if err != nil {
		t.Fatalf("NewAvatarService: %v", err)
	}
	return &avatarFixture{upload: NewUploadAvatarUseCase(svc), del: NewDeleteAvatarUseCase(svc), dir: dir}
}

// uploadPNG 每次生成有效 PNG，并使用传入文件名触发扩展名校验。
func (f *avatarFixture) uploadPNG(t *testing.T, userID, filename string) error {
	t.Helper()
	return f.upload.Execute(context.Background(), UploadAvatarInput{
		UserID:   userID,
		File:     newMultipartFile(solidSquarePNG(t)),
		Filename: filename,
	})
}

// avatarPath 按实际服务约定定位用户 PNG 文件，用于检查写入与删除结果。
func (f *avatarFixture) avatarPath(userID string) string {
	return filepath.Join(f.dir, "avatars", userID+".png")
}

// TestUploadAvatarUseCase_Execute_ValidImage_WritesAvatarFile 确认有效图片实际形成用户头像文件。
func TestUploadAvatarUseCase_Execute_ValidImage_WritesAvatarFile(t *testing.T) {
	f := newAvatarFixture(t)

	if err := f.uploadPNG(t, "user-1", "avatar.png"); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if _, err := os.Stat(f.avatarPath("user-1")); err != nil {
		t.Fatalf("expected avatar file to exist: %v", err)
	}
}

// TestUploadAvatarUseCase_Execute_InvalidExtension_ReturnsError 使用有效 PNG 配不允许的名称扩展名。
// 由此验证名称限制不会仅因内容能够解码而被绕过。
func TestUploadAvatarUseCase_Execute_InvalidExtension_ReturnsError(t *testing.T) {
	f := newAvatarFixture(t)

	if err := f.uploadPNG(t, "user-1", "avatar.bmp"); err == nil {
		t.Fatal("expected an error for a disallowed extension")
	}
}

// TestDeleteAvatarUseCase_Execute_RemovesUploadedAvatar 先核对文件存在，再要求删除后为不存在错误。
func TestDeleteAvatarUseCase_Execute_RemovesUploadedAvatar(t *testing.T) {
	f := newAvatarFixture(t)

	if err := f.uploadPNG(t, "user-1", "avatar.png"); err != nil {
		t.Fatalf("Execute (upload): %v", err)
	}
	path := f.avatarPath("user-1")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected avatar file to exist before delete: %v", err)
	}

	if err := f.del.Execute(context.Background(), DeleteAvatarInput{UserID: "user-1"}); err != nil {
		t.Fatalf("Execute (delete): %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected avatar file removed, got err=%v", err)
	}
}

// TestDeleteAvatarUseCase_Execute_NothingUploaded_NoOp 确认未上传头像时删除仍成功。
func TestDeleteAvatarUseCase_Execute_NothingUploaded_NoOp(t *testing.T) {
	f := newAvatarFixture(t)

	if err := f.del.Execute(context.Background(), DeleteAvatarInput{UserID: "user-with-no-avatar"}); err != nil {
		t.Fatalf("expected no error deleting a nonexistent avatar, got: %v", err)
	}
}
