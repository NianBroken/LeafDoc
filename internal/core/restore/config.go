package restore

import (
	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/settings"
	"github.com/NianBroken/LeafDoc/internal/core/snapshot"
)

// Config wires transactional database restoration and file-resource maintenance.
type Config struct {
	SnapshotManager    *snapshot.Manager
	DataDir            string
	WikiVersion        string
	WriteGate          *WriteGate
	AuthService        *auth.AuthService
	Reloadables        []settings.Reloadable
	UserResolver       *auth.UserResolver
	TriggerResync      func()
	ReloadRuntime      func() error
	MaxUploadSizeBytes int64
}
