// 本文件装配页面树服务，并从数据库加载和替换内存层级及查询索引。
package tree

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/ignore"
)

// TreeService 保存内存层级及 ID、标题和子路径索引，具体正文读写交给 NodeStore。
// 读写方法使用服务锁协调树状态，返回的节点指针并不是与树隔离的副本。
type TreeService struct {
	storageDir   string
	tree         *PageNode
	store        *NodeStore
	log          *slog.Logger
	nodesByID    map[string]*PageNode
	nodesByTitle map[string][]*PageNode
	childSlugs   map[string]map[string]*PageNode

	mu sync.RWMutex
}

const (
	errNilTreeReconstructed    = "internal error: tree reconstruction returned nil tree"
	errPersistChildOrderFailed = "could not persist child order: %w"
	errGetPageContentFailed    = "could not get page content: %w"
	errRollbackMovedNodeFailed = "rollback moved node: %w"
)

// NewTreeService 创建未加载树的服务和空索引，不在构造时读取数据库。
func NewTreeService(storageDir string) *TreeService {
	store := NewNodeStore(storageDir)

	return &TreeService{
		storageDir:   storageDir,
		tree:         nil,
		store:        store,
		log:          slog.Default().With("component", "TreeService"),
		nodesByID:    make(map[string]*PageNode),
		nodesByTitle: make(map[string][]*PageNode),
		childSlugs:   make(map[string]map[string]*PageNode),
	}
}

// SetIgnoreCache 把忽略规则缓存交给底层存储，本方法不取得树锁。
func (t *TreeService) SetIgnoreCache(ignoreCache *ignore.Cache) {
	t.store.SetIgnoreCache(ignoreCache)
}

// LoadTree 持写锁从 MySQL 重建层级，成功后替换根节点并重建全部索引。
// 数据库失败或返回空树时保留原内存树。
func (t *TreeService) LoadTree() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	reconstructed, err := t.store.ReloadFromDatabase()
	if err != nil {
		return err
	}
	if reconstructed == nil {
		return errors.New(errNilTreeReconstructed)
	}
	t.tree = reconstructed
	t.rebuildIndexesLocked()
	return nil
}

// withLockedTree 在写锁内执行回调，回调不能再次取得同一服务的锁。
func (t *TreeService) withLockedTree(fn func() error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return fn()
}

// withRLockedTree 在读锁内执行回调，返回回调错误，不额外检查树是否已加载。
func (t *TreeService) withRLockedTree(fn func() error) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return fn()
}

// TreeHash 持读锁计算当前树哈希，调用方须保证树已加载，否则 nil 根不支持 Hash。
func (t *TreeService) TreeHash() string {
	var hash string
	_ = t.withRLockedTree(func() error {
		hash = t.tree.Hash()
		return nil
	})
	return hash
}

// ReloadFromDatabase 使用后台上下文加载层级，实际交换逻辑由带上下文入口执行。
func (t *TreeService) ReloadFromDatabase() error {
	return t.ReloadFromDatabaseContext(context.Background())
}

// ReloadFromDatabaseContext 在树锁外读取数据库，读取期间旧树仍能被其他请求使用。
// 读取前后检查上下文，成功后持写锁替换整棵树并重建索引，不合并期间的内存变更。
// 存储读取失败会记录错误并向上传递，空结果作为内部错误返回。
func (t *TreeService) ReloadFromDatabaseContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	newTree, err := t.store.ReloadFromDatabaseContext(ctx)
	if err != nil {
		t.log.Error("Error loading hierarchy from MySQL", "error", err)
		return err
	}
	if newTree == nil {
		return fmt.Errorf("internal error: ReloadFromDatabase returned nil tree")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.tree = newTree
	t.rebuildIndexesLocked()

	return nil
}
