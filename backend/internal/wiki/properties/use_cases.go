package properties

import (
	"context"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	coreprop "github.com/NianBroken/LeafDoc/internal/core/properties"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/NianBroken/LeafDoc/internal/http/dto"
)

// ErrPropertiesMissingKey 表示键为空或只含空白，对应路由层的 400 响应。
var ErrPropertiesMissingKey = sharederrors.NewLocalizedError(
	ErrCodePropertiesMissingKey,
	"Query parameter 'key' is required",
	"query parameter key is required",
	nil,
)

// ErrPropertiesMissingValue 表示值为空或只含空白，当前查询入口不支持以空值作筛选。
var ErrPropertiesMissingValue = sharederrors.NewLocalizedError(
	ErrCodePropertiesMissingValue,
	"Query parameter 'value' is required",
	"query parameter value is required",
	nil,
)

// GetPropertyKeysInput 指定键名过滤文字与候选数量。
type GetPropertyKeysInput struct {
	Filter string
	Limit  int
}

// GetPropertyKeysOutput 返回键及页面数量，空结果使用空切片而非 nil。
type GetPropertyKeysOutput struct {
	Keys []coreprop.PropertyKeyCount
}

// GetPropertyKeysUseCase 统一处理候选条数与过滤文字，再读取属性服务。
type GetPropertyKeysUseCase struct {
	svc *coreprop.PropertiesService
}

// NewGetPropertyKeysUseCase 保存已有属性服务，不额外缓存候选结果。
func NewGetPropertyKeysUseCase(svc *coreprop.PropertiesService) *GetPropertyKeysUseCase {
	return &GetPropertyKeysUseCase{svc: svc}
}

// Execute 将非正条数设为五十，上限设为二百，过滤文字去边缘空白并转小写。
// 查询错误原样返回，空结果归为空数组；传入上下文不用于查询取消。
func (uc *GetPropertyKeysUseCase) Execute(_ context.Context, in GetPropertyKeysInput) (*GetPropertyKeysOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	keys, err := uc.svc.GetAllPropertyKeys(strings.ToLower(strings.TrimSpace(in.Filter)), limit)
	if err != nil {
		return nil, err
	}
	if keys == nil {
		keys = []coreprop.PropertyKeyCount{}
	}
	return &GetPropertyKeysOutput{Keys: keys}, nil
}

// GetPagesByPropertyInput 提供一个属性键值条件，不表示多条件组合查询。
type GetPagesByPropertyInput struct {
	Key   string
	Value string
}

// GetPagesByPropertyOutput 返回页面及该页全部索引属性和作者信息。
type GetPagesByPropertyOutput struct {
	Pages []*dto.PropertyPage
}

// GetPagesByPropertyUseCase 先取匹配页面标识，再批量读取属性并结合当前树构造结果。
type GetPagesByPropertyUseCase struct {
	svc          *coreprop.PropertiesService
	treeService  *tree.TreeService
	userResolver *auth.UserResolver
}

// NewGetPagesByPropertyUseCase 保存属性、树与作者解析依赖。
func NewGetPagesByPropertyUseCase(svc *coreprop.PropertiesService, treeService *tree.TreeService, userResolver *auth.UserResolver) *GetPagesByPropertyUseCase {
	return &GetPagesByPropertyUseCase{svc: svc, treeService: treeService, userResolver: userResolver}
}

// Execute 先检查键和值是否全空白，但交给服务查询的仍是原始文本。
// 无匹配返回空切片；索引查询失败返回错误，单个树节点读取失败则跳过该页。
// 页面结果保持标识遍历顺序，不额外排序，也不检查传入上下文。
func (uc *GetPagesByPropertyUseCase) Execute(_ context.Context, in GetPagesByPropertyInput) (*GetPagesByPropertyOutput, error) {
	if strings.TrimSpace(in.Key) == "" {
		return nil, ErrPropertiesMissingKey
	}
	if strings.TrimSpace(in.Value) == "" {
		return nil, ErrPropertiesMissingValue
	}

	pageIDs, err := uc.svc.GetPageIDsByProperty(in.Key, in.Value)
	if err != nil {
		return nil, err
	}
	if len(pageIDs) == 0 {
		return &GetPagesByPropertyOutput{Pages: []*dto.PropertyPage{}}, nil
	}

	propsPerPage, err := uc.svc.GetPropertiesForPages(pageIDs)
	if err != nil {
		return nil, err
	}

	pages := make([]*dto.PropertyPage, 0, len(pageIDs))
	for _, id := range pageIDs {
		node, err := uc.treeService.FindPageByID(id)
		if err != nil || node == nil {
			continue
		}
		pages = append(pages, dto.ToPropertyPage(node, propsPerPage[id], uc.userResolver))
	}

	return &GetPagesByPropertyOutput{Pages: pages}, nil
}
