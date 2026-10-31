package assets

import (
	"context"
	"errors"
	"log/slog"
	"mime/multipart"

	coreassets "github.com/NianBroken/LeafDoc/internal/core/assets"
	"github.com/NianBroken/LeafDoc/internal/core/revision"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// 页面缺失的对外消息与带标识的内部模板供各资源用例共享。
const (
	errPageNotFoundUserMsg = "Page not found"
	errPageNotFoundLogMsg  = "page %s not found"
)

// UploadAssetInput 指定操作者、所属页面、已打开文件及上传限制，文件由调用方负责关闭。
type UploadAssetInput struct {
	UserID   string
	PageID   string
	File     multipart.File
	Filename string
	MaxBytes int64
}

// UploadAssetOutput 只提供保存后的访问地址，不向上层暴露磁盘位置。
type UploadAssetOutput struct {
	URL string
}

// UploadAssetUseCase 先校验所属页面，再保存文件并尝试记录资源历史。
type UploadAssetUseCase struct {
	tree     *tree.TreeService
	asset    *coreassets.AssetService
	revision *revision.Service
	log      *slog.Logger
}

// NewUploadAssetUseCase 保存现有服务，历史服务为空时由公共记录助手跳过历史。
func NewUploadAssetUseCase(t *tree.TreeService, a *coreassets.AssetService, r *revision.Service, log *slog.Logger) *UploadAssetUseCase {
	return &UploadAssetUseCase{tree: t, asset: a, revision: r, log: log}
}

// Execute 将页面缺失转换为资源领域错误，其他查找与保存错误原样返回。
// 文件保存成功后历史失败仅记警告，不撤销文件或改变成功结果；传入上下文未参与处理。
func (uc *UploadAssetUseCase) Execute(_ context.Context, in UploadAssetInput) (*UploadAssetOutput, error) {
	page, err := uc.tree.FindPageByID(in.PageID)
	if err != nil {
		if errors.Is(err, tree.ErrPageNotFound) {
			return nil, sharederrors.NewLocalizedError("asset_page_not_found", errPageNotFoundUserMsg, errPageNotFoundLogMsg, err, in.PageID)
		}
		return nil, err
	}
	url, err := uc.asset.SaveAssetForPage(page, in.File, in.Filename, in.MaxBytes)
	if err != nil {
		return nil, err
	}
	recordAssetRevision(uc.revision, in.PageID, in.UserID, uc.log)
	return &UploadAssetOutput{URL: url}, nil
}

// ListAssetsInput 仅需页面标识，权限由调用用例的路由层检查。
type ListAssetsInput struct {
	PageID string
}

// ListAssetsOutput 保存资源服务返回的地址列表，不重新排序或拼接。
type ListAssetsOutput struct {
	Files []string
}

// ListAssetsUseCase 通过树验证页面，再读取其资源列表，不写入历史。
type ListAssetsUseCase struct {
	tree  *tree.TreeService
	asset *coreassets.AssetService
}

// NewListAssetsUseCase 复用树与资源服务，不建立自己的缓存。
func NewListAssetsUseCase(t *tree.TreeService, a *coreassets.AssetService) *ListAssetsUseCase {
	return &ListAssetsUseCase{tree: t, asset: a}
}

// Execute 区分页面不存在与其他读取错误，成功时直接返回地址列表。
// 此入口不检查传入上下文的取消状态。
func (uc *ListAssetsUseCase) Execute(_ context.Context, in ListAssetsInput) (*ListAssetsOutput, error) {
	page, err := uc.tree.FindPageByID(in.PageID)
	if err != nil {
		if errors.Is(err, tree.ErrPageNotFound) {
			return nil, sharederrors.NewLocalizedError("asset_page_not_found", errPageNotFoundUserMsg, errPageNotFoundLogMsg, err, in.PageID)
		}
		return nil, err
	}
	files, err := uc.asset.ListAssetsForPage(page)
	if err != nil {
		return nil, err
	}
	return &ListAssetsOutput{Files: files}, nil
}

// RenameAssetInput 指定同一页面下的旧名、新名及历史作者，不支持跨页面移动。
type RenameAssetInput struct {
	UserID      string
	PageID      string
	OldFilename string
	NewFilename string
}

// RenameAssetOutput 提供重命名后的资源访问地址。
type RenameAssetOutput struct {
	URL string
}

// RenameAssetUseCase 将名称校验与磁盘操作交给资源服务，成功后尝试记录历史。
type RenameAssetUseCase struct {
	tree     *tree.TreeService
	asset    *coreassets.AssetService
	revision *revision.Service
	log      *slog.Logger
}

// NewRenameAssetUseCase 保存页面、资源及可选历史依赖。
func NewRenameAssetUseCase(t *tree.TreeService, a *coreassets.AssetService, r *revision.Service, log *slog.Logger) *RenameAssetUseCase {
	return &RenameAssetUseCase{tree: t, asset: a, revision: r, log: log}
}

// Execute 确认页面后执行重命名，任一主操作错误立即返回，不执行历史记录。
// 历史记录错误不会回滚重命名，传入上下文在本用例中不使用。
func (uc *RenameAssetUseCase) Execute(_ context.Context, in RenameAssetInput) (*RenameAssetOutput, error) {
	page, err := uc.tree.FindPageByID(in.PageID)
	if err != nil {
		if errors.Is(err, tree.ErrPageNotFound) {
			return nil, sharederrors.NewLocalizedError("asset_page_not_found", errPageNotFoundUserMsg, errPageNotFoundLogMsg, err, in.PageID)
		}
		return nil, err
	}
	newPath, err := uc.asset.RenameAsset(page, in.OldFilename, in.NewFilename)
	if err != nil {
		return nil, err
	}
	recordAssetRevision(uc.revision, in.PageID, in.UserID, uc.log)
	return &RenameAssetOutput{URL: newPath}, nil
}

// DeleteAssetInput 指定所属页面与文件名，用户标识仅用于资源历史作者。
type DeleteAssetInput struct {
	UserID   string
	PageID   string
	Filename string
}

// DeleteAssetUseCase 在页面存在时执行资源删除，并在成功后调用公共历史记录助手。
type DeleteAssetUseCase struct {
	tree     *tree.TreeService
	asset    *coreassets.AssetService
	revision *revision.Service
	log      *slog.Logger
}

// NewDeleteAssetUseCase 保存删除依赖，不提前检查页面或目标文件。
func NewDeleteAssetUseCase(t *tree.TreeService, a *coreassets.AssetService, r *revision.Service, log *slog.Logger) *DeleteAssetUseCase {
	return &DeleteAssetUseCase{tree: t, asset: a, revision: r, log: log}
}

// Execute 将页面缺失映射到资源领域，删除错误直接返回，历史错误只写日志。
// 本入口不使用上下文取消，也不将文件删除和历史写入组合为同一事务。
func (uc *DeleteAssetUseCase) Execute(_ context.Context, in DeleteAssetInput) error {
	page, err := uc.tree.FindPageByID(in.PageID)
	if err != nil {
		if errors.Is(err, tree.ErrPageNotFound) {
			return sharederrors.NewLocalizedError("asset_page_not_found", errPageNotFoundUserMsg, errPageNotFoundLogMsg, err, in.PageID)
		}
		return err
	}
	if err := uc.asset.DeleteAsset(page, in.Filename); err != nil {
		return err
	}
	recordAssetRevision(uc.revision, in.PageID, in.UserID, uc.log)
	return nil
}

// recordAssetRevision 在历史服务存在时记录当前资源状态，失败以警告报告而不返回错误。
// 空原因交给历史服务采用其默认规则，页面及操作者标识原样传递。
func recordAssetRevision(svc *revision.Service, pageID, userID string, log *slog.Logger) {
	if svc == nil {
		return
	}
	if _, _, err := svc.RecordAssetChange(pageID, userID, ""); err != nil {
		log.Warn("failed to record asset revision", "pageID", pageID, "error", err)
	}
}
