package revision

import (
	"log/slog"

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

type Service struct {
	storageDir         string
	pages              *tree.TreeService
	store              *FSStore
	maxRevisions       int // 0 = unlimited
	coalesceWindow     time.Duration
	log                *slog.Logger
	assetManifestCache sync.Map // pageID → assetManifestEntry
	pageLocks          sync.Map // pageID → *sync.Mutex
}

type ServiceOptions struct {
	MaxRevisions   int           // Maximum revisions to keep per page; 0 = unlimited
	CoalesceWindow time.Duration // Window for coalescing rapid successive saves; 0 = disabled
}

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

	store := NewFSStore(storageDir, logger)
	return &Service{
		storageDir:     storageDir,
		pages:          pages,
		store:          store,
		maxRevisions:   maxRevisions,
		coalesceWindow: coalesceWindow,
		log:            logger.With("component", "RevisionService"),
	}
}
