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

// CapturePageState returns a full detached snapshot including current assets.
// This is the "expensive" path and is mainly used for asset changes and delete.
func (s *Service) CapturePageState(pageID string) (*RevisionState, error) {
	return s.capturePageState(pageID, true)
}

// RecordContentUpdate records a content revision.
// Performance choice for V1:
//   - only content is re-hashed every time
//   - the latest asset manifest is reused if it already exists
//   - if this is the first revision for the page, assets are captured once
//
// Assumption: asset changes go through Upload/Rename/Delete hooks and call RecordAssetChange.
func (s *Service) RecordContentUpdate(pageID, authorID, summary string) (*Revision, bool, error) {
	page, err := s.pages.GetPage(pageID)
	if err != nil {
		return nil, false, err
	}
	return s.recordContentUpdateForPage(page, authorID, summary)
}

func (s *Service) RecordContentUpdates(pages []*tree.Page, authorID, summary string) []error {
	errs := make([]error, len(pages))
	if len(pages) == 0 {
		return errs
	}

	type batchItem struct {
		index int
		page  *tree.Page
	}

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

// RecordAssetChange records a full snapshot when live assets changed.
// This method hashes the current assets and only writes a new revision when
// content or the asset manifest actually changed.
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

func (s *Service) resolveAssetManifestHash(pageID string, prev *Revision) (string, error) {
	// Check in-memory cache first — avoids a full asset scan on every content save.
	// Use a stat to verify the file still exists without parsing its JSON content.
	if v, ok := s.assetManifestCache.Load(pageID); ok {
		entry := v.(assetManifestEntry)
		if s.store.AssetManifestExists(entry.hash) {
			return entry.hash, nil
		}
		s.assetManifestCache.Delete(pageID)
	}

	if prev != nil && prev.AssetManifestHash != "" {
		if _, err := s.store.LoadAssetManifest(prev.AssetManifestHash); err == nil {
			s.assetManifestCache.Store(pageID, assetManifestEntry{hash: prev.AssetManifestHash})
			return prev.AssetManifestHash, nil
		}
	}

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

func (s *Service) pageWriteLock(pageID string) *sync.Mutex {
	v, _ := s.pageLocks.LoadOrStore(pageID, &sync.Mutex{})
	return v.(*sync.Mutex)
}

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
