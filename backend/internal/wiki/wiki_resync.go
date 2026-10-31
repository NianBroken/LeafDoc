package wiki

import (
	"context"
	"fmt"
	"time"

	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
	wikiresync "github.com/NianBroken/LeafDoc/internal/wiki/resync"
)

// launchReloadWithProgress 在启动后台协程之前登记等待组，结束时保证退出登记。
// 调用方应先通过 resyncJob.Start 取得任务资格，此方法不再次判断已有任务。
func (w *Wiki) launchReloadWithProgress() {
	w.reloadWG.Add(1)
	go func() {
		defer w.reloadWG.Done()
		w.reloadWithProgress(w.shutdownCtx)
	}()
}

// reloadWithProgress 使用 reloadMu 串行执行页面树与派生索引重建，并报告任务阶段。
// 应通过登记等待组的启动入口调用，确保关闭实例时能够等待它完成。
// 此流程不重建账号服务、不搬运资源文件，也不重新建立历史基线。
func (w *Wiki) reloadWithProgress(ctx context.Context) {
	job := w.resyncJob
	started := time.Now()
	var finishErr error

	defer func() {
		w.metrics.ObserveResyncRun(finishErr, started)
	}()

	// 阶段 panic 转为任务错误，锁由独立 defer 释放，外层指标记录最终错误结果。
	defer func() {
		if r := recover(); r != nil {
			finishErr = fmt.Errorf("panic during reload: %v", r)
			w.log.Error("panic during database reload", "panic", r)
			job.Finish(finishErr)
		}
	}()

	w.reloadMu.Lock()
	defer w.reloadMu.Unlock()

	w.log.Info("database reload started (async)")

	finishErr = w.runReloadPhases(ctx, job)
	if finishErr == nil {
		w.log.Info("database reload completed (async)")
	}
	job.Finish(finishErr)
}

// runReloadPhases 按树、链接、标签及属性、搜索的顺序执行并切换任务阶段。
// 树与搜索错误直接返回；链接错误在非取消情况下只记录警告并继续。
// 标签与属性入口自行记录错误，不向此方法返回失败，也不在其内部检查上下文。
func (w *Wiki) runReloadPhases(ctx context.Context, job *wikiresync.ResyncJob) error {
	job.SetPhase(wikiresync.PhaseTree)
	if err := w.tree.ReloadFromDatabaseContext(ctx); err != nil {
		return fmt.Errorf("tree reconstruction failed: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	job.SetPhase(wikiresync.PhaseLinks)
	if err := w.links.IndexAllPagesContext(ctx); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.log.Warn("link re-index failed during reload", "error", err)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	// 标签与属性共用一次页面读取，其间发生的取消由后续搜索入口继续处理。
	job.SetPhase(wikiresync.PhaseTags)
	w.bootstrapTagsAndProperties()

	job.SetPhase(wikiresync.PhaseSearch)
	w.status.Start()
	searchEffect := pagesave.NewSearchIndexSideEffect(w.searchIndex, w.tree, w.log, w.metrics)
	if err := searchEffect.IndexAllPagesContext(ctx); err != nil {
		w.log.Warn("search re-index failed during reload", "error", err)
		w.status.Fail()
		w.status.Finish()
		return fmt.Errorf("search re-index failed: %w", err)
	}
	w.status.Success()
	w.status.Finish()
	return nil
}

// TriggerResyncAsync 仅在共享任务登记成功时启动后台重建，已有任务时记录并返回。
// 信号处理入口复用该方法，使信号和 HTTP 触发共享任务状态。
func (w *Wiki) TriggerResyncAsync() {
	if !w.resyncJob.Start() {
		w.log.Info("resync already running, signal ignored")
		return
	}
	w.launchReloadWithProgress()
}

// ReloadAfterRestore 在调用线程中取得重建锁并刷新树及索引，返回阶段执行错误。
// 此入口不启动后台协程、不登记等待组，也不记录整体同步耗时指标。
// Start 的布尔结果未参与分支，执行结束后仍统一将共享任务标为完成。
func (w *Wiki) ReloadAfterRestore() error {
	w.reloadMu.Lock()
	defer w.reloadMu.Unlock()
	w.resyncJob.Start()
	err := w.runReloadPhases(w.shutdownCtx, w.resyncJob)
	w.resyncJob.Finish(err)
	return err
}
