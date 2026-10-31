// 本文件连接重建任务状态与组装层回调，并提供状态查询用例。
package wikiresync

import (
	"context"

	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// TriggerResyncUseCase 先占用任务，再调用传入的启动回调。metrics 保存依赖，
// 此处没有读取或记录指标，实际执行过程由组装层管理。
type TriggerResyncUseCase struct {
	job     *ResyncJob
	trigger func()
	metrics *httpmetrics.HTTPMetrics
}

// NewTriggerResyncUseCase 绑定共享任务对象与启动回调，不在构造时执行重建。
func NewTriggerResyncUseCase(job *ResyncJob, trigger func(), metrics *httpmetrics.HTTPMetrics) *TriggerResyncUseCase {
	return &TriggerResyncUseCase{job: job, trigger: trigger, metrics: metrics}
}

// Execute 拒绝运行中的任务。占用成功后同步调用 trigger，回调自行安排后台执行。
// 请求上下文不参与任务取消，回调返回后用例返回成功，任务结束状态由工作过程填写。
func (uc *TriggerResyncUseCase) Execute(_ context.Context) error {
	if !uc.job.Start() {
		return ErrResyncAlreadyRunning
	}
	uc.trigger()
	return nil
}

// GetResyncStatusOutput 包含共享任务在查询时刻的状态副本。
type GetResyncStatusOutput struct {
	Status JobStatus
}

// GetResyncStatusUseCase 读取与触发用例相同的状态对象，不启动处理步骤。
type GetResyncStatusUseCase struct {
	job *ResyncJob
}

// NewGetResyncStatusUseCase 绑定供接口轮询的任务对象。
func NewGetResyncStatusUseCase(job *ResyncJob) *GetResyncStatusUseCase {
	return &GetResyncStatusUseCase{job: job}
}

// Execute 在任务对象内部加读锁取得快照，不使用请求上下文改变任务生命周期。
func (uc *GetResyncStatusUseCase) Execute(_ context.Context) GetResyncStatusOutput {
	return GetResyncStatusOutput{Status: uc.job.Status()}
}
