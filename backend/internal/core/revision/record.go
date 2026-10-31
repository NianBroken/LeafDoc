// 本文件捕获页面状态并记录正文、结构与资源变更，按页面锁串行化同实例的写入。
// 正文和额外前置字段用于正文保存去重，资源变更使用独立的完整扫描路径。
package revision

import (
	"encoding/json"

	"fmt"

	"runtime"

	"strings"
	"sync"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/markdown"
	"github.com/NianBroken/LeafDoc/internal/core/shared"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// CapturePageState 读取页面和原始前置字段，并逐个扫描当前资源生成完整内存状态。
// 捕获本身不写历史描述或资源块，也不在入口取得页面写锁。
func (s *Service) CapturePageState(pageID string) (*RevisionState, error) {
	return s.capturePageState(pageID, true)
}

// RecordContentUpdate 先读取页面，再按正文和额外前置字段决定跳过、合并或保存历史。
// 资源清单优先复用缓存或上一条历史，无法复用时才扫描并保存资源。
// 返回布尔值表示历史已写入，合并覆盖原历史时也为 true，并不只表示创建不同 ID。
func (s *Service) RecordContentUpdate(pageID, authorID, summary string) (*Revision, bool, error) {
	page, err := s.pages.GetPage(pageID)
	if err != nil {
		return nil, false, err
	}
	return s.recordContentUpdateForPage(page, authorID, summary)
}

// RecordContentUpdates 返回与输入逐项对应的错误切片，不因单页失败停止其他页。
// 外部执行器按输入顺序串行使用，普通连接下按页面 ID 分组，不同页面并行处理。
// 每组保留输入次序，活动组数量受 GOMAXPROCS 限制，全部组完成后才返回。
func (s *Service) RecordContentUpdates(pages []*tree.Page, authorID, summary string) []error {
	errs := make([]error, len(pages))
	if len(pages) == 0 {
		return errs
	}

	// 单个外部事务不进入多组并行路径，避免在同一执行器上并发编排这些写操作。
	if s.store.executor != nil {
		for i, page := range pages {
			if page == nil {
				errs[i] = fmt.Errorf("page is required")
				continue
			}
			_, _, errs[i] = s.recordContentUpdateForPage(page, authorID, summary)
		}
		return errs
	}

	type batchItem struct {
		index int
		page  *tree.Page
	}

	// 同页重复输入归到同组，错误仍写回各自在原切片中的下标。
	grouped := make(map[string][]batchItem)
	nilItems := make([]int, 0)
	for i, page := range pages {
		if page == nil {
			nilItems = append(nilItems, i)
			continue
		}
		grouped[page.ID] = append(grouped[page.ID], batchItem{index: i, page: page})
	}

	for _, i := range nilItems {
		errs[i] = fmt.Errorf("page is required")
	}

	// 每组创建一个 goroutine，信号量只限制同时工作的组，不限制 goroutine 总数。
	parallelism := runtime.GOMAXPROCS(0)
	if parallelism < 1 {
		parallelism = 1
	}
	sem := make(chan struct{}, parallelism)

	var wg sync.WaitGroup
	wg.Add(len(grouped))
	for _, items := range grouped {
		go func(items []batchItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			for _, item := range items {
				if _, _, err := s.recordContentUpdateForPage(item.page, authorID, summary); err != nil {
					errs[item.index] = err
				}
			}
		}(items)
	}
	wg.Wait()

	return errs
}

// RecordAssetChange 持页面锁完整扫描资源，比较上一条历史的正文与资源清单哈希。
// 两项相同时直接返回旧历史，不单独以额外前置字段或页面结构变化触发记录。
// 需要保存时依次写正文、资源块、清单和历史描述，成功后更新缓存并执行数量裁剪。
func (s *Service) RecordAssetChange(pageID, authorID, summary string) (*Revision, bool, error) {
	mu := s.pageWriteLock(pageID)
	mu.Lock()
	defer mu.Unlock()

	prev, err := s.store.GetLatestRevision(pageID)
	if err != nil {
		return nil, false, err
	}

	state, err := s.capturePageState(pageID, true)
	if err != nil {
		return nil, false, err
	}

	if prev != nil &&
		prev.ContentHash == state.ContentHash &&
		prev.AssetManifestHash == state.AssetManifestHash {
		return prev, false, nil
	}

	contentHash, err := s.store.SaveContentBlob(pageID, []byte(state.Content))
	if err != nil {
		return nil, false, err
	}
	if contentHash != state.ContentHash {
		return nil, false, fmt.Errorf(errContentHashMismatch, state.ContentHash, contentHash)
	}

	if err := s.persistLiveAssets(pageID, state.Assets); err != nil {
		return nil, false, err
	}

	savedManifestHash, err := s.store.SaveAssetManifest(state.Assets)
	if err != nil {
		return nil, false, err
	}
	if savedManifestHash != state.AssetManifestHash {
		return nil, false, fmt.Errorf(errAssetManifestHashMismatch, state.AssetManifestHash, savedManifestHash)
	}

	rev, err := s.newRevision(RevisionTypeAssetUpdate, state, authorID, summary, savedManifestHash)
	if err != nil {
		return nil, false, err
	}
	if err := s.store.SaveRevision(rev); err != nil {
		return nil, false, err
	}
	s.assetManifestCache.Store(rev.PageID, assetManifestEntry{hash: savedManifestHash})
	s.pruneAfterSave(rev.PageID)

	return rev, true, nil
}

// RecordStructureChange 捕获页面结构与正文，复用或建立资源清单后写结构历史。
// 本方法不对照上一条历史跳过相同结构，每次成功都返回已写入状态。
func (s *Service) RecordStructureChange(pageID, authorID, summary string) (*Revision, bool, error) {
	mu := s.pageWriteLock(pageID)
	mu.Lock()
	defer mu.Unlock()

	prev, err := s.store.GetLatestRevision(pageID)
	if err != nil {
		return nil, false, err
	}

	state, err := s.capturePageState(pageID, false)
	if err != nil {
		return nil, false, err
	}

	assetManifestHash, err := s.resolveAssetManifestHash(pageID, prev)
	if err != nil {
		return nil, false, err
	}

	contentHash, err := s.store.SaveContentBlob(pageID, []byte(state.Content))
	if err != nil {
		return nil, false, err
	}
	if contentHash != state.ContentHash {
		return nil, false, fmt.Errorf(errContentHashMismatch, state.ContentHash, contentHash)
	}

	rev, err := s.newRevision(RevisionTypeStructureUpdate, state, authorID, summary, assetManifestHash)
	if err != nil {
		return nil, false, err
	}
	if err := s.store.SaveRevision(rev); err != nil {
		return nil, false, err
	}
	s.pruneAfterSave(rev.PageID)

	return rev, true, nil
}

// resolveAssetManifestHash 按内存缓存、上一条历史、完整资源扫描的顺序获得清单哈希。
// 缓存存在性检查执行数据库读取而非文件 Stat，缓存路径不解析清单 JSON。
func (s *Service) resolveAssetManifestHash(pageID string, prev *Revision) (string, error) {
	// 缓存记录可读即可复用，读失败清除缓存并继续后续恢复路径。
	if v, ok := s.assetManifestCache.Load(pageID); ok {
		entry := v.(assetManifestEntry)
		if s.store.AssetManifestExists(entry.hash) {
			return entry.hash, nil
		}
		s.assetManifestCache.Delete(pageID)
	}

	// 上一条清单须能够完整解码；失败在此分支不直接返回，而是转入现场扫描。
	if prev != nil && prev.AssetManifestHash != "" {
		if _, err := s.store.LoadAssetManifest(prev.AssetManifestHash); err == nil {
			s.assetManifestCache.Store(pageID, assetManifestEntry{hash: prev.AssetManifestHash})
			return prev.AssetManifestHash, nil
		}
	}

	// 无可复用清单时读取现场资源、保存块和清单，并核对两次清单编码的哈希。
	fullState, err := s.capturePageState(pageID, true)
	if err != nil {
		return "", err
	}
	if err := s.persistLiveAssets(pageID, fullState.Assets); err != nil {
		return "", err
	}
	savedManifestHash, err := s.store.SaveAssetManifest(fullState.Assets)
	if err != nil {
		return "", err
	}
	if savedManifestHash != fullState.AssetManifestHash {
		return "", fmt.Errorf(errAssetManifestHashMismatch, fullState.AssetManifestHash, savedManifestHash)
	}
	s.assetManifestCache.Store(pageID, assetManifestEntry{hash: savedManifestHash})
	return savedManifestHash, nil
}

// capturePageState 先构造页面状态并读取额外前置字段，按 withAssets 决定是否扫描资源。
// 不扫描时资源列表及清单哈希保持零值，任一步读取或编码失败不返回部分状态。
func (s *Service) capturePageState(pageID string, withAssets bool) (*RevisionState, error) {
	page, err := s.pages.GetPage(pageID)
	if err != nil {
		return nil, err
	}

	state := s.revisionStateFromPage(page)
	if err := s.enrichStateWithExtraFrontmatter(page.ID, state); err != nil {
		return nil, err
	}

	if !withAssets {
		return state, nil
	}

	assets, err := s.scanLiveAssets(pageID)
	if err != nil {
		return nil, err
	}

	state.Assets = assets
	hash, err := computeAssetManifestHash(assets)
	if err != nil {
		return nil, err
	}
	state.AssetManifestHash = hash

	return state, nil
}

// revisionStateFromPage 从非空页面复制结构、正文和元信息，并计算正文哈希。
// 根目录父节点表示为空父 ID，时间统一为 UTC，作者 ID 裁去边缘空白。
// 额外前置字段和资源不在此步骤读取。
func (s *Service) revisionStateFromPage(page *tree.Page) *RevisionState {
	parentID := ""
	if page.Parent != nil && page.Parent.ID != "root" {
		parentID = page.Parent.ID
	}

	return &RevisionState{
		PageID:        page.ID,
		ParentID:      parentID,
		Title:         page.Title,
		Slug:          page.Slug,
		Kind:          string(page.Kind),
		Path:          page.CalculatePath(),
		Content:       page.Content,
		ContentHash:   sha256HexBytes([]byte(page.Content)),
		PageCreatedAt: page.Metadata.CreatedAt.UTC(),
		PageUpdatedAt: page.Metadata.UpdatedAt.UTC(),
		CreatorID:     strings.TrimSpace(page.Metadata.CreatorID),
		LastAuthorID:  strings.TrimSpace(page.Metadata.LastAuthorID),
		CapturedAt:    time.Now().UTC(),
	}
}

// pageWriteLock 为每个精确页面 ID 复用一个互斥锁，锁映射不在使用后删除。
func (s *Service) pageWriteLock(pageID string) *sync.Mutex {
	v, _ := s.pageLocks.LoadOrStore(pageID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// shouldCoalesce 只合并同一作者的正文更新，前条时间距现在须位于闭区间零到窗口。
// 窗口非正数、没有前条、前条来自未来或其他操作类别时均不合并。
func (s *Service) shouldCoalesce(prev *Revision, authorID string) bool {
	if s.coalesceWindow <= 0 || prev == nil {
		return false
	}
	elapsed := time.Since(prev.CreatedAt)
	return prev.Type == RevisionTypeContentUpdate &&
		prev.AuthorID == strings.TrimSpace(authorID) &&
		elapsed >= 0 &&
		elapsed <= s.coalesceWindow
}

// recordContentUpdateForPage 要求非空页面，持锁读取前条历史并捕获额外前置字段。
// 正文与额外字段哈希都相同则跳过，不单独因标题、路径或资源差异创建正文历史。
func (s *Service) recordContentUpdateForPage(page *tree.Page, authorID, summary string) (*Revision, bool, error) {
	mu := s.pageWriteLock(page.ID)
	mu.Lock()
	defer mu.Unlock()

	prev, err := s.store.GetLatestRevision(page.ID)
	if err != nil {
		return nil, false, err
	}

	state := s.revisionStateFromPage(page)
	if err := s.enrichStateWithExtraFrontmatter(page.ID, state); err != nil {
		return nil, false, err
	}

	if prev != nil && prev.ContentHash == state.ContentHash && prev.ExtraFrontmatterHash == state.ExtraFrontmatterHash {
		return prev, false, nil
	}

	// 合并沿用历史 ID 和 CreatedAt，仅覆盖下列内容与描述字段，窗口不随合并延长。
	if s.shouldCoalesce(prev, authorID) {
		s.log.Debug("coalescing revision", "pageID", page.ID, "revisionID", prev.ID, "authorID", authorID)
		oldHash := prev.ContentHash
		contentHash, err := s.store.SaveContentBlob(page.ID, []byte(state.Content))
		if err != nil {
			return nil, false, err
		}
		if contentHash != state.ContentHash {
			return nil, false, fmt.Errorf(errContentHashMismatch, state.ContentHash, contentHash)
		}
		assetManifestHash, err := s.resolveAssetManifestHash(page.ID, prev)
		if err != nil {
			return nil, false, err
		}
		prev.ContentHash = contentHash
		prev.AssetManifestHash = assetManifestHash
		prev.ExtraFrontmatter = state.ExtraFrontmatter
		prev.ExtraFrontmatterHash = state.ExtraFrontmatterHash
		prev.Title = state.Title
		prev.Slug = state.Slug
		prev.Path = state.Path
		prev.PageUpdatedAt = state.PageUpdatedAt
		prev.LastAuthorID = state.LastAuthorID
		prev.Summary = summary
		if err := s.store.UpdateRevision(prev); err != nil {
			return nil, false, err
		}
		// 描述成功更新后，旧正文无引用才删除；清理失败只记录警告，不撤销历史更新。
		if oldHash != contentHash {
			if err := s.store.DeleteContentBlobIfUnreferenced(page.ID, oldHash); err != nil {
				s.log.Warn("failed to gc orphaned content blob", "pageID", page.ID, "hash", oldHash, "error", err)
			}
		}
		return prev, true, nil
	}

	assetManifestHash, err := s.resolveAssetManifestHash(page.ID, prev)
	if err != nil {
		return nil, false, err
	}

	contentHash, err := s.store.SaveContentBlob(page.ID, []byte(state.Content))
	if err != nil {
		return nil, false, err
	}
	if contentHash != state.ContentHash {
		return nil, false, fmt.Errorf(errContentHashMismatch, state.ContentHash, contentHash)
	}

	rev, err := s.newRevision(RevisionTypeContentUpdate, state, authorID, summary, assetManifestHash)
	if err != nil {
		return nil, false, err
	}
	if err := s.store.SaveRevision(rev); err != nil {
		return nil, false, err
	}
	s.pruneAfterSave(rev.PageID)

	return rev, true, nil
}

// newRevision 生成独立历史 ID 和当前 UTC 创建时间，并从捕获状态复制描述字段。
// 额外字段映射直接引用状态中的映射，不在此处深复制或持久化。
func (s *Service) newRevision(t RevisionType, state *RevisionState, authorID, summary, assetManifestHash string) (*Revision, error) {
	revisionID, err := shared.GenerateUniqueID()
	if err != nil {
		return nil, fmt.Errorf("generate revision id: %w", err)
	}

	return &Revision{
		ID:                   revisionID,
		PageID:               state.PageID,
		ParentID:             state.ParentID,
		Type:                 t,
		AuthorID:             strings.TrimSpace(authorID),
		CreatedAt:            time.Now().UTC(),
		Title:                state.Title,
		Slug:                 state.Slug,
		Kind:                 state.Kind,
		Path:                 state.Path,
		ContentHash:          state.ContentHash,
		ExtraFrontmatter:     state.ExtraFrontmatter,
		ExtraFrontmatterHash: state.ExtraFrontmatterHash,
		AssetManifestHash:    assetManifestHash,
		PageCreatedAt:        state.PageCreatedAt.UTC(),
		PageUpdatedAt:        state.PageUpdatedAt.UTC(),
		CreatorID:            strings.TrimSpace(state.CreatorID),
		LastAuthorID:         strings.TrimSpace(state.LastAuthorID),
		Summary:              summary,
	}, nil
}

// enrichStateWithExtraFrontmatter 重新读取原始页面前置字段并写入状态。
// 无额外字段时显式清空状态字段；解析失败返回错误，不使用旧值代替。
func (s *Service) enrichStateWithExtraFrontmatter(pageID string, state *RevisionState) error {
	if state == nil {
		return fmt.Errorf("revision state is required")
	}

	raw, err := s.pages.ReadPageRaw(pageID)
	if err != nil {
		return err
	}

	fm, _, has, err := markdown.ParseFrontmatter(raw)
	if err != nil {
		return err
	}
	if !has || len(fm.ExtraFields) == 0 {
		state.ExtraFrontmatter = nil
		state.ExtraFrontmatterHash = ""
		return nil
	}

	hash, err := hashExtraFrontmatter(fm.ExtraFields)
	if err != nil {
		return err
	}
	state.ExtraFrontmatter = fm.ExtraFields
	state.ExtraFrontmatterHash = hash
	return nil
}

// hashExtraFrontmatter 对非空额外字段的 JSON 编码计算哈希，空映射返回空哈希。
// 不支持 JSON 编码的值会返回带上下文错误，不生成替代编码。
func hashExtraFrontmatter(extra map[string]interface{}) (string, error) {
	if len(extra) == 0 {
		return "", nil
	}

	raw, err := json.Marshal(extra)
	if err != nil {
		return "", fmt.Errorf("marshal extra frontmatter: %w", err)
	}
	return sha256HexBytes(raw), nil
}
