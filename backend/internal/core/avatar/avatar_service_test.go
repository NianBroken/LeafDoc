// 本文件用内存生成图像和测试临时目录核验头像格式、尺寸、裁剪、删除与路径限制。
package avatar

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/webp"
)

// fakeMultipartFile 在字节读取器上提供空关闭方法，使内存数据满足 multipart.File。
type fakeMultipartFile struct {
	*bytes.Reader
}

func (f fakeMultipartFile) Close() error { return nil }

// newMultipartFile 为每次上传创建独立读取位置，不复制底层字节切片。
func newMultipartFile(data []byte) multipart.File {
	return fakeMultipartFile{bytes.NewReader(data)}
}

// newTestAvatarService 使用自动清理的临时目录创建服务，同时返回目录供文件断言。
func newTestAvatarService(t *testing.T) (*AvatarService, string) {
	t.Helper()
	dir := t.TempDir()
	svc, err := NewAvatarService(dir)
	if err != nil {
		t.Fatalf("NewAvatarService() error: %v", err)
	}
	return svc, dir
}

// quadrantImage 按水平和垂直中点把图像分成四色区域，便于观察裁剪后保留的内容。
// 返回指定宽高的 RGBA 图像，颜色分布由像素坐标明确决定。
func quadrantImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	colors := [4]color.RGBA{
		{255, 0, 0, 255},   // 左上为红色。
		{0, 255, 0, 255},   // 右上为绿色。
		{0, 0, 255, 255},   // 左下为蓝色。
		{255, 255, 0, 255}, // 右下为黄色。
	}
	midX, midY := width/2, height/2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var c color.RGBA
			switch {
			case x < midX && y < midY:
				c = colors[0]
			case x >= midX && y < midY:
				c = colors[1]
			case x < midX && y >= midY:
				c = colors[2]
			default:
				c = colors[3]
			}
			img.Set(x, y, c)
		}
	}
	return img
}

// encodePNG 编码测试图像，编码失败直接终止当前测试。
func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error: %v", err)
	}
	return buf.Bytes()
}

// encodeJPEG 使用库默认质量生成 JPEG 测试载荷。
func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("jpeg.Encode() error: %v", err)
	}
	return buf.Bytes()
}

// encodeGIF 将图像编码为单张 GIF，测试只核验该静态载荷的上传结果。
func encodeGIF(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		t.Fatalf("gif.Encode() error: %v", err)
	}
	return buf.Bytes()
}

// solidSquarePNG 生成指定边长的纯色正方形，供不需要复杂图案的上传测试复用。
func solidSquarePNG(t *testing.T, size int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.RGBA{10, 20, 30, 255})
		}
	}
	return encodePNG(t, img)
}

// 删除构造时建立的空头像目录后上传，确认上传路径会重新建立目录并生成文件。
func TestAvatarService_UploadAvatar_RecreatesMissingDirectory(t *testing.T) {
	svc, dir := newTestAvatarService(t)
	if err := os.Remove(filepath.Join(dir, "avatars")); err != nil {
		t.Fatal(err)
	}
	if err := svc.UploadAvatar("user-1", newMultipartFile(solidSquarePNG(t, 100)), "avatar.png"); err != nil {
		t.Fatal(err)
	}
	if _, exists := svc.AvatarPath("user-1"); !exists {
		t.Fatal("uploaded avatar does not exist")
	}
}

// JPEG、PNG 和 GIF 输入都输出固定尺寸的 PNG，不保留输入文件格式。
func TestAvatarService_UploadAvatar_ValidFormats_ProducesTargetSizePNG(t *testing.T) {
	square := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 300; x++ {
			square.Set(x, y, color.RGBA{100, 150, 200, 255})
		}
	}

	tests := []struct {
		name string
		data []byte
		ext  string
	}{
		{"jpeg", encodeJPEG(t, square), ".jpg"},
		{"png", encodePNG(t, square), ".png"},
		{"gif", encodeGIF(t, square), ".gif"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, dir := newTestAvatarService(t)
			if err := svc.UploadAvatar("user-1", newMultipartFile(tt.data), "avatar"+tt.ext); err != nil {
				t.Fatalf("UploadAvatar() error: %v", err)
			}

			path := filepath.Join(dir, "avatars", "user-1.png")
			f, err := os.Open(path)
			if err != nil {
				t.Fatalf("expected avatar file at %s: %v", path, err)
			}
			defer func() { _ = f.Close() }()

			cfg, err := png.DecodeConfig(f)
			if err != nil {
				t.Fatalf("stored avatar is not a valid PNG: %v", err)
			}
			if cfg.Width != TargetSize || cfg.Height != TargetSize {
				t.Errorf("expected %dx%d, got %dx%d", TargetSize, TargetSize, cfg.Width, cfg.Height)
			}
		})
	}
}

func TestAvatarService_UploadAvatar_WebpFormat_ProducesTargetSizePNG(t *testing.T) {
	// 固定字节作为 WebP 输入，先确认解码成功，再检查上传后的 PNG 格式和目标尺寸。
	// 测试不在运行时调用外部图像生成器。
	data := []byte{
		0x52, 0x49, 0x46, 0x46, 0x24, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
		0x56, 0x50, 0x38, 0x4c, 0x17, 0x00, 0x00, 0x00, 0x2f, 0x63, 0xc0, 0x18,
		0x00, 0x07, 0x50, 0x8a, 0x2a, 0xd4, 0xa3, 0xff, 0x61, 0x00, 0x12, 0xc2,
		0xff, 0xfd, 0x52, 0x44, 0xff, 0x53, 0x89, 0x00,
	}
	if _, err := webp.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("test fixture is not a valid webp: %v", err)
	}

	svc, dir := newTestAvatarService(t)
	if err := svc.UploadAvatar("user-webp", newMultipartFile(data), "avatar.webp"); err != nil {
		t.Fatalf("UploadAvatar() error: %v", err)
	}

	path := filepath.Join(dir, "avatars", "user-webp.png")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("expected avatar file at %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("stored avatar is not a valid PNG: %v", err)
	}
	if cfg.Width != TargetSize || cfg.Height != TargetSize {
		t.Errorf("expected %dx%d, got %dx%d", TargetSize, TargetSize, cfg.Width, cfg.Height)
	}
}

// 即使字节是有效 PNG，不允许的扩展名仍使上传返回错误，此处不检查具体错误码。
func TestAvatarService_UploadAvatar_RejectedExtension_ReturnsInvalidTypeError(t *testing.T) {
	svc, _ := newTestAvatarService(t)
	data := solidSquarePNG(t, 100)

	err := svc.UploadAvatar("user-1", newMultipartFile(data), "avatar.bmp")
	if err == nil {
		t.Fatal("expected an error for a disallowed extension")
	}
}

// 合法扩展名不能替代图像解码，非图像字节必须返回错误。
func TestAvatarService_UploadAvatar_CorruptBytes_ReturnsDecodeFailedError(t *testing.T) {
	svc, _ := newTestAvatarService(t)
	garbage := []byte("this is not an image, just plain garbage bytes")

	err := svc.UploadAvatar("user-1", newMultipartFile(garbage), "avatar.png")
	if err == nil {
		t.Fatal("expected a decode error for corrupt image bytes")
	}
}

func TestAvatarService_UploadAvatar_OversizedUpload_Rejected(t *testing.T) {
	svc, _ := newTestAvatarService(t)

	// 使用固定种子的像素噪声生成较难压缩的 PNG，并先断言编码字节确实超过上传上限。
	// 检查的是编码后文件大小，不以像素数量替代字节大小判断。
	size := 1600
	rng := rand.New(rand.NewSource(1))
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	noise := make([]byte, size*size*4)
	if _, err := rng.Read(noise); err != nil {
		t.Fatalf("rng.Read() error: %v", err)
	}
	copy(img.Pix, noise)
	data := encodePNG(t, img)
	if int64(len(data)) <= MaxUploadSize {
		t.Fatalf("test fixture too small to exceed MaxUploadSize: %d bytes", len(data))
	}

	err := svc.UploadAvatar("user-1", newMultipartFile(data), "avatar.png")
	if err == nil {
		t.Fatal("expected an error for an oversized upload")
	}
}

func TestAvatarService_UploadAvatar_NonSquareInput_CropsCenteredNotCorner(t *testing.T) {
	svc, dir := newTestAvatarService(t)

	// 四百乘二百的宽图居中裁成正方形，横向范围为一百到三百之间的二百像素。
	// 四个颜色区域仍应在裁剪与缩放后保留，贴边裁剪不会呈现相同分布。
	src := quadrantImage(400, 200)
	data := encodePNG(t, src)

	if err := svc.UploadAvatar("user-1", newMultipartFile(data), "avatar.png"); err != nil {
		t.Fatalf("UploadAvatar() error: %v", err)
	}

	path := filepath.Join(dir, "avatars", "user-1.png")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("expected avatar file at %s: %v", path, err)
	}
	defer func() { _ = f.Close() }()

	out, err := png.Decode(f)
	if err != nil {
		t.Fatalf("stored avatar is not a valid PNG: %v", err)
	}
	if out.Bounds().Dx() != TargetSize || out.Bounds().Dy() != TargetSize {
		t.Fatalf("expected %dx%d output, got %dx%d", TargetSize, TargetSize, out.Bounds().Dx(), out.Bounds().Dy())
	}

	// 在输出四角向内取样，避开边界缩放混色，以通道阈值归类并统计不同颜色标签。
	// 当前断言检查标签集合至少四项，不逐角强制对应某个指定颜色。
	inset := TargetSize / 8
	corners := map[string][2]int{
		"top-left":     {inset, inset},
		"top-right":    {TargetSize - 1 - inset, inset},
		"bottom-left":  {inset, TargetSize - 1 - inset},
		"bottom-right": {TargetSize - 1 - inset, TargetSize - 1 - inset},
	}
	seen := map[string]bool{}
	closestColorName := func(c color.Color) string {
		r, g, b, _ := c.RGBA()
		r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
		switch {
		case r8 > 150 && g8 < 100 && b8 < 100:
			return "red"
		case r8 < 100 && g8 > 150 && b8 < 100:
			return "green"
		case r8 < 100 && g8 < 100 && b8 > 150:
			return "blue"
		case r8 > 150 && g8 > 150 && b8 < 100:
			return "yellow"
		default:
			return "unknown"
		}
	}
	for corner, xy := range corners {
		name := closestColorName(out.At(xy[0], xy[1]))
		seen[name] = true
		t.Logf("corner %s -> color %s", corner, name)
	}
	if len(seen) < 4 {
		t.Errorf("expected all 4 quadrant colors visible in a centered crop, saw: %v", seen)
	}
}

func TestAvatarService_DeleteAvatar_IdempotentAndNoOpWhenNothingUploaded(t *testing.T) {
	svc, dir := newTestAvatarService(t)

	// 尚未上传的用户没有头像文件，删除应直接成功。
	if err := svc.DeleteAvatar("no-such-user"); err != nil {
		t.Fatalf("DeleteAvatar() on nonexistent user error: %v", err)
	}

	data := solidSquarePNG(t, 100)
	if err := svc.UploadAvatar("user-1", newMultipartFile(data), "avatar.png"); err != nil {
		t.Fatalf("UploadAvatar() error: %v", err)
	}
	path := filepath.Join(dir, "avatars", "user-1.png")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected avatar file to exist before delete: %v", err)
	}

	if err := svc.DeleteAvatar("user-1"); err != nil {
		t.Fatalf("DeleteAvatar() error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected avatar file removed, got err=%v", err)
	}

	// 文件删除后再次调用仍成功，验证重复删除的结果一致。
	if err := svc.DeleteAvatar("user-1"); err != nil {
		t.Fatalf("DeleteAvatar() second call error: %v", err)
	}
}

// 上传前报告不存在，上传后返回可被 Stat 的真实文件路径。
func TestAvatarService_AvatarPath_ReportsExistence(t *testing.T) {
	svc, _ := newTestAvatarService(t)

	if _, found := svc.AvatarPath("user-1"); found {
		t.Error("expected AvatarPath to report not-found before any upload")
	}

	data := solidSquarePNG(t, 100)
	if err := svc.UploadAvatar("user-1", newMultipartFile(data), "avatar.png"); err != nil {
		t.Fatalf("UploadAvatar() error: %v", err)
	}

	path, found := svc.AvatarPath("user-1")
	if !found {
		t.Fatal("expected AvatarPath to report found after upload")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("AvatarPath returned a path that does not exist: %v", err)
	}
}

// 含父目录、两种分隔符或绝对路径前缀的用户 ID 均被拒绝。
func TestAvatarService_UploadAvatar_UnsafeUserID_Rejected(t *testing.T) {
	svc, _ := newTestAvatarService(t)
	data := solidSquarePNG(t, 100)

	for _, id := range []string{"../escape", "a/b", "a\\b", "/abs"} {
		if err := svc.UploadAvatar(id, newMultipartFile(data), "avatar.png"); err == nil {
			t.Errorf("expected UploadAvatar to reject unsafe user id %q", id)
		}
	}
}

var _ io.ReaderAt = fakeMultipartFile{} // 编译时确认测试适配器实现随机偏移读取接口。
