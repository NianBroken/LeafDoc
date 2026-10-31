package importer

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/assets"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
)

// ExecutionResult 累计已执行计划项的结果及计数，并携带执行前后的页面树摘要。
// 当前执行分支递增导入数或跳过数，UpdatedCount 不在此执行器中递增。
type ExecutionResult struct {
	ImportedCount  int                   `json:"imported_count"`
	UpdatedCount   int                   `json:"updated_count"`
	SkippedCount   int                   `json:"skipped_count"`
	Items          []ExecutionItemResult `json:"items"`
	TreeHash       string                `json:"tree_hash"`        // 最近一次刷新时的页面树摘要，正常完成时为执行结束状态。
	TreeHashBefore string                `json:"tree_hash_before"` // 首段执行开始前的页面树摘要，恢复执行保留原值。
}

// ExecutionProgress 描述一次进度通知，处理数以计划项位置计数而不是成功页面数。
// 当前源路径只在进入单项时提供，结束通知同时提供结束时间。
type ExecutionProgress struct {
	ProcessedItems        int        `json:"processed_items"`
	TotalItems            int        `json:"total_items"`
	CurrentItemSourcePath *string    `json:"current_item_source_path,omitempty"`
	StartedAt             *time.Time `json:"started_at,omitempty"`
	FinishedAt            *time.Time `json:"finished_at,omitempty"`
}

// ExecutionAction 表示单项最终报告的创建或跳过状态，错误作为独立字段保存。
type ExecutionAction string

const (
	ExecutionActionCreated ExecutionAction = "created"
	ExecutionActionSkipped ExecutionAction = "skipped"
)

// ExecutionItemResult 保留源路径、目标路径和计划备注，失败项以 skipped 携带错误文字。
type ExecutionItemResult struct {
	SourcePath string          `json:"source_path"`
	TargetPath string          `json:"target_path"`
	Action     ExecutionAction `json:"action"`
	Error      *string         `json:"error,omitempty"`
	Notes      []string        `json:"notes,omitempty"`
}

// Executor 按计划顺序建立页面并写入转换后的正文，可在项与项之间检查取消请求。
// 恢复执行使用累计结果和起始下标，不回滚此前已建立的页面或已上传的附件。
type Executor struct {
	plan          *PlanResult
	planOptions   *PlanOptions
	assetMaxBytes int64
	wiki          ImporterWiki
	logger        *slog.Logger
	progressFn    func(ExecutionProgress, *ExecutionResult)
	cancelFn      func() bool
	startIndex    int
	initialResult *ExecutionResult
}

// NewExecutor 绑定计划和 Wiki 服务，附件上限为非正数时采用附件模块默认值。
// 传入的日志对象用于建立导入专用日志，调用方须提供可用对象。
func NewExecutor(plan *PlanResult, planOptions *PlanOptions, assetMaxBytes int64, wiki ImporterWiki, logger *slog.Logger) *Executor {
	if assetMaxBytes <= 0 {
		assetMaxBytes = assets.DefaultMaxUploadSizeBytes
	}
	return &Executor{
		plan:          plan,
		planOptions:   planOptions,
		assetMaxBytes: assetMaxBytes,
		wiki:          wiki,
		logger:        logger.With("component", "ImporterExecutor"),
	}
}

// WithProgressCallback 设置同步进度回调并返回当前执行器，空回调不接收通知。
func (e *Executor) WithProgressCallback(progressFn func(ExecutionProgress, *ExecutionResult)) *Executor {
	e.progressFn = progressFn
	return e
}

// WithCancelCheck 设置每项开始前执行的取消检查，不中断正在进行的文件或网络调用。
func (e *Executor) WithCancelCheck(cancelFn func() bool) *Executor {
	e.cancelFn = cancelFn
	return e
}

// WithResumeState 保存恢复下标并复制累计结果，起始下标由调用方负责校验。
func (e *Executor) WithResumeState(startIndex int, initialResult *ExecutionResult) *Executor {
	e.startIndex = startIndex
	e.initialResult = cloneExecutionResult(initialResult)
	return e
}

// buildImportedContent 使用正文和额外元信息重新构造导入内容，不直接复制整个源文件。
// 标准元信息由后续页面服务管理，额外字段的输出规则交给 Markdown 构造函数。
func buildImportedContent(mdFile *markdown.MarkdownFile) (string, error) {
	fm := mdFile.GetFrontmatter()
	return markdown.BuildMarkdownWithExtraFrontmatter(fm.ExtraFields, mdFile.GetContent())
}

// Execute 在页面树摘要匹配后逐项执行计划，正常结束时返回累计结果。
// 恢复下标大于零时与累计结果中的摘要比较，否则与原计划摘要比较。
// 页面建立、正文读取、附件转换和保存的单项失败都记录后继续，不撤销已产生的副作用。
func (e *Executor) Execute(userID string) (*ExecutionResult, error) {
	beforeExecution := e.wiki.TreeHash()
	expectedTreeHash := e.plan.TreeHash
	if e.startIndex > 0 {
		if e.initialResult == nil || e.initialResult.TreeHash == "" {
			return nil, fmt.Errorf("resume state missing tree hash")
		}
		expectedTreeHash = e.initialResult.TreeHash
	}
	if expectedTreeHash != beforeExecution {
		return nil, fmt.Errorf("plan is stale: expected tree_hash %s but got %s", expectedTreeHash, beforeExecution)
	}

	// 恢复执行也重新建立链接索引和附件缓存，累计结果另行复制。
	transformer := newContentTransformer(e.plan, e.planOptions.SourceBasePath, e.assetMaxBytes)
	startedAt := time.Now()

	result := cloneExecutionResult(e.initialResult)
	if result == nil {
		result = &ExecutionResult{
			TreeHashBefore: beforeExecution,
		}
	}
	if result.TreeHashBefore == "" {
		result.TreeHashBefore = beforeExecution
	}

	e.reportProgress(ExecutionProgress{
		ProcessedItems: e.startIndex,
		TotalItems:     len(e.plan.Items),
		StartedAt:      &startedAt,
	}, result)

	// 取消检查位于每项开始前，取消返回已有结果，不发送正常结束通知。
	for index := e.startIndex; index < len(e.plan.Items); index++ {
		if e.cancelFn != nil && e.cancelFn() {
			return result, ErrImportCanceled
		}

		item := e.plan.Items[index]
		currentItemSourcePath := item.SourcePath
		e.reportProgress(ExecutionProgress{
			ProcessedItems:        index,
			TotalItems:            len(e.plan.Items),
			CurrentItemSourcePath: &currentItemSourcePath,
			StartedAt:             &startedAt,
		}, result)

		execItem := ExecutionItemResult{
			SourcePath: item.SourcePath,
			TargetPath: item.TargetPath,
			Notes:      append([]string{}, item.Notes...),
			Error:      nil,
		}

		switch item.Action {
		case PlanActionCreate:
			// 先确保页面及父级路径存在，再准备正文；后续失败不删除已建立的节点。
			page, err := e.wiki.EnsurePath(userID, item.TargetPath, item.Title, &item.Kind)
			if err != nil {
				errMsg := err.Error()
				execItem.Action = ExecutionActionSkipped
				execItem.Error = &errMsg
				result.SkippedCount++
				result.Items = append(result.Items, execItem)
				e.logger.Error("Failed to ensure path", "target_path", item.TargetPath, "error", err)
				continue
			}
			// 接口未返回页面对象也按失败项报告，避免继续访问空页面的字段。
			if page == nil {
				errMsg := "could not create page"
				execItem.Action = ExecutionActionSkipped
				execItem.Error = &errMsg
				result.SkippedCount++
				result.Items = append(result.Items, execItem)
				e.logger.Error("Could not create page", "target_path", item.TargetPath, "error", errMsg)
				continue
			}
			// 执行时重新读取文件内容，规划阶段提取标题的对象不用于正文保存。
			sourceAbs := filepath.Join(e.planOptions.SourceBasePath, filepath.FromSlash(item.SourcePath))
			mdFile, err := markdown.LoadMarkdownFile(sourceAbs)
			if err != nil {
				errMsg := err.Error()
				execItem.Action = ExecutionActionSkipped
				execItem.Error = &errMsg
				result.SkippedCount++
				result.Items = append(result.Items, execItem)
				e.logger.Error("Failed to load source file", "source_path", sourceAbs, "error", err)
				continue
			}
			importedContent, err := buildImportedContent(mdFile)
			if err != nil {
				errMsg := err.Error()
				execItem.Action = ExecutionActionSkipped
				execItem.Error = &errMsg
				result.SkippedCount++
				result.Items = append(result.Items, execItem)
				e.logger.Error("Failed to prepare imported content", "source_path", sourceAbs, "error", err)
				continue
			}
			// 链接转换在正文保存前完成，期间附件上传会先产生实际存储结果。
			importedContent, err = transformer.TransformContent(userID, item.SourcePath, page, importedContent, e.wiki)
			if err != nil {
				errMsg := err.Error()
				execItem.Action = ExecutionActionSkipped
				execItem.Error = &errMsg
				result.SkippedCount++
				result.Items = append(result.Items, execItem)
				e.logger.Error("Failed to transform imported content", "source_path", sourceAbs, "error", err)
				continue
			}
			// 标题、路径段和类型使用 EnsurePath 返回值，仅正文使用转换结果。
			if _, err := e.wiki.UpdatePage(userID, page.ID, page.Title, page.Slug, &importedContent, &page.Kind); err != nil {
				errMsg := err.Error()
				execItem.Action = ExecutionActionSkipped
				execItem.Error = &errMsg
				result.SkippedCount++
				result.Items = append(result.Items, execItem)
				e.logger.Error("Failed to update page content", "page_id", page.ID, "error", err)
				continue
			}
			execItem.Action = ExecutionActionCreated
			result.ImportedCount++
			e.logger.Info("Imported page", "source_path", item.SourcePath, "target_path", item.TargetPath, "page_id", page.ID)
		case PlanActionSkip:
			execItem.Action = ExecutionActionSkipped
			e.logger.Info("Skipped page", "source_path", item.SourcePath, "target_path", item.TargetPath)
			result.SkippedCount++
		default:
			// 声明过但未实现的 update 动作也进入这里，并作为带错误的跳过项报告。
			errMsg := "unknown action"
			execItem.Action = ExecutionActionSkipped
			execItem.Error = &errMsg
			e.logger.Info("Skipped page with unknown action", "source_path", item.SourcePath, "target_path", item.TargetPath)
			result.SkippedCount++
		}

		// 创建分支中的失败路径已经写入结果并 continue，不经过此处的摘要和进度刷新。
		result.Items = append(result.Items, execItem)
		result.TreeHash = e.wiki.TreeHash()
		e.reportProgress(ExecutionProgress{
			ProcessedItems: index + 1,
			TotalItems:     len(e.plan.Items),
			StartedAt:      &startedAt,
		}, result)
	}

	result.TreeHash = e.wiki.TreeHash()
	finishedAt := time.Now()
	e.reportProgress(ExecutionProgress{
		ProcessedItems: len(e.plan.Items),
		TotalItems:     len(e.plan.Items),
		StartedAt:      &startedAt,
		FinishedAt:     &finishedAt,
	}, result)

	return result, nil
}

// reportProgress 在执行调用栈内直接通知进度，传入的是当前累计结果对象。
// 未设置回调时不执行额外操作，此处不复制结果也不创建后台任务。
func (e *Executor) reportProgress(progress ExecutionProgress, result *ExecutionResult) {
	if e.progressFn == nil {
		return
	}
	e.progressFn(progress, result)
}
