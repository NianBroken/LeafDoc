package tree

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/ignore"
)

// TreeService is our main component for handling tree operations
// We use this service to create pages, delete pages, update pages, etc.
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

// SetIgnoreCache sets the ignore cache for multi-level ignore resolution.
func (t *TreeService) SetIgnoreCache(ignoreCache *ignore.Cache) {
	t.store.SetIgnoreCache(ignoreCache)
}

// LoadTree reconstructs the in-memory hierarchy exclusively from MySQL.
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

func (t *TreeService) withLockedTree(fn func() error) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	return fn()
}

func (t *TreeService) withRLockedTree(fn func() error) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return fn()
}

// TreeHash returns the current hash of the tree
func (t *TreeService) TreeHash() string {
	var hash string
	_ = t.withRLockedTree(func() error {
		hash = t.tree.Hash()
		return nil
	})
	return hash
}

// ReloadFromDatabase reloads the hierarchy from MySQL.
// The database read runs without holding the write lock so readers can
// continue serving the current tree concurrently. The lock is acquired
// only for the fast in-memory swap and index rebuild.
func (t *TreeService) ReloadFromDatabase() error {
	return t.ReloadFromDatabaseContext(context.Background())
}

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
