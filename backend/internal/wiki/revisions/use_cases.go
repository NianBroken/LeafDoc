package revisions

import (
	"context"
	"log/slog"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
	"github.com/NianBroken/LeafDoc/internal/wiki/pagesave"
)

// RevisionResponse 将历史元数据转换为 HTTP 字段，时间为文本，作者标签按当前缓存解析。
// 内容及资源以摘要标识，正文与资源清单另放在快照响应中。
type RevisionResponse struct {
	ID                string              `json:"id"`
	PageID            string              `json:"pageId"`
	ParentID          string              `json:"parentId,omitempty"`
	Type              string              `json:"type"`
	AuthorID          string              `json:"authorId"`
	Author            *coreauth.UserLabel `json:"author,omitempty"`
	CreatedAt         string              `json:"createdAt"`
	Title             string              `json:"title"`
	Slug              string              `json:"slug"`
	Kind              string              `json:"kind"`
	Path              string              `json:"path"`
	ContentHash       string              `json:"contentHash"`
	AssetManifestHash string              `json:"assetManifestHash"`
	PageCreatedAt     string              `json:"pageCreatedAt,omitempty"`
	PageUpdatedAt     string              `json:"pageUpdatedAt,omitempty"`
	CreatorID         string              `json:"creatorId,omitempty"`
	LastAuthorID      string              `json:"lastAuthorId,omitempty"`
	Summary           string              `json:"summary,omitempty"`
}

// RevisionAssetResponse 提供历史资源名称、内容摘要、字节数与可选类型，不暴露磁盘路径。
type RevisionAssetResponse struct {
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"sizeBytes"`
	MIMEType  string `json:"mimeType,omitempty"`
}

// RevisionSnapshotResponse 将版本信息、历史正文与资源清单组合为一个快照。
type RevisionSnapshotResponse struct {
	Revision *RevisionResponse       `json:"revision"`
	Content  string                  `json:"content"`
	Assets   []RevisionAssetResponse `json:"assets"`
}

// RevisionAssetDeltaResponse 按名称给出服务计算的资源变化状态，不在响应层重算差异。
type RevisionAssetDeltaResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// RevisionComparisonResponse 同时返回两个快照、正文变化标记和资源变化列表。
type RevisionComparisonResponse struct {
	Base           *RevisionSnapshotResponse    `json:"base"`
	Target         *RevisionSnapshotResponse    `json:"target"`
	ContentChanged bool                         `json:"contentChanged"`
	AssetChanges   []RevisionAssetDeltaResponse `json:"assetChanges"`
}

// formatTime 将零时间映射为空文本，其他时间按自身时区格式化为 RFC3339，不强制转为 UTC。
func formatTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.Format(time.RFC3339)
}

// toRevisionResponse 对空版本返回空值，作者解析器缺失或解析失败时仍保留作者标识。
// 其余字段直接复制，时间字段使用同一零值处理规则。
func toRevisionResponse(rev *revision.Revision, userResolver *coreauth.UserResolver) *RevisionResponse {
	if rev == nil {
		return nil
	}
	var author *coreauth.UserLabel
	if userResolver != nil {
		author, _ = userResolver.ResolveUserLabel(rev.AuthorID)
	}
	return &RevisionResponse{
		ID:                rev.ID,
		PageID:            rev.PageID,
		ParentID:          rev.ParentID,
		Type:              string(rev.Type),
		AuthorID:          rev.AuthorID,
		Author:            author,
		CreatedAt:         formatTime(rev.CreatedAt),
		Title:             rev.Title,
		Slug:              rev.Slug,
		Kind:              rev.Kind,
		Path:              rev.Path,
		ContentHash:       rev.ContentHash,
		AssetManifestHash: rev.AssetManifestHash,
		PageCreatedAt:     formatTime(rev.PageCreatedAt),
		PageUpdatedAt:     formatTime(rev.PageUpdatedAt),
		CreatorID:         rev.CreatorID,
		LastAuthorID:      rev.LastAuthorID,
		Summary:           rev.Summary,
	}
}

// toSnapshotResponse 按原清单顺序复制资源字段，空清单使用可编码为空数组的切片。
func toSnapshotResponse(snapshot *revision.RevisionSnapshot, userResolver *coreauth.UserResolver) *RevisionSnapshotResponse {
	if snapshot == nil {
		return nil
	}
	assets := make([]RevisionAssetResponse, 0, len(snapshot.Assets))
	for _, a := range snapshot.Assets {
		assets = append(assets, RevisionAssetResponse{Name: a.Name, SHA256: a.SHA256, SizeBytes: a.SizeBytes, MIMEType: a.MIMEType})
	}
	return &RevisionSnapshotResponse{
		Revision: toRevisionResponse(snapshot.Revision, userResolver),
		Content:  snapshot.Content,
		Assets:   assets,
	}
}

// toComparisonResponse 对两端快照复用同一转换入口，并复制已计算的资源状态与正文标记。
func toComparisonResponse(cmp *revision.RevisionComparison, userResolver *coreauth.UserResolver) *RevisionComparisonResponse {
	if cmp == nil {
		return nil
	}
	changes := make([]RevisionAssetDeltaResponse, 0, len(cmp.AssetChanges))
	for _, c := range cmp.AssetChanges {
		changes = append(changes, RevisionAssetDeltaResponse{Name: c.Name, Status: c.Status})
	}
	return &RevisionComparisonResponse{
		Base:           toSnapshotResponse(cmp.Base, userResolver),
		Target:         toSnapshotResponse(cmp.Target, userResolver),
		ContentChanged: cmp.ContentChanged,
		AssetChanges:   changes,
	}
}

// ListRevisionsInput 提供页面、分页游标及条数，格式约束由 HTTP 层和历史服务处理。
type ListRevisionsInput struct {
	PageID string
	Cursor string
	Limit  int
}

// ListRevisionsOutput 保存当前页版本及继续读取所需游标。
type ListRevisionsOutput struct {
	Revisions  []*revision.Revision
	NextCursor string
}

// ListRevisionsUseCase 将分页列表查询委托给可选历史服务。
type ListRevisionsUseCase struct {
	revision *revision.Service
}

// NewListRevisionsUseCase 保存服务指针，不在构造时读取页面历史。
func NewListRevisionsUseCase(r *revision.Service) *ListRevisionsUseCase {
	return &ListRevisionsUseCase{revision: r}
}

// Execute 在服务为空时返回空列表，否则保留服务的分页结果与错误，不使用请求上下文。
func (uc *ListRevisionsUseCase) Execute(_ context.Context, in ListRevisionsInput) (*ListRevisionsOutput, error) {
	if uc.revision == nil {
		return &ListRevisionsOutput{Revisions: []*revision.Revision{}}, nil
	}
	revs, nextCursor, err := uc.revision.ListRevisionsPage(in.PageID, in.Cursor, in.Limit)
	if err != nil {
		return nil, err
	}
	return &ListRevisionsOutput{Revisions: revs, NextCursor: nextCursor}, nil
}

// GetRevisionInput 以页面和版本标识定位一个快照。
type GetRevisionInput struct {
	PageID     string
	RevisionID string
}

// GetRevisionOutput 的空快照交由路由解释为版本不存在。
type GetRevisionOutput struct {
	Snapshot *revision.RevisionSnapshot
}

// GetRevisionUseCase 只读取历史快照，不恢复当前内容。
type GetRevisionUseCase struct {
	revision *revision.Service
}

// NewGetRevisionUseCase 保存快照服务。
func NewGetRevisionUseCase(r *revision.Service) *GetRevisionUseCase {
	return &GetRevisionUseCase{revision: r}
}

// Execute 对空服务返回空输出，其他读取错误原样返回。
func (uc *GetRevisionUseCase) Execute(_ context.Context, in GetRevisionInput) (*GetRevisionOutput, error) {
	if uc.revision == nil {
		return &GetRevisionOutput{}, nil
	}
	snapshot, err := uc.revision.GetRevisionSnapshot(in.PageID, in.RevisionID)
	if err != nil {
		return nil, err
	}
	return &GetRevisionOutput{Snapshot: snapshot}, nil
}

// CompareRevisionsInput 指定同一页面下的基准与目标版本。
type CompareRevisionsInput struct {
	PageID           string
	BaseRevisionID   string
	TargetRevisionID string
}

// CompareRevisionsOutput 保存服务形成的快照与资源差异。
type CompareRevisionsOutput struct {
	Comparison *revision.RevisionComparison
}

// CompareRevisionsUseCase 复用历史服务的比较规则，不在此层解析正文差异。
type CompareRevisionsUseCase struct {
	revision *revision.Service
}

// NewCompareRevisionsUseCase 保存比较服务入口。
func NewCompareRevisionsUseCase(r *revision.Service) *CompareRevisionsUseCase {
	return &CompareRevisionsUseCase{revision: r}
}

// Execute 对空服务返回空比较输出，调用方负责将空快照映射为读取结果。
func (uc *CompareRevisionsUseCase) Execute(_ context.Context, in CompareRevisionsInput) (*CompareRevisionsOutput, error) {
	if uc.revision == nil {
		return &CompareRevisionsOutput{}, nil
	}
	comparison, err := uc.revision.CompareRevisionSnapshots(in.PageID, in.BaseRevisionID, in.TargetRevisionID)
	if err != nil {
		return nil, err
	}
	return &CompareRevisionsOutput{Comparison: comparison}, nil
}

// GetRevisionAssetInput 在特定页面版本中按资源名定位内容。
type GetRevisionAssetInput struct {
	PageID     string
	RevisionID string
	AssetName  string
}

// GetRevisionAssetOutput 包含资源记录及内部字节文件位置，HTTP 层只用位置发送文件。
type GetRevisionAssetOutput struct {
	Asset *revision.RevisionAssetContent
}

// GetRevisionAssetUseCase 查询资源描述与内容位置，不自行打开文件句柄。
type GetRevisionAssetUseCase struct {
	revision *revision.Service
}

// NewGetRevisionAssetUseCase 保存历史资源读取服务。
func NewGetRevisionAssetUseCase(r *revision.Service) *GetRevisionAssetUseCase {
	return &GetRevisionAssetUseCase{revision: r}
}

// Execute 返回服务定位结果，空服务形成空输出，实际打开和关闭文件由响应处理器负责。
func (uc *GetRevisionAssetUseCase) Execute(_ context.Context, in GetRevisionAssetInput) (*GetRevisionAssetOutput, error) {
	if uc.revision == nil {
		return &GetRevisionAssetOutput{}, nil
	}
	asset, err := uc.revision.GetRevisionAsset(in.PageID, in.RevisionID, in.AssetName)
	if err != nil {
		return nil, err
	}
	return &GetRevisionAssetOutput{Asset: asset}, nil
}

// GetLatestRevisionInput 指定要读取最新版本的页面。
type GetLatestRevisionInput struct {
	PageID string
}

// GetLatestRevisionOutput 的空版本由路由映射为 404。
type GetLatestRevisionOutput struct {
	Revision *revision.Revision
}

// GetLatestRevisionUseCase 只取得服务选定的最新版本，不修改历史游标。
type GetLatestRevisionUseCase struct {
	revision *revision.Service
}

// NewGetLatestRevisionUseCase 保存历史服务。
func NewGetLatestRevisionUseCase(r *revision.Service) *GetLatestRevisionUseCase {
	return &GetLatestRevisionUseCase{revision: r}
}

// Execute 对缺失服务返回空输出，服务存在时直接转交查找结果及错误。
func (uc *GetLatestRevisionUseCase) Execute(_ context.Context, in GetLatestRevisionInput) (*GetLatestRevisionOutput, error) {
	if uc.revision == nil {
		return &GetLatestRevisionOutput{}, nil
	}
	rev, err := uc.revision.GetLatestRevision(in.PageID)
	if err != nil {
		return nil, err
	}
	return &GetLatestRevisionOutput{Revision: rev}, nil
}

// RestoreRevisionInput 提供恢复作者与目标页面、版本，权限在路由层检查。
type RestoreRevisionInput struct {
	UserID     string
	PageID     string
	RevisionID string
}

// RestoreRevisionOutput 返回恢复后重新读取的当前页面。
type RestoreRevisionOutput struct {
	Page *tree.Page
}

// RestoreRevisionUseCase 将历史恢复和页面保存附带处理放入编排器管理的工作单元。
type RestoreRevisionUseCase struct {
	revision     *revision.Service
	tree         *tree.TreeService
	orchestrator *pagesave.PageSaveOrchestrator
	log          *slog.Logger
	metrics      *httpmetrics.HTTPMetrics
}

// NewRestoreRevisionUseCase 保存恢复、树、事务编排与指标依赖，不立即读取版本。
func NewRestoreRevisionUseCase(r *revision.Service, t *tree.TreeService, o *pagesave.PageSaveOrchestrator, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *RestoreRevisionUseCase {
	return &RestoreRevisionUseCase{revision: r, tree: t, orchestrator: o, log: log, metrics: metrics}
}

// Execute 在未绑定工作单元时由编排器取得执行器，复制用例并绑定树及历史服务后递归执行。
// 绑定后的路径恢复版本、读取当前页并提交恢复事件，任何返回错误交由编排器处理事务。
// 保存指标在绑定路径中记录，因此观察该路径返回错误，不等同于外层提交阶段的最终结果。
func (uc *RestoreRevisionUseCase) Execute(ctx context.Context, in RestoreRevisionInput) (out *RestoreRevisionOutput, err error) {
	if !uc.tree.InUnitOfWork() {
		var result *RestoreRevisionOutput
		err := uc.orchestrator.Execute(ctx, uc.tree, func(bound *tree.TreeService, o *pagesave.PageSaveOrchestrator, exec database.Executor) error {
			copy := *uc
			copy.tree = bound
			copy.orchestrator = o
			copy.revision = uc.revision.WithExecutor(exec, bound)
			var inner error
			result, inner = copy.Execute(ctx, in)
			return inner
		})
		return result, err
	}

	// 命名返回值 err 由 defer 读取，记录本路径耗时与结果。
	started := time.Now()
	defer func() {
		uc.metrics.ObservePageSaveWorkflow(string(pagesave.PageOperationRestore), err, started)
	}()

	if uc.revision == nil {
		return nil, sharederrors.NewLocalizedError(
			ErrCodeRevisionServiceUnavailable,
			"Revision service is not available",
			"revision service is not available",
			nil,
		)
	}
	if err := uc.revision.RestoreRevision(in.PageID, in.RevisionID, in.UserID); err != nil {
		return nil, err
	}
	page, err := uc.tree.GetPage(in.PageID)
	if err != nil {
		return nil, err
	}
	if err := uc.orchestrator.Run(pagesave.PageSaveEvent{
		Operation: pagesave.PageOperationRestore,
		UserID:    in.UserID,
		After:     page,
	}); err != nil {
		return nil, err
	}
	return &RestoreRevisionOutput{Page: page}, nil
}

// CheckIntegrityInput 指定要检查历史记录及引用资源完整性的页面。
type CheckIntegrityInput struct {
	PageID string
}

// CheckIntegrityOutput 返回服务发现的问题集合，不把发现问题本身转换为执行错误。
type CheckIntegrityOutput struct {
	Issues []revision.RevisionIntegrityIssue
}

// CheckIntegrityUseCase 为内部调用提供历史完整性检查，当前路由文件未挂载独立接口。
type CheckIntegrityUseCase struct {
	revision *revision.Service
}

// NewCheckIntegrityUseCase 保存历史检查服务。
func NewCheckIntegrityUseCase(r *revision.Service) *CheckIntegrityUseCase {
	return &CheckIntegrityUseCase{revision: r}
}

// Execute 对空服务返回空问题列表，其他情况下保留检查错误或问题结果，不使用请求取消。
func (uc *CheckIntegrityUseCase) Execute(_ context.Context, in CheckIntegrityInput) (*CheckIntegrityOutput, error) {
	if uc.revision == nil {
		return &CheckIntegrityOutput{Issues: []revision.RevisionIntegrityIssue{}}, nil
	}
	issues, err := uc.revision.CheckRevisionIntegrity(in.PageID)
	if err != nil {
		return nil, err
	}
	return &CheckIntegrityOutput{Issues: issues}, nil
}
