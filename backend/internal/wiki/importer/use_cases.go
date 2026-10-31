package importer

import (
	"context"
	"errors"
	"io"

	coreimporter "github.com/NianBroken/LeafDoc/internal/core/importer"
	coreshared "github.com/NianBroken/LeafDoc/internal/core/shared"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// CreateImportPlanInput 提供 ZIP 字节读取入口及目标基础路径，读取器所有权由调用方保留。
type CreateImportPlanInput struct {
	File           io.Reader
	TargetBasePath string
}

// CreateImportPlanOutput 返回服务当前保存的计划状态，而非直接采用创建方法的返回值。
type CreateImportPlanOutput struct {
	Plan *coreimporter.CurrentPlanState
}

// CreateImportPlanUseCase 通过导入服务创建计划，再读取其当前状态供页面展示。
type CreateImportPlanUseCase struct {
	svc *coreimporter.ImporterService
}

// NewCreateImportPlanUseCase 保存共享导入服务，不创建独立的计划仓库。
func NewCreateImportPlanUseCase(svc *coreimporter.ImporterService) *CreateImportPlanUseCase {
	return &CreateImportPlanUseCase{svc: svc}
}

// Execute 顺序完成 ZIP 计划创建和状态读取，不使用传入上下文取消这些操作。
// 三类解压限制错误被转换为固定本地化消息并保留底层原因，其他错误原样返回。
func (uc *CreateImportPlanUseCase) Execute(_ context.Context, in CreateImportPlanInput) (*CreateImportPlanOutput, error) {
	if _, err := uc.svc.CreateImportPlanFromZipUpload(in.File, in.TargetBasePath); err != nil {
		// 使用 errors.Is 穿透错误包装识别限制原因，客户端不直接收到内部调用链文本。
		// 路由中的日志助手另行记录原因，结构化错误保留原错误供内部检查。
		if errors.Is(err, coreshared.ErrFileTooLarge) {
			return nil, sharederrors.NewLocalizedError(
				ErrCodeImporterZipEntryTooLarge, "A file inside the uploaded archive is too large",
				"zip entry exceeds the per-file size limit", err,
			)
		}
		if errors.Is(err, coreshared.ErrCumulativeSizeTooLarge) {
			return nil, sharederrors.NewLocalizedError(
				ErrCodeImporterZipExtractedTooLarge, "The uploaded archive is too large once decompressed",
				"decompressed zip contents exceed the total size limit", err,
			)
		}
		if errors.Is(err, coreshared.ErrDecompressionRatioTooHigh) {
			return nil, sharederrors.NewLocalizedError(
				ErrCodeImporterZipRatioTooHigh, "The uploaded archive looks like a decompression bomb",
				"zip entry's decompression ratio exceeds the allowed maximum", err,
			)
		}
		return nil, err
	}
	plan, err := uc.svc.GetCurrentPlan()
	if err != nil {
		return nil, err
	}
	return &CreateImportPlanOutput{Plan: plan}, nil
}

// GetImportPlanOutput 的 Plan 为空表示当前没有导入任务，是正常空闲结果。
type GetImportPlanOutput struct {
	Plan *coreimporter.CurrentPlanState
}

// GetImportPlanUseCase 读取共享服务的当前计划，不启动或恢复执行。
type GetImportPlanUseCase struct {
	svc *coreimporter.ImporterService
}

// NewGetImportPlanUseCase 保存轮询使用的同一服务实例。
func NewGetImportPlanUseCase(svc *coreimporter.ImporterService) *GetImportPlanUseCase {
	return &GetImportPlanUseCase{svc: svc}
}

// Execute 仅将 ErrNoPlan 转为空成功输出，其他状态读取错误继续返回。
func (uc *GetImportPlanUseCase) Execute(_ context.Context) (*GetImportPlanOutput, error) {
	plan, err := uc.svc.GetCurrentPlan()
	if err != nil {
		if errors.Is(err, coreimporter.ErrNoPlan) {
			// 设置页面可以展示空闲状态，不将没有当前计划视为请求失败。
			return &GetImportPlanOutput{}, nil
		}
		return nil, err
	}
	return &GetImportPlanOutput{Plan: plan}, nil
}

// ExecuteImportInput 提供后台写入时记录的用户标识。
type ExecuteImportInput struct {
	UserID string
}

// ExecuteImportOutput 同时返回当前状态和是否由该调用实际启动任务的标记。
type ExecuteImportOutput struct {
	State   *coreimporter.CurrentPlanState
	Started bool
}

// ExecuteImportUseCase 委托服务启动当前计划，业务写入在服务的后台执行路径中进行。
type ExecuteImportUseCase struct {
	svc *coreimporter.ImporterService
}

// NewExecuteImportUseCase 保存负责并发执行控制的共享服务。
func NewExecuteImportUseCase(svc *coreimporter.ImporterService) *ExecuteImportUseCase {
	return &ExecuteImportUseCase{svc: svc}
}

// Execute 将运行冲突、无计划及状态不可用转换为对应领域错误，其他错误原样返回。
// 请求上下文不传入后台执行，HTTP 连接结束不通过本入口取消导入。
func (uc *ExecuteImportUseCase) Execute(_ context.Context, in ExecuteImportInput) (*ExecuteImportOutput, error) {
	state, started, err := uc.svc.StartCurrentPlanExecution(in.UserID)
	if err != nil {
		if errors.Is(err, coreimporter.ErrImportExecutionRunning) {
			return nil, sharederrors.NewLocalizedError("importer_execution_running", "Import is already running", "import is already running", err)
		}
		if errors.Is(err, coreimporter.ErrNoPlan) {
			return nil, sharederrors.NewLocalizedError("importer_no_plan", "No import plan available", "no import plan available", err)
		}
		if errors.Is(err, coreimporter.ErrImportStateUnavailable) {
			return nil, sharederrors.NewLocalizedError("importer_state_unavailable", "Import state is unavailable", "import state is unavailable", err)
		}
		return nil, err
	}
	return &ExecuteImportOutput{State: state, Started: started}, nil
}

// ClearImportPlanUseCase 请求取消当前任务，并在无需等待运行退出时清理计划。
type ClearImportPlanUseCase struct {
	svc *coreimporter.ImporterService
}

// NewClearImportPlanUseCase 复用导入服务的取消与清理规则。
func NewClearImportPlanUseCase(svc *coreimporter.ImporterService) *ClearImportPlanUseCase {
	return &ClearImportPlanUseCase{svc: svc}
}

// Execute 对仍在运行且已经请求取消的状态直接返回，供路由以 202 响应。
// 无计划不阻断后续清理，其他取消错误先返回；清理完成返回空状态。
func (uc *ClearImportPlanUseCase) Execute(_ context.Context) (*coreimporter.CurrentPlanState, error) {
	state, _, err := uc.svc.CancelCurrentPlan()
	if err == nil && state != nil && state.ExecutionStatus == coreimporter.ExecutionStatusRunning && state.CancelRequested {
		return state, nil
	}
	if err != nil && !errors.Is(err, coreimporter.ErrNoPlan) {
		if errors.Is(err, coreimporter.ErrImportStateUnavailable) {
			return nil, sharederrors.NewLocalizedError("importer_state_unavailable", "Import state is unavailable", "import state is unavailable", err)
		}
		return nil, err
	}
	if err := uc.svc.ClearCurrentPlan(); err != nil {
		return nil, err
	}
	return nil, nil
}
