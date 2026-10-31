// Package avatar 将上传图片居中裁为正方形、缩放为固定尺寸，再按用户 ID 保存为 PNG。
// 同一用户始终写入 avatars/<userID>.png，头像文件位于实例数据目录，不单独保存配置记录。
// 服务处理图像与文件，不在本包判断请求者是否有权修改对应用户的头像。
package avatar

import (
	"bytes"
	"fmt"
	"image"

	// 空白导入注册 GIF 和 JPEG 解码器，image.Decode 根据实际内容识别格式。
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // 注册 WebP 解码器，输出仍使用 PNG 编码。

	"github.com/NianBroken/LeafDoc/internal/core/branding"
	"github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// MaxUploadSize 限制上传源文件和最终 PNG 文件的字节数，不限制解码后的像素总数。
const MaxUploadSize = 5 << 20 // 五 MiB，按二进制字节单位计算。

// TargetSize 是输出头像的宽和高，输入先裁正方形，再统一缩放到此像素尺寸。
const TargetSize = 256

// allowedExts 保存允许的输入扩展名，小写且带点，与输出固定 PNG 的规则分别处理。
var allowedExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// AllowedExts 返回排序后的独立扩展名切片，供配置响应和错误提示复用。
// 排序消除 map 遍历顺序差异，修改返回切片不会修改本包允许表。
func AllowedExts() []string {
	exts := make([]string, 0, len(allowedExts))
	for ext := range allowedExts {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

// allowedExtsAsString 用逗号和空格连接允许列表，供错误消息填入具体格式范围。
func allowedExtsAsString() string {
	return strings.Join(AllowedExts(), ", ")
}

// AvatarService 保存头像根目录，具体路径由用户 ID 和固定 PNG 扩展名组成。
type AvatarService struct {
	avatarsDir string
}

// NewAvatarService 创建 avatars 目录，目录已存在时沿用，不读取或迁移已有头像。
// 创建失败附带上下文返回，不返回部分初始化的服务。
func NewAvatarService(storageDir string) (*AvatarService, error) {
	avatarsDir := filepath.Join(storageDir, "avatars")
	if err := os.MkdirAll(avatarsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create avatars directory: %w", err)
	}
	return &AvatarService{avatarsDir: avatarsDir}, nil
}

// UploadAvatar 依次校验用户路径和文件扩展名、限量读取、解码、裁剪、缩放、编码与写盘。
// 扩展名与实际解码格式分别判断，不要求二者匹配，也不在本方法关闭传入文件。
// 只有最终临时文件提交成功才返回 nil，同一用户先前的 PNG 由替换操作覆盖。
func (s *AvatarService) UploadAvatar(userID string, file multipart.File, filename string) error {
	if branding.ContainsUnsafePath(userID) {
		return sharederrors.NewLocalizedError(
			"avatar_invalid_type",
			"Invalid avatar upload",
			"invalid user id for avatar upload",
			nil,
		)
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !allowedExts[ext] {
		return sharederrors.NewLocalizedError(
			"avatar_invalid_type",
			"Invalid avatar file type",
			"invalid avatar file type %s (allowed: %s)",
			nil,
			ext,
			allowedExtsAsString(),
		)
	}

	// 在服务入口独立限制读入字节，不依赖 HTTP 层是否提前限量。
	// 复制错误统一包装成文件过大提示，底层原因仍保留，包含实际读取失败。
	var buf bytes.Buffer
	if err := shared.CopyWithLimit(&buf, file, MaxUploadSize); err != nil {
		return sharederrors.NewLocalizedError(
			"avatar_upload_failed",
			"Avatar file too large",
			"avatar file too large",
			err,
		)
	}

	// 解码器根据字节识别格式，返回的格式名不继续参与校验。
	img, _, err := image.Decode(&buf)
	if err != nil {
		return sharederrors.NewLocalizedError(
			"avatar_decode_failed",
			"Failed to decode avatar image",
			"failed to decode avatar image",
			err,
		)
	}

	// 裁剪决定取景范围，缩放决定输出像素尺寸，两步按此顺序执行。
	cropped := cropToSquare(img)
	resized := resize(cropped, TargetSize, TargetSize)

	var out bytes.Buffer
	if err := png.Encode(&out, resized); err != nil {
		return sharederrors.NewLocalizedError(
			"avatar_upload_failed",
			"Failed to encode avatar image",
			"failed to encode avatar image",
			err,
		)
	}

	targetPath := s.avatarPath(userID)
	// 写入前再次保证目录存在，支持服务构造后目录曾被外部移除的状态。
	err = os.MkdirAll(s.avatarsDir, 0o755)
	if err == nil {
		err = shared.WriteStreamAtomic(targetPath, &out, MaxUploadSize, 0o644)
	}
	if err != nil {
		return sharederrors.NewLocalizedError(
			"avatar_upload_failed",
			"Failed to save avatar file",
			"failed to save avatar file",
			err,
		)
	}

	return nil
}

// DeleteAvatar 先校验路径片段，再删除该用户固定头像路径。
// 目标不存在视为成功，其他删除错误包装后返回，不清理父目录或其他用户文件。
func (s *AvatarService) DeleteAvatar(userID string) error {
	if branding.ContainsUnsafePath(userID) {
		return sharederrors.NewLocalizedError(
			"avatar_delete_failed",
			"Failed to delete avatar",
			"invalid user id for avatar delete",
			nil,
		)
	}

	if err := os.Remove(s.avatarPath(userID)); err != nil && !os.IsNotExist(err) {
		return sharederrors.NewLocalizedError(
			"avatar_delete_failed",
			"Failed to delete avatar",
			"failed to delete avatar",
			err,
		)
	}
	return nil
}

// AvatarPath 返回可通过 os.Stat 查询的头像路径，任何查询错误均映射为空路径和 false。
// 当前只检查 Stat 是否成功，不额外要求目标是普通文件或大小大于零。
func (s *AvatarService) AvatarPath(userID string) (string, bool) {
	if branding.ContainsUnsafePath(userID) {
		return "", false
	}
	path := s.avatarPath(userID)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// avatarPath 拼接目录、用户 ID 与 .png，不单独校验 ID，外部入口需先作安全检查。
func (s *AvatarService) avatarPath(userID string) string {
	return filepath.Join(s.avatarsDir, userID+".png")
}

// cropToSquare 以较短边为边长，从原图中心取最大正方形，考虑 Bounds 的非零起点。
// 多出的奇数像素按整数除法分配，不对图像内容作人脸或主体识别。
func cropToSquare(img image.Image) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	side := w
	if h < side {
		side = h
	}

	offsetX := bounds.Min.X + (w-side)/2
	offsetY := bounds.Min.Y + (h-side)/2
	cropRect := image.Rect(offsetX, offsetY, offsetX+side, offsetY+side)

	// 图像支持 SubImage 时直接返回子图，结果与原图共享像素存储。
	type subImager interface {
		SubImage(r image.Rectangle) image.Image
	}
	if si, ok := img.(subImager); ok {
		return si.SubImage(cropRect)
	}

	// 不支持子图接口时创建独立 RGBA，将选中区域从原图对应起点复制进去。
	dst := image.NewRGBA(image.Rect(0, 0, side, side))
	draw.Draw(dst, dst.Bounds(), img, cropRect.Min, draw.Src)
	return dst
}

// resize 创建目标大小的 RGBA 图像，用 CatmullRom 对完整源范围进行缩放绘制。
// 不保持输入原有宽高比，调用方先裁为正方形以配合头像的正方形目标。
func resize(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}
