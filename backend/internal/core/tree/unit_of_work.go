// 本文件把页面及派生数据库写入组合到请求事务，成功提交后才发布内存层级。
package tree

import (
	"context"
	"database/sql"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// InUnitOfWork 以存储是否绑定外部执行器判断当前树是否属于工作单元，不验证执行器类型。
func (t *TreeService) InUnitOfWork() bool { return t.store.executor != nil }

// WithUnitOfWork 在原树写锁内开启读已提交事务，并复制层级供回调操作。
// 回调失败、上下文取消或提交失败时不替换原树，延迟回滚错误合并进返回值。
// 已绑定执行器的嵌套调用直接复用当前树和执行器，不再建事务或取原树锁。
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
	// 独立树视图持有复制节点和自己的索引，数据库操作绑定同一个事务。
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
	// 提交成功后一次发布复制树，再为原服务重建查询索引。
	t.tree = bound.tree
	t.rebuildIndexesLocked()
	return nil
}

// cloneHierarchy 复制节点值和全部 Children 切片，并重接副本中的 Parent 引用。
// nil 节点保留为空，结构外的存储、锁及其他服务引用不参与复制。
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
