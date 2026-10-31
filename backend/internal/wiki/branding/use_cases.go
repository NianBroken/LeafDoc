package branding

import (
	"context"
	"mime/multipart"

	corebanding "github.com/NianBroken/LeafDoc/internal/core/branding"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// 品牌读取失败的消息与模板由各个写后读取流程共同使用。
const (
	errFailedToLoadBrandingUserMsg = "Failed to load branding config"
	errFailedToLoadBrandingLogMsg  = "failed to load branding config"
)

// GetBrandingOutput 保存完整品牌响应，包含当前文件名和上传约束。
type GetBrandingOutput struct {
	Config *corebanding.BrandingConfigResponse
}

// GetBrandingUseCase 读取当前配置并将失败包装为稳定的品牌领域错误。
type GetBrandingUseCase struct {
	branding *corebanding.BrandingService
}

// NewGetBrandingUseCase 保存服务，不缓存读取结果。
func NewGetBrandingUseCase(b *corebanding.BrandingService) *GetBrandingUseCase {
	return &GetBrandingUseCase{branding: b}
}

// Execute 读取配置并保留底层错误为原因，不使用请求上下文取消读取。
func (uc *GetBrandingUseCase) Execute(_ context.Context) (*GetBrandingOutput, error) {
	cfg, err := uc.branding.GetBranding()
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			ErrCodeBrandingConfigUnavailable,
			errFailedToLoadBrandingUserMsg,
			errFailedToLoadBrandingLogMsg,
			err,
		)
	}
	return &GetBrandingOutput{Config: cfg}, nil
}

// UpdateBrandingInput 只包含站点名，图片文件由独立上传和删除入口管理。
type UpdateBrandingInput struct {
	SiteName string
}

// UpdateBrandingUseCase 写站点名后再次读取配置，两个调用不组合成此层事务。
type UpdateBrandingUseCase struct {
	branding *corebanding.BrandingService
}

// NewUpdateBrandingUseCase 保存现有品牌服务。
func NewUpdateBrandingUseCase(b *corebanding.BrandingService) *UpdateBrandingUseCase {
	return &UpdateBrandingUseCase{branding: b}
}

// Execute 写入失败直接返回，写后读取失败返回配置不可用错误，不回滚已保存站点名。
func (uc *UpdateBrandingUseCase) Execute(_ context.Context, in UpdateBrandingInput) (*GetBrandingOutput, error) {
	if err := uc.branding.UpdateBranding(in.SiteName); err != nil {
		return nil, err
	}
	cfg, err := uc.branding.GetBranding()
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			ErrCodeBrandingConfigUnavailable,
			errFailedToLoadBrandingUserMsg,
			errFailedToLoadBrandingLogMsg,
			err,
		)
	}
	return &GetBrandingOutput{Config: cfg}, nil
}

// UploadLogoInput 提供已打开标识文件及客户端名称，文件句柄由调用方关闭。
type UploadLogoInput struct {
	File     multipart.File
	Filename string
}

// UploadLogoOutput 同时返回上传地址与写后读取的品牌配置。
type UploadLogoOutput struct {
	Path   string
	Config *corebanding.BrandingConfigResponse
}

// UploadLogoUseCase 将图片约束及文件写入交给品牌服务，随后取得展示配置。
type UploadLogoUseCase struct {
	branding *corebanding.BrandingService
}

// NewUploadLogoUseCase 保存共享品牌服务。
func NewUploadLogoUseCase(b *corebanding.BrandingService) *UploadLogoUseCase {
	return &UploadLogoUseCase{branding: b}
}

// Execute 成功上传后再读配置，读取失败不会撤销文件保存；上下文不参与处理。
func (uc *UploadLogoUseCase) Execute(_ context.Context, in UploadLogoInput) (*UploadLogoOutput, error) {
	path, err := uc.branding.UploadLogo(in.File, in.Filename)
	if err != nil {
		return nil, err
	}
	cfg, err := uc.branding.GetBranding()
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			ErrCodeBrandingConfigUnavailable,
			errFailedToLoadBrandingUserMsg,
			errFailedToLoadBrandingLogMsg,
			err,
		)
	}
	return &UploadLogoOutput{Path: path, Config: cfg}, nil
}

// DeleteLogoUseCase 删除自定义标识后返回服务当前配置。
type DeleteLogoUseCase struct {
	branding *corebanding.BrandingService
}

// NewDeleteLogoUseCase 保存删除与读取共用的服务。
func NewDeleteLogoUseCase(b *corebanding.BrandingService) *DeleteLogoUseCase {
	return &DeleteLogoUseCase{branding: b}
}

// Execute 先删除后读取，第二步失败返回领域错误但不恢复已删除文件。
func (uc *DeleteLogoUseCase) Execute(_ context.Context) (*GetBrandingOutput, error) {
	if err := uc.branding.DeleteLogo(); err != nil {
		return nil, err
	}
	cfg, err := uc.branding.GetBranding()
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			ErrCodeBrandingConfigUnavailable,
			errFailedToLoadBrandingUserMsg,
			errFailedToLoadBrandingLogMsg,
			err,
		)
	}
	return &GetBrandingOutput{Config: cfg}, nil
}

// UploadFaviconInput 提供图标文件和名称，类型检查由品牌服务执行。
type UploadFaviconInput struct {
	File     multipart.File
	Filename string
}

// UploadFaviconOutput 保存上传地址和随后读取的品牌配置。
type UploadFaviconOutput struct {
	Path   string
	Config *corebanding.BrandingConfigResponse
}

// UploadFaviconUseCase 使用图标专属服务规则，不借用标识上传入口。
type UploadFaviconUseCase struct {
	branding *corebanding.BrandingService
}

// NewUploadFaviconUseCase 保存现有品牌服务。
func NewUploadFaviconUseCase(b *corebanding.BrandingService) *UploadFaviconUseCase {
	return &UploadFaviconUseCase{branding: b}
}

// Execute 顺序上传图标和读取配置，读取失败不回滚上传，文件仍由调用方关闭。
func (uc *UploadFaviconUseCase) Execute(_ context.Context, in UploadFaviconInput) (*UploadFaviconOutput, error) {
	path, err := uc.branding.UploadFavicon(in.File, in.Filename)
	if err != nil {
		return nil, err
	}
	cfg, err := uc.branding.GetBranding()
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			ErrCodeBrandingConfigUnavailable,
			errFailedToLoadBrandingUserMsg,
			errFailedToLoadBrandingLogMsg,
			err,
		)
	}
	return &UploadFaviconOutput{Path: path, Config: cfg}, nil
}

// DeleteFaviconUseCase 删除自定义图标并读取当前品牌配置。
type DeleteFaviconUseCase struct {
	branding *corebanding.BrandingService
}

// NewDeleteFaviconUseCase 保存共享品牌服务。
func NewDeleteFaviconUseCase(b *corebanding.BrandingService) *DeleteFaviconUseCase {
	return &DeleteFaviconUseCase{branding: b}
}

// Execute 主删除错误直接返回，删除后的配置读取失败包装为领域错误且不恢复文件。
func (uc *DeleteFaviconUseCase) Execute(_ context.Context) (*GetBrandingOutput, error) {
	if err := uc.branding.DeleteFavicon(); err != nil {
		return nil, err
	}
	cfg, err := uc.branding.GetBranding()
	if err != nil {
		return nil, sharederrors.NewLocalizedError(
			ErrCodeBrandingConfigUnavailable,
			errFailedToLoadBrandingUserMsg,
			errFailedToLoadBrandingLogMsg,
			err,
		)
	}
	return &GetBrandingOutput{Config: cfg}, nil
}
