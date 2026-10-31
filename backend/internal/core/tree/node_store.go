// 本文件将页面正文、元信息和父节点标识保存到数据库记录，并重建内存页面树。
package tree

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/ignore"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
)

// nodeNamespace 将页面记录与同一数据库目录下的其他模块记录分开。
const nodeNamespace = "page_nodes"

// 写操作共用参数错误文字，具体错误在各入口按缺失参数返回。
const errEntryRequired = "a page is required"
const errParentEntryRequired = "a parent is required"

// NodeStore 通过记录存储接口读写页面，executor 非空时沿用调用方提供的执行器。
// storageDir 标识存储所属目录，ignoreCache 由设置入口保存供模块协作使用。
type NodeStore struct {
	executor    database.Executor
	storageDir  string
	log         *slog.Logger
	ignoreCache *ignore.Cache
}

// nodeRecord 将节点字段、父标识和完整 Markdown 放在同一条 JSON 记录中。
// 写入时去除节点中的父指针和子列表，避免树结构被重复或循环编码。
type nodeRecord struct {
	Node     PageNode `json:"node"`
	ParentID string   `json:"parentId"`
	Raw      string   `json:"raw"`
}

// NewNodeStore 仅保存目录和日志器，不立即连接数据库或加载节点。
func NewNodeStore(storageDir string) *NodeStore {
	return &NodeStore{storageDir: storageDir, log: slog.Default().With("component", "NodeStore")}
}

// SetIgnoreCache 保存忽略缓存引用，不在此触发页面读取或过滤。
func (s *NodeStore) SetIgnoreCache(cache *ignore.Cache) { s.ignoreCache = cache }

// formatMetadataTime 将有效时间转为 UTC 文本，零时间用空字符串表示以省略元信息。
func formatMetadataTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// syncManagedFrontmatter 用节点当前状态覆盖 Markdown 对象中的系统管理字段。
// 自定义字段和正文不在此方法中替换，调用者随后负责组装与保存。
func (s *NodeStore) syncManagedFrontmatter(md *markdown.MarkdownFile, n *PageNode) {
	md.SetLeafDocFrontmatter(n.ID, n.Title)
	md.SetLeafDocMetadata(formatMetadataTime(n.Metadata.CreatedAt), formatMetadataTime(n.Metadata.UpdatedAt), n.Metadata.CreatorID, n.Metadata.LastAuthorID)
	md.SetLeafDocPinned(n.Pinned)
}

// readRecord 按页面标识读取并解码记录，数据库无行转换为业务层页面不存在错误。
// 其他读取错误和 JSON 错误原样返回，不构造空页面作为回退。
func (s *NodeStore) readRecord(id string) (*nodeRecord, error) {
	raw, err := s.records().Read(nodeNamespace, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPageNotFound
	}
	if err != nil {
		return nil, err
	}
	var record nodeRecord
	if err = json.Unmarshal(raw, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// nodePayload 复制节点并移除树关系后编码，父标识由当前 Parent 指针推导。
// 没有父指针时记录 root，原内存节点的关系字段保持原值。
func nodePayload(n *PageNode, raw string) ([]byte, error) {
	copy := *n
	copy.Children = nil
	copy.Parent = nil
	parent := "root"
	if n.Parent != nil {
		parent = n.Parent.ID
	}
	return json.Marshal(nodeRecord{Node: copy, ParentID: parent, Raw: raw})
}

// writeRecord 将节点与全文一起编码，并按节点标识写入页面记录命名空间。
func (s *NodeStore) writeRecord(n *PageNode, raw string) error {
	payload, err := nodePayload(n, raw)
	if err != nil {
		return err
	}
	return s.records().Write(nodeNamespace, n.ID, payload)
}

// readMarkdown 从数据库全文创建 Markdown 对象，路径段加 .md 仅作为对象文件名。
// 此处没有读取同名磁盘文件，解析失败向上传递。
func (s *NodeStore) readMarkdown(n *PageNode) (*markdown.MarkdownFile, error) {
	record, err := s.readRecord(n.ID)
	if err != nil {
		return nil, err
	}
	return markdown.NewMarkdownFileFromRaw(n.Slug+".md", record.Raw)
}

// writeMarkdown 同步系统字段，组装元信息和正文后保存整条页面记录。
// 组装失败不会执行数据库写入，已有 Markdown 对象的系统字段已在内存中改变。
func (s *NodeStore) writeMarkdown(n *PageNode, md *markdown.MarkdownFile) error {
	s.syncManagedFrontmatter(md, n)
	raw, err := markdown.BuildMarkdownWithFrontmatter(md.GetFrontmatter(), md.GetContent())
	if err != nil {
		return err
	}
	return s.writeRecord(n, raw)
}

// ReadPageRaw 返回数据库保存的完整 Markdown，包括页首元信息。
// 空节点与不存在记录均返回页面不存在，不通过页面路径读取文件系统。
func (s *NodeStore) ReadPageRaw(n *PageNode) (string, error) {
	if n == nil {
		return "", ErrPageNotFound
	}
	r, e := s.readRecord(n.ID)
	if e != nil {
		return "", e
	}
	return r.Raw, nil
}

// ReadPageAndRaw 同时提供剥离元信息后的正文与原始全文，避免调用方重复读取。
// 读取失败返回两个空字符串，解析失败仍返回已读取的全文供调用方识别现场。
func (s *NodeStore) ReadPageAndRaw(n *PageNode) (string, string, error) {
	raw, err := s.ReadPageRaw(n)
	if err != nil {
		return "", "", err
	}
	md, err := markdown.NewMarkdownFileFromRaw(n.Slug+".md", raw)
	if err != nil {
		return "", raw, err
	}
	return md.GetContent(), raw, nil
}

// ReadPageContent 复用一次读取和解析流程，只向调用方交付正文和错误。
func (s *NodeStore) ReadPageContent(n *PageNode) (string, error) {
	body, _, err := s.ReadPageAndRaw(n)
	return body, err
}

// ReloadFromDatabase 使用后台上下文重建页面树，数据来源仅为数据库页面记录。
func (s *NodeStore) ReloadFromDatabase() (*PageNode, error) {
	return s.ReloadFromDatabaseContext(context.Background())
}

// ReloadFromDatabaseContext 先加载全部节点，再连接父子关系并从虚拟根节点校验排序。
// 上下文在读取前和解码每项前检查，数据库 List 本身不接收该上下文。
// 缺失父节点、非分区父节点、重复同级路径段或不可达节点都会拒绝整棵树。
func (s *NodeStore) ReloadFromDatabaseContext(ctx context.Context) (*PageNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	records, err := s.records().List(nodeNamespace)
	if err != nil {
		return nil, err
	}
	// 根节点由固定字段在内存中建立，数据库中键为 root 的记录不替换该对象。
	root := &PageNode{ID: "root", Title: "root", Slug: "root", Kind: NodeKindSection, Children: []*PageNode{}}
	nodes := map[string]*PageNode{"root": root}
	parents := map[string]string{}
	for id, raw := range records {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var r nodeRecord
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, fmt.Errorf("decode page %s: %w", id, err)
		}
		if id == "root" {
			continue
		}
		n := r.Node
		n.Children = []*PageNode{}
		nodes[id] = &n
		parents[id] = r.ParentID
	}
	// 所有节点先进入映射，连接时不依赖数据库返回记录的遍历顺序。
	for id, parentID := range parents {
		n := nodes[id]
		p := nodes[parentID]
		if p == nil {
			return nil, fmt.Errorf("page %s has missing parent %s", id, parentID)
		}
		if p.Kind != NodeKindSection {
			return nil, fmt.Errorf("page %s has non-section parent", id)
		}
		n.Parent = p
		p.Children = append(p.Children, n)
	}
	// 遍历记录访问标识，并按 Position 排序，同位置使用标识文字确定次序。
	visited := map[string]bool{}
	var sortTree func(*PageNode) error
	sortTree = func(n *PageNode) error {
		if visited[n.ID] {
			return fmt.Errorf("cyclic page hierarchy at %s", n.ID)
		}
		visited[n.ID] = true
		sort.SliceStable(n.Children, func(i, j int) bool {
			if n.Children[i].Position == n.Children[j].Position {
				return n.Children[i].ID < n.Children[j].ID
			}
			return n.Children[i].Position < n.Children[j].Position
		})
		// 此处同级路径段按字符串精确比较，不执行不区分大小写的归一化。
		slugs := map[string]bool{}
		for _, c := range n.Children {
			if slugs[c.Slug] {
				return ErrPageAlreadyExists
			}
			slugs[c.Slug] = true
			if err := sortTree(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := sortTree(root); err != nil {
		return nil, err
	}
	// 不与根相连的闭环或孤立部分不会被递归访问，在这里统一拒绝。
	if len(visited) != len(nodes) {
		return nil, fmt.Errorf("page hierarchy contains unreachable nodes")
	}
	return root, nil
}

// records 将目录和可选执行器传给共用记录存储，不在此保留连接或结果集。
func (s *NodeStore) records() database.Records {
	return database.Records{Dir: s.storageDir, Executor: s.executor}
}
