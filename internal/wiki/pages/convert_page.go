package pages

import (
	"context"
	"log/slog"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/revision"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// ConvertPageInput is the input for ConvertPageUseCase.
type ConvertPageInput struct {
	UserID     string
	ID         string
	Version    string
	TargetKind tree.NodeKind
}

// ConvertPageUseCase converts a page to a different node kind (page ↔ section).
type ConvertPageUseCase struct {
	tree     *tree.TreeService
	revision *revision.Service
	log      *slog.Logger
}

// NewConvertPageUseCase constructs a ConvertPageUseCase.
func NewConvertPageUseCase(t *tree.TreeService, r *revision.Service, log *slog.Logger) *ConvertPageUseCase {
	return &ConvertPageUseCase{tree: t, revision: r, log: log}
}

// Execute converts the node kind and records a structure revision.
func (uc *ConvertPageUseCase) Execute(ctx context.Context, in ConvertPageInput) error {
	if !uc.tree.InUnitOfWork() {
		return uc.tree.WithUnitOfWork(ctx, func(bound *tree.TreeService, exec database.Executor) error {
			copy := *uc
			copy.tree = bound
			copy.revision = uc.revision.WithExecutor(exec, bound)
			return copy.Execute(ctx, in)
		})
	}

	if in.ID == "root" || in.ID == "" {
		return newPageRootOperationError("convert")
	}
	in.Version = sanitizeClientVersion(in.Version)
	if err := uc.tree.ConvertNode(in.UserID, in.ID, in.TargetKind, in.Version); err != nil {
		return err
	}
	if uc.revision != nil {
		if _, _, err := uc.revision.RecordStructureChange(in.ID, in.UserID, ""); err != nil {
			return err
		}
	}
	return nil
}
