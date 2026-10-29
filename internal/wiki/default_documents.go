package wiki

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/NianBroken/LeafDoc/internal/core/defaultdocs"
	"github.com/NianBroken/LeafDoc/internal/core/shared"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	wikipages "github.com/NianBroken/LeafDoc/internal/wiki/pages"
)

const defaultDocumentsStateFile = defaultdocs.StateFile

// defaultDocumentsState separates an interrupted initialization from intentional deletion.
type defaultDocumentsState struct {
	Complete bool `json:"complete"`
}

// EnsureDefaultDocuments initializes empty instances once, resuming interrupted writes.
// Existing nonempty instances are left intact. Completed instances never recreate deleted pages.
func (w *Wiki) EnsureDefaultDocuments() error {
	statePath := filepath.Join(w.storageDir, defaultDocumentsStateFile)
	state := defaultDocumentsState{}
	data, err := os.ReadFile(statePath)
	if err == nil {
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("read default document state: %w", err)
		}
		if state.Complete {
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	} else {
		if w.tree.HasPages() {
			return shared.WriteFileAtomic(statePath, []byte("{\"complete\":true}\n"), 0o644)
		}
		if err := shared.WriteFileAtomic(statePath, []byte("{\"complete\":false}\n"), 0o644); err != nil {
			return err
		}
	}
	documents, err := defaultdocs.Load()
	if err != nil {
		return err
	}
	kind := tree.NodeKindPage
	orchestrator := w.newPageOrchestrator()
	for _, document := range documents {
		var id string
		for _, node := range w.tree.GetTree().Children {
			if node.Slug == document.Slug {
				if node.Metadata.CreatorID != SYSTEM_USER_ID {
					return fmt.Errorf("default document path is occupied: %s", document.Slug)
				}
				id = node.ID
				break
			}
		}
		if id == "" {
			out, err := wikipages.NewCreatePageUseCase(w.tree, w.slug, orchestrator, w.log, w.metrics).Execute(context.Background(), wikipages.CreatePageInput{
				UserID: SYSTEM_USER_ID, Title: document.Title, Slug: document.Slug, Kind: &kind,
			})
			if err != nil {
				return err
			}
			id = out.Page.ID
		}
		page, err := w.tree.GetPage(id)
		if err != nil {
			return err
		}
		if _, err := wikipages.NewUpdatePageUseCase(w.tree, w.slug, orchestrator, w.log, w.metrics).Execute(context.Background(), wikipages.UpdatePageInput{
			UserID: SYSTEM_USER_ID, ID: id, Version: page.Version(), Title: document.Title, Slug: document.Slug, Content: &document.Content, Kind: &kind,
		}); err != nil {
			return err
		}
	}
	return shared.WriteFileAtomic(statePath, []byte("{\"complete\":true}\n"), 0o644)
}
