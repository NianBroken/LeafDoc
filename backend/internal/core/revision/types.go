// Package revision 保存页面历史描述、正文载荷和资源清单，并提供比较与恢复能力。
package revision

import "time"

// RevisionType 标记历史记录对应的操作类别，字符串同时用于持久化和接口响应。
type RevisionType string

const (
	RevisionTypeContentUpdate   RevisionType = "content_update"
	RevisionTypeAssetUpdate     RevisionType = "asset_update"
	RevisionTypeRestore         RevisionType = "restore"
	RevisionTypeStructureUpdate RevisionType = "structure_update"
)

// AssetRef 用文件名、内容摘要和字节数描述一个资源，MIMEType 为空时不输出该 JSON 字段。
type AssetRef struct {
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
	MIMEType  string `json:"mime_type,omitempty"`
}

// RevisionState 是保存流程捕获的内存状态，组合页面结构、正文、额外前置字段与资源清单。
// 各摘要字段用于判断内容是否变化，CapturedAt 表示此次状态捕获时间。
type RevisionState struct {
	PageID               string
	ParentID             string
	Title                string
	Slug                 string
	Kind                 string
	Path                 string
	Content              string
	ContentHash          string
	ExtraFrontmatter     map[string]interface{}
	ExtraFrontmatterHash string
	Assets               []AssetRef
	AssetManifestHash    string
	PageCreatedAt        time.Time
	PageUpdatedAt        time.Time
	CreatorID            string
	LastAuthorID         string
	CapturedAt           time.Time
}

// Revision 是持久化的历史描述，通过哈希引用正文和资源清单，不直接嵌入正文。
// CreatedAt 为历史记录时间，PageCreatedAt 与 PageUpdatedAt 保留捕获时页面本身的时间。
type Revision struct {
	ID                   string                 `json:"id"`
	PageID               string                 `json:"page_id"`
	ParentID             string                 `json:"parent_id,omitempty"`
	Type                 RevisionType           `json:"type"`
	AuthorID             string                 `json:"author_id"`
	CreatedAt            time.Time              `json:"created_at"`
	Title                string                 `json:"title"`
	Slug                 string                 `json:"slug"`
	Kind                 string                 `json:"kind"`
	Path                 string                 `json:"path"`
	ContentHash          string                 `json:"content_hash"`
	ExtraFrontmatter     map[string]interface{} `json:"extra_frontmatter,omitempty"`
	ExtraFrontmatterHash string                 `json:"extra_frontmatter_hash,omitempty"`
	AssetManifestHash    string                 `json:"asset_manifest_hash"`
	PageCreatedAt        time.Time              `json:"page_created_at"`
	PageUpdatedAt        time.Time              `json:"page_updated_at"`
	CreatorID            string                 `json:"creator_id"`
	LastAuthorID         string                 `json:"last_author_id"`
	Summary              string                 `json:"summary,omitempty"`
}

// assetManifest 为资源引用列表提供稳定的 JSON 外层结构。
type assetManifest struct {
	Items []AssetRef `json:"items"`
}

// RevisionIntegrityIssue 描述完整性检查发现的单项异常，Code 供调用方分类处理。
// Path 既能表示文件路径，也能表示 MySQL 记录定位文本，不统一作为本地文件打开。
type RevisionIntegrityIssue struct {
	PageID     string `json:"page_id"`
	RevisionID string `json:"revision_id,omitempty"`
	Code       string `json:"code"`
	Message    string `json:"message"`
	Path       string `json:"path,omitempty"`
}

// RevisionSnapshot 把历史描述与已加载的正文和资源引用组合为预览或比较输入。
type RevisionSnapshot struct {
	Revision *Revision
	Content  string
	Assets   []AssetRef
}

// RevisionAssetContent 返回资源信息和文件位置，本对象不持有已打开的文件句柄。
type RevisionAssetContent struct {
	Asset AssetRef
	Path  string // 资源块的文件系统路径，由调用方打开并负责流式响应的关闭。
}

// RevisionAssetDelta 用文件名和状态表示资源集合差异，不携带资源内容。
type RevisionAssetDelta struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// RevisionComparison 包含两份快照、正文是否不同，以及按名称排列的资源差异。
type RevisionComparison struct {
	Base           *RevisionSnapshot
	Target         *RevisionSnapshot
	ContentChanged bool
	AssetChanges   []RevisionAssetDelta
}
