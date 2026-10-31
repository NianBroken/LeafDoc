// 本文件清理品牌资源通配结果，供上传完成后移除同类的其他文件。
package branding

import (
	"log"
	"os"
	"path/filepath"
)

// removeOtherMatches 展开 glob，跳过与 keepPath 文本完全相同的路径，删除其余匹配项。
// 匹配或删除错误仅记录日志，不通过返回值中止调用方的后续保存。
// 不递归删除目录，也不根据内容或扩展名额外筛选匹配结果。
func removeOtherMatches(glob string, keepPath string) {
	matches, err := filepath.Glob(glob)
	if err != nil {
		log.Printf("failed to glob pattern %s: %v", glob, err)
		return
	}
	for _, p := range matches {
		if p == keepPath {
			continue
		}
		if err := os.Remove(p); err != nil {
			log.Printf("failed to remove old file %s: %v", p, err)
		}
	}
}
