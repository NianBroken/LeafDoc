package avatar

import (
	"context"
	"mime/multipart"

	coreavatar "github.com/NianBroken/LeafDoc/internal/core/avatar"
)

// UploadAvatarInput 提供目标用户与已打开上传文件，权限和文件关闭由调用方负责。
type UploadAvatarInput struct {
	UserID string
	File   multipart.File
	// Filename 是客户端提供的名称，其扩展名交给头像服务验证，不作为最终存储文件名。
	Filename string
}

// UploadAvatarUseCase 通过头像服务检查并保存图片，不单独维护另一套格式规则。
type UploadAvatarUseCase struct {
	avatars *coreavatar.AvatarService
}

// NewUploadAvatarUseCase 保存当前头像服务，不打开额外资源。
func NewUploadAvatarUseCase(a *coreavatar.AvatarService) *UploadAvatarUseCase {
	return &UploadAvatarUseCase{avatars: a}
}

// Execute 原样传递用户、文件和名称，返回服务错误，不使用传入上下文取消处理。
func (uc *UploadAvatarUseCase) Execute(_ context.Context, in UploadAvatarInput) error {
	return uc.avatars.UploadAvatar(in.UserID, in.File, in.Filename)
}

// DeleteAvatarInput 指定需要删除自定义头像的用户。
type DeleteAvatarInput struct {
	UserID string
}

// DeleteAvatarUseCase 将头像删除委托给共享服务，不删除用户账号或内置默认图片。
type DeleteAvatarUseCase struct {
	avatars *coreavatar.AvatarService
}

// NewDeleteAvatarUseCase 保存现有头像服务。
func NewDeleteAvatarUseCase(a *coreavatar.AvatarService) *DeleteAvatarUseCase {
	return &DeleteAvatarUseCase{avatars: a}
}

// Execute 删除指定用户的上传头像，不存在文件时遵循服务的成功空操作规则。
// 请求上下文在本入口中不参与处理。
func (uc *DeleteAvatarUseCase) Execute(_ context.Context, in DeleteAvatarInput) error {
	return uc.avatars.DeleteAvatar(in.UserID)
}
