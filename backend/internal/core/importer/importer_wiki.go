package importer

import (
	"mime/multipart"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// ImporterWiki 是导入流程读取页面结构、建立页面和上传附件所使用的接口。
// 导入器通过此接口调用实际 Wiki 服务，用户身份、内容和上传限制随调用传入。
type ImporterWiki interface {
	// TreeHash 提供页面树的状态摘要，供导入执行阶段比较结构状态。
	TreeHash() string
	// LookupPagePath 查找目标路径，并返回页面路径的解析结果或错误。
	LookupPagePath(path string) (*tree.PathLookup, error)
	// EnsurePath 确保目标路径存在，页面标题和节点类型随导入计划传入。
	EnsurePath(userID string, targetPath string, title string, kind *tree.NodeKind) (*tree.Page, error)
	// UpdatePage 写入已有页面的标题、路径段、内容和节点类型，返回服务处理结果。
	UpdatePage(userID string, id, title, slug string, content *string, kind *tree.NodeKind) (*tree.Page, error)
	// UploadAsset 将文件流关联到指定页面，并返回附件公开路径；maxBytes 传递大小限制。
	UploadAsset(userID, pageID string, file multipart.File, filename string, maxBytes int64) (string, error)
}
