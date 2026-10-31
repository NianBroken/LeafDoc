package apikeys

import (
	"context"
	"errors"
	"strings"
	"time"

	coreauth "github.com/NianBroken/LeafDoc/internal/core/auth"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
)

// ErrAPIKeysDisabled 表示用例没有密钥服务，创建、列出与撤销均先检查该条件。
var ErrAPIKeysDisabled = errors.New("api key management is disabled")

// CreateAPIKeyInput 提供密钥名称、所属用户、权限上限、可选到期时刻与创建者。
type CreateAPIKeyInput struct {
	Name      string
	UserID    string
	Role      string // 空角色交给核心服务采用查看者角色。
	ExpiresAt *time.Time
	CreatedBy string
}

// CreateAPIKeyOutput 同时返回可公开元数据和创建时的明文令牌，后续列表不提供令牌。
type CreateAPIKeyOutput struct {
	Key    *coreauth.APIKey
	Secret string // 本次创建返回的明文令牌，存储中的哈希不能用于还原此值。
}

// CreateAPIKeyUseCase 先汇总输入验证错误，再将创建操作交给核心服务。
type CreateAPIKeyUseCase struct {
	keys *coreauth.APIKeyService
}

// NewCreateAPIKeyUseCase 保存可选服务，空服务在执行时返回功能关闭错误。
func NewCreateAPIKeyUseCase(k *coreauth.APIKeyService) *CreateAPIKeyUseCase {
	return &CreateAPIKeyUseCase{keys: k}
}

// Execute 检查名称与用户非全空白、非空角色合法、到期时刻严格晚于当前时刻。
// 所有字段问题汇总返回，验证通过后仍将原始文本传给服务，不在此处重写输入。
// 用户存在性、密钥生成及持久化由服务完成，传入上下文不用于取消。
func (uc *CreateAPIKeyUseCase) Execute(_ context.Context, in CreateAPIKeyInput) (*CreateAPIKeyOutput, error) {
	if uc.keys == nil {
		return nil, ErrAPIKeysDisabled
	}

	ve := sharederrors.NewValidationErrors()
	if strings.TrimSpace(in.Name) == "" {
		ve.Add("name", "Name must not be empty")
	}
	if strings.TrimSpace(in.UserID) == "" {
		ve.Add("userId", "Owning user is required")
	}
	if in.Role != "" && !coreauth.IsValidRole(in.Role) {
		ve.Add("role", "Invalid role")
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now()) {
		ve.Add("expiresAt", "Expiry date must be in the future")
	}
	if ve.HasErrors() {
		return nil, ve
	}

	key, secret, err := uc.keys.CreateAPIKey(coreauth.CreateAPIKeyParams{
		Name:      in.Name,
		UserID:    in.UserID,
		Role:      in.Role,
		ExpiresAt: in.ExpiresAt,
		CreatedBy: in.CreatedBy,
	})
	if err != nil {
		return nil, err
	}
	return &CreateAPIKeyOutput{Key: key, Secret: secret}, nil
}

// ListAPIKeysOutput 保存服务返回的密钥元数据，HTTP 层另外选择可公开字段。
type ListAPIKeysOutput struct {
	Keys []*coreauth.APIKey
}

// ListAPIKeysUseCase 读取全部密钥列表，不在该层按当前操作者过滤。
type ListAPIKeysUseCase struct {
	keys *coreauth.APIKeyService
}

// NewListAPIKeysUseCase 保存列表查询服务。
func NewListAPIKeysUseCase(k *coreauth.APIKeyService) *ListAPIKeysUseCase {
	return &ListAPIKeysUseCase{keys: k}
}

// Execute 先检查服务是否启用，随后原样返回列表或查询错误，不使用请求上下文。
func (uc *ListAPIKeysUseCase) Execute(_ context.Context) (*ListAPIKeysOutput, error) {
	if uc.keys == nil {
		return nil, ErrAPIKeysDisabled
	}

	keys, err := uc.keys.ListAPIKeys()
	if err != nil {
		return nil, err
	}
	return &ListAPIKeysOutput{Keys: keys}, nil
}

// RevokeAPIKeyInput 指定密钥记录标识，不使用明文令牌执行撤销。
type RevokeAPIKeyInput struct{ ID string }

// RevokeAPIKeyUseCase 将撤销规则交给核心服务，权限由外部路由检查。
type RevokeAPIKeyUseCase struct {
	keys *coreauth.APIKeyService
}

// NewRevokeAPIKeyUseCase 保存撤销所用的共享服务。
func NewRevokeAPIKeyUseCase(k *coreauth.APIKeyService) *RevokeAPIKeyUseCase {
	return &RevokeAPIKeyUseCase{keys: k}
}

// Execute 在服务存在时按标识执行撤销，直接返回核心服务的错误。
func (uc *RevokeAPIKeyUseCase) Execute(_ context.Context, in RevokeAPIKeyInput) error {
	if uc.keys == nil {
		return ErrAPIKeysDisabled
	}
	return uc.keys.RevokeAPIKey(in.ID)
}
