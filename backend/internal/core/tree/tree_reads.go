// 本文件提供树及页面读取，持读锁期间组合节点引用与数据库正文。
package tree

import (
	"fmt"
	"strings"
	"sync"
)

// GetTree 持读锁取得根节点引用后返回，不深复制，也不让调用方继续持有锁。
func (t *TreeService) GetTree() *PageNode {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.tree
}

// IsLoaded 只检查内存根是否非 nil，不执行数据库连通性检查。
func (t *TreeService) IsLoaded() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tree != nil
}

// HasPages 检查根的直接子节点切片是否非空，章节同样算作存在内容节点。
func (t *TreeService) HasPages() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tree != nil && len(t.tree.Children) > 0
}

// NodeCounts 持读锁遍历 ID 索引，section 计为章节，其余类型均计为页面。
// 索引不收录合成根节点，因此根不计数，此方法不读取正文。
func (t *TreeService) NodeCounts() (pages, sections int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	for _, node := range t.nodesByID {
		if node.Kind == NodeKindSection {
			sections++
			continue
		}
		pages++
	}
	return pages, sections
}

// WalkNodes 先取得非根 ID 的深度优先快照，再在树锁外逐项调用回调。
// 回调可以调用其他树方法，首个错误立即停止；未加载树时没有回调并返回成功。
// 收集后树发生变化时，此轮仍使用原 ID 列表。
func (t *TreeService) WalkNodes(fn func(id string) error) error {
	ids := t.collectIDsDFS()
	for _, id := range ids {
		if err := fn(id); err != nil {
			return err
		}
	}
	return nil
}

// collectIDsDFS 持读锁按当前 Children 次序收集 ID，跳过 ID 精确为 root 的节点。
// 树未加载返回 nil，遍历要求子节点有效且层级无环。
func (t *TreeService) collectIDsDFS() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tree == nil {
		return nil
	}

	var ids []string
	var collect func(*PageNode)
	collect = func(node *PageNode) {
		if node.ID != "root" {
			ids = append(ids, node.ID)
		}
		for _, child := range node.Children {
			collect(child)
		}
	}
	collect(t.tree)
	return ids
}

// GetPages 返回与输入位置对应的页面和错误切片，失败位置的页面保持 nil。
// 整轮读取始终持树读锁，普通服务逐页启动 goroutine，事务视图串行读取正文。
// 节点不存在仅影响对应下标，所有有效任务完成后才返回，节点引用不深复制。
func (t *TreeService) GetPages(ids []string) ([]*Page, []error) {
	pages := make([]*Page, len(ids))
	errs := make([]error, len(ids))
	if len(ids) == 0 {
		return pages, errs
	}

	type task struct {
		index int
		node  *PageNode
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tree == nil {
		for i := range errs {
			errs[i] = ErrTreeNotLoaded
		}
		return pages, errs
	}

	tasks := make([]task, 0, len(ids))
	for i, id := range ids {
		node := t.getNodeByIDLocked(id)
		if node == nil {
			errs[i] = ErrPageNotFound
			continue
		}
		tasks = append(tasks, task{index: i, node: node})
	}

	if len(tasks) == 0 {
		return pages, errs
	}

	// 外部事务使用同一执行器，按顺序消费各页读取结果，不进入并行读取分支。
	if t.InUnitOfWork() {
		for _, tk := range tasks {
			content, raw, err := t.store.ReadPageAndRaw(tk.node)
			if err != nil {
				errs[tk.index] = fmt.Errorf(errGetPageContentFailed, err)
			} else {
				pages[tk.index] = &Page{PageNode: tk.node, Content: content, RawContent: raw}
			}
		}
		return pages, errs
	}

	// 正文读取并发进行，结果写回由局部锁保护，等待全部 goroutine 退出再释放树读锁。
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(tasks))
	for _, tk := range tasks {
		go func(tk task) {
			defer wg.Done()
			content, raw, err := t.store.ReadPageAndRaw(tk.node)
			mu.Lock()
			if err != nil {
				errs[tk.index] = fmt.Errorf(errGetPageContentFailed, err)
			} else {
				pages[tk.index] = &Page{PageNode: tk.node, Content: content, RawContent: raw}
			}
			mu.Unlock()
		}(tk)
	}
	wg.Wait()

	return pages, errs
}

// GetPage 持读锁按 ID 取节点并读取正文和原始内容，读取错误带上下文向上传递。
func (t *TreeService) GetPage(id string) (*Page, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tree == nil {
		return nil, ErrTreeNotLoaded
	}

	// 内存索引负责确认节点存在，正文仍从存储读取，不保存在节点结构中。
	page := t.getNodeByIDLocked(id)
	if page == nil {
		return nil, ErrPageNotFound
	}

	content, raw, err := t.store.ReadPageAndRaw(page)
	if err != nil {
		return nil, fmt.Errorf(errGetPageContentFailed, err)
	}

	return &Page{
		PageNode:   page,
		Content:    content,
		RawContent: raw,
	}, nil
}

// ReadPageRaw 持读锁读取含前置字段的原始 Markdown，树未加载和节点不存在分别报错。
func (t *TreeService) ReadPageRaw(id string) (string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tree == nil {
		return "", ErrTreeNotLoaded
	}

	page := t.getNodeByIDLocked(id)
	if page == nil {
		return "", ErrPageNotFound
	}

	raw, err := t.store.ReadPageRaw(page)
	if err != nil {
		return "", fmt.Errorf("could not get page raw content: %w", err)
	}

	return raw, nil
}

// ResolvePermalinkTarget 将稳定 ID 转成当前 slug 与层级路径，不读取页面正文。
// 结果路径去掉一个开头斜杠，供永久链接入口定位当前路由。
func (t *TreeService) ResolvePermalinkTarget(id string) (*PermalinkTarget, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tree == nil {
		return nil, ErrTreeNotLoaded
	}

	node := t.getNodeByIDLocked(id)
	if node == nil {
		return nil, ErrPageNotFound
	}

	return &PermalinkTarget{
		ID:   node.ID,
		Slug: node.Slug,
		Path: strings.TrimPrefix(node.CalculatePath(), "/"),
	}, nil
}

// FindPageByRoutePath 将输入按斜杠逐段查找，要求每段非空且 slug 大小写精确匹配。
// 不裁去首尾斜杠或空白，找到最终节点后读取正文，不只返回结构节点。
func (t *TreeService) FindPageByRoutePath(routePath string) (*Page, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tree == nil {
		return nil, ErrTreeNotLoaded
	}

	// 含开头、结尾或连续斜杠的输入会产生空段，并在循环中被拒绝。
	routePart := strings.Split(routePath, "/")
	if len(routePart) == 0 {
		return nil, ErrPageNotFound
	}

	parent := t.tree
	var node *PageNode
	for _, part := range routePart {
		if part == "" {
			return nil, ErrPageNotFound
		}

		node = t.findChildBySlugExactInParentLocked(parent, part)
		if node == nil {
			return nil, ErrPageNotFound
		}

		parent = node
	}

	content, raw, err := t.store.ReadPageAndRaw(node)
	if err != nil {
		return nil, fmt.Errorf(errGetPageContentFailed, err)
	}

	return &Page{
		PageNode:   node,
		Content:    content,
		RawContent: raw,
	}, nil
}

// LookupPagePath 持读锁查询各路径段存在状态和创建合法性，不读取或创建正文。
func (t *TreeService) LookupPagePath(p string) (*PathLookup, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.lookupPagePathLocked(p)
}

// lookupPagePathLocked 要求调用方已持树锁，裁去路径两端空白和斜杠后逐段查询。
// 空路径返回不可创建结果，普通路径使用忽略大小写的子 slug 查找。
func (t *TreeService) lookupPagePathLocked(p string) (*PathLookup, error) {
	if t.tree == nil {
		return nil, ErrTreeNotLoaded
	}

	slugService := NewSlugService()
	path := strings.TrimSpace(p)
	path = strings.Trim(path, "/")
	if path == "" {
		return &PathLookup{
			Path:      path,
			Segments:  []PathSegment{},
			Exists:    false,
			CanCreate: false,
		}, nil
	}

	// 执行一次全局双斜杠替换，不循环压缩所有长度的连续斜杠。
	path = strings.ReplaceAll(path, "//", "/")

	// 预分配与路径段等长的结果，后续逐段写入存在信息。
	pathParts := strings.Split(path, "/")
	if len(pathParts) == 0 {
		return &PathLookup{
			Path:      path,
			Segments:  []PathSegment{},
			Exists:    false,
			CanCreate: false,
		}, nil
	}

	lookup := &PathLookup{
		Path:      path,
		Segments:  make([]PathSegment, len(pathParts)),
		Exists:    true,
		CanCreate: true,
	}

	parent := t.tree

	// 空段和点路径段直接报错，不把它们解释为当前目录或父目录。
	for i, part := range pathParts {
		if part == "" || part == "." || part == ".." {
			return nil, fmt.Errorf("invalid path segment: %q", part)
		}

		// 每轮先写入未命中的段状态，再尝试读取当前父节点下的对应子节点。
		segment := PathSegment{
			Slug:   part,
			Exists: false,
		}

		// 当前段的原始文字保留为 Slug，不在结果中替换为规范化的新值。
		lookup.Segments[i] = segment

		// 父节点为 nil 时子查找返回 nil，因此首个缺失段后不会再解析实际节点。
		e := t.findChildBySlugInParentLocked(parent, part)
		if e != nil {
			// 存在段的可选字段指向实际节点字段，并非独立字符串副本。
			lookup.Segments[i].Exists = true
			lookup.Segments[i].ID = &e.ID
			lookup.Segments[i].Kind = &e.Kind
			lookup.Segments[i].Title = &e.Title

			// 下一个路径片段以当前命中节点作为父级。
			parent = e
		}

		// 缺失段需要通过 slug 校验，CanCreate 失败后不再恢复为 true。
		if !lookup.Segments[i].Exists {
			if lookup.CanCreate && slugService.IsValidSlug(part) != nil {
				lookup.CanCreate = false
			}

			// 预填后续缺失段并检查其创建合法性，外层循环仍会继续逐段处理。
			for j := i + 1; j < len(pathParts); j++ {
				if lookup.CanCreate && slugService.IsValidSlug(pathParts[j]) != nil {
					lookup.CanCreate = false
				}
				lookup.Segments[j] = PathSegment{
					Slug:   pathParts[j],
					Exists: false,
				}
			}

			lookup.Exists = false

			parent = nil
		}
	}

	return lookup, nil
}
