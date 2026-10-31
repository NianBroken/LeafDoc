package pagesave

import (
	"log/slog"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/links"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	httpmetrics "github.com/NianBroken/LeafDoc/internal/http/metrics"
)

// LinkIndexSideEffect 根据页面正文、旧路径和标题变化维护出链与入链解析状态。
// 具体解析和数据库更新由链接服务执行，处理器按事件组织调用顺序。
type LinkIndexSideEffect struct {
	svc     *links.LinkService
	log     *slog.Logger
	metrics *httpmetrics.HTTPMetrics
}

// NewLinkIndexSideEffect 保存链接服务和指标，空日志器使用默认值。
func NewLinkIndexSideEffect(svc *links.LinkService, log *slog.Logger, metrics *httpmetrics.HTTPMetrics) *LinkIndexSideEffect {
	if log == nil {
		log = slog.Default()
	}
	return &LinkIndexSideEffect{svc: svc, log: log, metrics: metrics}
}

// Name 返回链接处理器的固定名称。
func (e *LinkIndexSideEffect) Name() string { return "links" }

// WithExecutor 复制处理器并绑定链接服务与事务树，保留原处理器。
func (e *LinkIndexSideEffect) WithExecutor(exec database.Executor, boundTree *tree.TreeService) PageSideEffect {
	copy := *e
	if e.svc != nil {
		copy.svc = e.svc.WithExecutor(exec, boundTree)
	}
	return &copy
}

// Apply 把返回错误写入日志和失败指标，不把错误继续返回给普通接口调用者。
func (e *LinkIndexSideEffect) Apply(event PageSaveEvent) {
	if err := e.ApplyTransactional(event); err != nil {
		e.log.Error("link index update failed", "error", err)
		e.metrics.IncPageSaveSideEffectFailure(string(event.Operation), e.Name())
	}
}

// updateAndHeal 先从纯正文重建页面出链，再尝试解析精确指向当前路径的链接。
// 空页面直接成功，出链更新失败时不继续执行路径匹配。
func (e *LinkIndexSideEffect) updateAndHeal(page *tree.Page) error {
	if page == nil {
		return nil
	}
	if err := e.svc.UpdateLinksForPage(page, page.Content); err != nil {
		return err
	}
	return e.svc.HealLinksForExactPath(page)
}

// reconcileTitles 以去空白并转小写的键去重，实际调用保留首次输入的标题文字。
// 空标题跳过，首个标题重匹配错误立即返回。
func (e *LinkIndexSideEffect) reconcileTitles(titles ...string) error {
	seen := map[string]bool{}
	for _, title := range titles {
		key := strings.ToLower(strings.TrimSpace(title))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if err := e.svc.ReconcileTitle(title); err != nil {
			return err
		}
	}
	return nil
}

// markBrokenForOldPath 对非空旧路径执行前缀失效标记，覆盖旧位置下的子路径引用。
func (e *LinkIndexSideEffect) markBrokenForOldPath(path string) error {
	if path == "" {
		return nil
	}
	return e.svc.MarkLinksBrokenForPrefix(path)
}

// ApplyTransactional 按操作顺序执行链接更新，首个错误返回给外层工作单元决定回滚。
// 此方法本身不打开事务，空服务时直接成功。
func (e *LinkIndexSideEffect) ApplyTransactional(event PageSaveEvent) error {
	if e.svc == nil {
		return nil
	}
	switch event.Operation {
	case PageOperationCreate, PageOperationRestore:
		// 创建与恢复先处理出链及精确路径入链，再对当前标题重新匹配。
		if err := e.updateAndHeal(event.After); err != nil {
			return err
		}
		if event.After != nil {
			return e.reconcileTitles(event.After.Title)
		}
	case PageOperationUpdate:
		// 路径段变化先标记旧前缀失效并重建受影响页面，否则只处理 After。
		if event.SlugChanged {
			if err := e.markBrokenForOldPath(event.OldPath); err != nil {
				return err
			}
			for _, page := range event.AffectedPages {
				if err := e.updateAndHeal(page); err != nil {
					return err
				}
			}
		} else if err := e.updateAndHeal(event.After); err != nil {
			return err
		}
		if event.TitleChanged && event.After != nil {
			return e.reconcileTitles(event.OldTitle, event.After.Title)
		}
	case PageOperationMove:
		// 移动使用旧路径清理失效引用，再按移动后路由处理整个受影响列表。
		if err := e.markBrokenForOldPath(event.OldPath); err != nil {
			return err
		}
		for _, page := range event.AffectedPages {
			if err := e.updateAndHeal(page); err != nil {
				return err
			}
		}
	case PageOperationDelete:
		// 先删除出链，随后按列表规模标记入链与旧路径失效，再重匹配涉及的标题。
		for _, page := range event.AffectedPages {
			if err := e.svc.DeleteOutgoingLinksForPage(page.ID); err != nil {
				return err
			}
		}
		if event.Before != nil {
			if len(event.AffectedPages) > 1 {
				if err := e.markBrokenForOldPath(event.OldPath); err != nil {
					return err
				}
				for _, page := range event.AffectedPages {
					if err := e.svc.MarkIncomingLinksBrokenForPage(page.ID); err != nil {
						return err
					}
				}
			} else {
				if err := e.svc.MarkIncomingLinksBrokenForPage(event.Before.ID); err != nil {
					return err
				}
				if event.OldPath != "" {
					if err := e.svc.MarkLinksBrokenForPath(event.OldPath); err != nil {
						return err
					}
				}
			}
		}
		titles := []string{}
		for _, page := range event.AffectedPages {
			titles = append(titles, page.Title)
		}
		return e.reconcileTitles(titles...)
	}
	return nil
}
