// Package brandassets 集中提供编译进程序的默认站点名称和图片资源。
// 默认图片通过嵌入文件系统读取，不依赖部署目录中另行存在同名文件。
package brandassets

import "embed"

const (
	// 站点名称与文件名供其他模块引用，图片内容由下方 Files 保存。
	SiteName    = "LeafDoc"
	AvatarFile  = "leafdoc-avatar.webp"
	LogoFile    = "leafdoc-logo.webp"
	FaviconFile = "leafdoc-favicon.ico"
)

// Files 保存 files 目录中由嵌入通配符选中的全部资源。
// 文件在嵌入文件系统中的路径包含 files/ 前缀，资源集合以实际目录内容为准。
//
//go:embed files/*
var Files embed.FS

// Read 在名称前拼接 files/，从嵌入文件系统返回对应资源的字节。
// 不存在的文件和不合法的嵌入路径直接返回 ReadFile 的错误。
// 此函数不读取操作系统文件，也不单独限制名称必须是已声明的文件名常量。
func Read(name string) ([]byte, error) { return Files.ReadFile("files/" + name) }
