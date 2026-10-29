package health

import (
	"os"

	"github.com/NianBroken/LeafDoc/internal/core/search"
)

type checkResult struct {
	mysql   string
	dataDir string
	search  string
}

type HealthUseCase struct {
	index      *search.MySQLIndex
	status     *search.IndexingStatus
	storageDir string
}

func NewHealthUseCase(index *search.MySQLIndex, status *search.IndexingStatus, storageDir string) *HealthUseCase {
	return &HealthUseCase{
		index:      index,
		status:     status,
		storageDir: storageDir,
	}
}

func (uc *HealthUseCase) Execute() (bool, map[string]string) {
	r := checkResult{}

	if err := uc.index.Ping(); err != nil {
		r.mysql = "failed"
	} else {
		r.mysql = "ok"
	}

	if info, err := os.Stat(uc.storageDir); err != nil || !info.IsDir() {
		r.dataDir = "failed"
	} else {
		r.dataDir = "ok"
	}

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
