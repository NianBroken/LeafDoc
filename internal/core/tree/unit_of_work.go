package tree

import (
	"context"
	"database/sql"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// InUnitOfWork reports whether this tree belongs to a request transaction.
func (t *TreeService) InUnitOfWork() bool { return t.store.executor != nil }

// WithUnitOfWork publishes a detached hierarchy only after every database mutation commits.
func (t *TreeService) WithUnitOfWork(ctx context.Context, fn func(*TreeService, database.Executor) error) (resultErr error) {
	if t.InUnitOfWork() {
		return fn(t, t.store.executor)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	db, err := database.Shared(t.storageDir)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer database.Rollback(tx, &resultErr)
	bound := NewTreeService(t.storageDir)
	bound.store.executor = tx
	bound.store.ignoreCache = t.store.ignoreCache
	bound.tree = cloneHierarchy(t.tree, nil)
	bound.rebuildIndexesLocked()
	if err = fn(bound, tx); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	t.tree = bound.tree
	t.rebuildIndexesLocked()
	return nil
}
func cloneHierarchy(n, parent *PageNode) *PageNode {
	if n == nil {
		return nil
	}
	copy := *n
	copy.Parent = parent
	copy.Children = make([]*PageNode, len(n.Children))
	for i, child := range n.Children {
		copy.Children[i] = cloneHierarchy(child, &copy)
	}
	return &copy
}
