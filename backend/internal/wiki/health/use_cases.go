package health

import (
	"os"

	"github.com/NianBroken/LeafDoc/internal/core/search"
)

// checkResult 保存本次三个检查的文本结果，再转换为对外使用的命名字段。
type checkResult struct {
	mysql   string
	dataDir string
	search  string
}

// HealthUseCase 顺序检查搜索数据库连通性、存储目录类型与搜索索引状态。
// 它不验证登录、文件写权限、所有业务表或 TLS 网关。
type HealthUseCase struct {
	index      *search.MySQLIndex
	status     *search.IndexingStatus
	storageDir string
}

// NewHealthUseCase 保存检查对象，调用方保证搜索服务与状态指针可用。
func NewHealthUseCase(index *search.MySQLIndex, status *search.IndexingStatus, storageDir string) *HealthUseCase {
	return &HealthUseCase{
		index:      index,
		status:     status,
		storageDir: storageDir,
	}
}

// Execute 执行全部检查并返回整体结果，不因第一项失败而跳过其他项。
// 数据库及目录必须正常，搜索只要未明确失败即可通过，索引中状态仍视为健康。
func (uc *HealthUseCase) Execute() (bool, map[string]string) {
	r := checkResult{}

	if err := uc.index.Ping(); err != nil {
		r.mysql = "failed"
	} else {
		r.mysql = "ok"
	}

	// Stat 成功且目标为目录即可，不尝试创建或写入文件。
	if info, err := os.Stat(uc.storageDir); err != nil || !info.IsDir() {
		r.dataDir = "failed"
	} else {
		r.dataDir = "ok"
	}

	// 失败优先于就绪，尚未就绪且未失败时向调用方报告 indexing。
	switch {
	case uc.status.IsFailed():
		r.search = "failed"
	case uc.status.IsReady():
		r.search = "ok"
	default:
		r.search = "indexing"
	}

	checks := map[string]string{
		"mysql":    r.mysql,
		"data_dir": r.dataDir,
		"search":   r.search,
	}

	healthy := r.mysql == "ok" && r.dataDir == "ok" && r.search != "failed"
	return healthy, checks
}
