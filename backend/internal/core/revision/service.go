// 本文件装配历史服务和单请求执行器视图，具体捕获、读取、恢复分别放在同包文件中。
package revision

import (
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

const (
	errContentHashMismatch       = "content hash mismatch: computed=%s saved=%s"
	errAssetManifestHashMismatch = "asset manifest hash mismatch: computed=%s saved=%s"
	errFailedToRestorePage       = "Failed to restore page"
	errFailedToRestorePageFmt    = "failed to restore page %s"
)

// Service 组合页面树、历史存储和保存策略，资源缓存与页面锁属于服务实例。
type Service struct {
	storageDir         string
	pages              *tree.TreeService
	store              *MySQLStore
	maxRevisions       int // 非正数不执行数量裁剪。
	coalesceWindow     time.Duration
	log                *slog.Logger
	assetManifestCache sync.Map // 页面 ID 对应 assetManifestEntry，缓存资源清单相关状态。
	pageLocks          sync.Map // 页面 ID 对应互斥锁，用于同实例的单页操作互斥。
}

// ServiceOptions 配置单页历史数量与连续保存合并窗口，不在构造时主动清理记录。
type ServiceOptions struct {
	MaxRevisions   int           // 每页保留数量，非正数不执行保存后的裁剪。
	CoalesceWindow time.Duration // 相邻保存合并使用的时间窗口，由记录写入流程判断。
}

// NewService 保存配置并创建历史存储包装，未提供日志器时使用默认日志器。
// 可变选项仅读取首项，构造不打开数据库连接或扫描资源目录。
func NewService(storageDir string, pages *tree.TreeService, logger *slog.Logger, opts ...ServiceOptions) *Service {
	if logger == nil {
		logger = slog.Default()
	}

	var maxRevisions int
	var coalesceWindow time.Duration
	if len(opts) > 0 {
		maxRevisions = opts[0].MaxRevisions
		coalesceWindow = opts[0].CoalesceWindow
	}

	store := NewMySQLStore(storageDir, logger)
	return &Service{
		storageDir:     storageDir,
		pages:          pages,
		store:          store,
		maxRevisions:   maxRevisions,
		coalesceWindow: coalesceWindow,
		log:            logger.With("component", "RevisionService"),
	}
}

// WithExecutor 使用相同配置创建独立缓存和锁映射，并绑定调用方执行器及页面树。
// 原服务为 nil 时返回 nil，两个服务实例的页面锁不互相共享。
func (s *Service) WithExecutor(executor database.Executor, pages *tree.TreeService) *Service {
	if s == nil {
		return nil
	}
	bound := NewService(s.storageDir, pages, s.log, ServiceOptions{MaxRevisions: s.maxRevisions, CoalesceWindow: s.coalesceWindow})
	bound.store.executor = executor
	return bound
}
