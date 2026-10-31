package links

import (
	"context"
	"errors"

	corelinks "github.com/NianBroken/LeafDoc/internal/core/links"
	sharederrors "github.com/NianBroken/LeafDoc/internal/core/shared/errors"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// ErrLinkServiceUnavailable 表示没有链接服务实例，区别于运行期间的一次查询错误。
var ErrLinkServiceUnavailable = sharederrors.NewLocalizedError(
	ErrCodeLinkUnavailable,
	"Link service is unavailable",
	"link service is unavailable",
	nil,
)

// GetLinkStatusInput 指定要读取链接状态的单个页面标识。
type GetLinkStatusInput struct {
	PageID string
}

// GetLinkStatusOutput 保存链接服务计算出的状态结果，不在该层复制或过滤字段。
type GetLinkStatusOutput struct {
	Status *corelinks.LinkStatusResult
}

// GetLinkStatusUseCase 先读取页面及其路径，再按标识和路径查询相关链接。
type GetLinkStatusUseCase struct {
	links *corelinks.LinkService
	tree  *tree.TreeService
}

// NewGetLinkStatusUseCase 保存链接与树服务，执行时才检查链接服务是否为空。
func NewGetLinkStatusUseCase(l *corelinks.LinkService, t *tree.TreeService) *GetLinkStatusUseCase {
	return &GetLinkStatusUseCase{links: l, tree: t}
}

// Execute 将页面不存在转换为领域 404 错误，其余页面或链接读取错误原样返回。
// 路径由当前页面计算，不从客户端额外接受；传入上下文不参与底层查询。
func (uc *GetLinkStatusUseCase) Execute(_ context.Context, in GetLinkStatusInput) (*GetLinkStatusOutput, error) {
	if uc.links == nil {
		return nil, ErrLinkServiceUnavailable
	}
	page, err := uc.tree.GetPage(in.PageID)
	if err != nil {
		if errors.Is(err, tree.ErrPageNotFound) {
			return nil, sharederrors.NewLocalizedError(
				ErrCodeLinkPageNotFound,
				"Page not found",
				"page not found",
				err,
			)
		}
		return nil, err
	}
	status, err := uc.links.GetLinkStatusForPage(in.PageID, page.CalculatePath())
	if err != nil {
		return nil, err
	}
	return &GetLinkStatusOutput{Status: status}, nil
}

// GetBrokenLinksOutput 将全站审计结果包装为 links 数组，空结果也保持数组形式。
type GetBrokenLinksOutput struct {
	Links []corelinks.BrokenLink `json:"links"`
}

// GetBrokenLinksUseCase 只查询当前链接索引，不扫描远程 URL 或重新建立索引。
type GetBrokenLinksUseCase struct {
	links *corelinks.LinkService
}

// NewGetBrokenLinksUseCase 保存链接服务，允许缺失并在执行时返回不可用错误。
func NewGetBrokenLinksUseCase(
	l *corelinks.LinkService,
) *GetBrokenLinksUseCase {
	return &GetBrokenLinksUseCase{
		links: l,
	}
}

// Execute 透传查询错误，成功时把 nil 列表归为空切片，不使用请求取消上下文。
func (uc *GetBrokenLinksUseCase) Execute(
	_ context.Context,
) (*GetBrokenLinksOutput, error) {
	if uc.links == nil {
		return nil, ErrLinkServiceUnavailable
	}

	links, err := uc.links.GetBrokenLinks()
	if err != nil {
		return nil, err
	}

	if links == nil {
		links = []corelinks.BrokenLink{}
	}

	return &GetBrokenLinksOutput{
		Links: links,
	}, nil
}
