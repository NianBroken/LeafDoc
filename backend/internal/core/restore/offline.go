// 本文件执行数据库事务与资源目录交换，供离线入口和在线恢复管理器共用。
// 数据库只在资源就位后提交，文件系统回滚失败通过专用错误传递给在线管理器。
package restore

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	coreshared "github.com/NianBroken/LeafDoc/internal/core/shared"
)

// interventionError 包装资源回滚失败，让在线调用方识别后保持维护状态。
// Error 和 Unwrap 直接暴露合并后的底层原因，本类型自身不操作写入屏障。
type interventionError struct{ cause error }

func (e *interventionError) Error() string { return e.cause.Error() }
func (e *interventionError) Unwrap() error { return e.cause }

// rollbackResources 尝试恢复所有已触及目录，回滚成功保留原错误，
// 回滚失败则合并原因并包装为需要介入的错误。
func rollbackResources(sw *swapper, cause error) error {
	if rollbackErr := sw.RollbackAll(); rollbackErr != nil {
		return &interventionError{errors.Join(cause, rollbackErr)}
	}
	return cause
}

// RestoreOffline 使用不限制大小的解压配置验证归档，再取得维护锁执行数据恢复。
// 它不关闭 HTTP 写入准入或重载运行态，调用方应在离线场景中使用。
// 暂存目录始终安排删除，清理错误与恢复错误合并返回。
func RestoreOffline(dataDir, zipPath string) (resultErr error) {
	staging, _, err := extractAndValidateWithLimits(zipPath, dataDir, coreshared.UnrestrictedExtractionLimits)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, os.RemoveAll(staging)) }()
	database.MaintenanceMu.Lock()
	defer database.MaintenanceMu.Unlock()
	return restoreStaged(dataDir, staging)
}

// restoreStaged 读取已校验的数据库载荷，建立导入事务，再交换四类文件资源。
// 交换或事务提交失败均尝试恢复文件资源，函数退出时再回滚尚未结束的事务。
// 数据库提交成功后清理旧资源副本，清理失败由 swapper 记警告而不使提交回退。
func restoreStaged(dataDir, staging string) (resultErr error) {
	raw, err := os.ReadFile(filepath.Join(staging, "database.json"))
	if err != nil {
		return err
	}
	db, err := database.Open(dataDir)
	if err != nil {
		return err
	}
	defer database.CloseResource(db, &resultErr)
	// 导入使用后台上下文，不随触发恢复的 HTTP 请求结束而取消。
	tx, err := database.Import(context.Background(), db, raw)
	if err != nil {
		return err
	}
	defer database.Rollback(tx, &resultErr)
	sw := newSwapper(dataDir, staging)
	if err := sw.SwapAll(); err != nil {
		return rollbackResources(sw, err)
	}
	if err := tx.Commit(); err != nil {
		return rollbackResources(sw, err)
	}
	// 此时数据库已提交，运行态重载由在线调用方在本函数返回后继续执行。
	sw.CommitAll()
	return nil
}
