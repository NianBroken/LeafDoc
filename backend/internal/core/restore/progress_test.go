package restore

import (
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

// TestProgressCredentialLifecycle covers rotation and denial without exposing
// administrator status to callers that lack the per-job credential.
func TestProgressCredentialLifecycle(t *testing.T) {
	manager := NewManager(Config{})
	if _, ok := manager.ProgressStatus(""); ok {
		t.Fatal("idle manager authorized anonymous progress")
	}
	manager.progressMu.Lock()
	first, err := manager.startProgressJob()
	manager.progressMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	random, err := hex.DecodeString(first)
	if err != nil || len(random) != 32 {
		t.Fatal("credential is not 256 random bits")
	}
	if _, ok := manager.ProgressStatus(first); !ok {
		t.Fatal("current credential rejected")
	}
	if _, ok := manager.ProgressStatus("wrong"); ok {
		t.Fatal("incorrect credential accepted")
	}
	manager.progressMu.Lock()
	_, err = manager.startProgressJob()
	manager.progressMu.Unlock()
	if !errors.Is(err, ErrRestoreAlreadyRunning) {
		t.Fatalf("concurrent start: %v", err)
	}
	if _, ok := manager.ProgressStatus(first); !ok {
		t.Fatal("rejected concurrent job invalidated active credential")
	}
	manager.job.Finish(nil)
	if status, ok := manager.ProgressStatus(first); !ok || !status.Done {
		t.Fatal("completed job inaccessible after session invalidation")
	}
	manager.progressMu.Lock()
	second, err := manager.startProgressJob()
	manager.progressMu.Unlock()
	if err != nil || second == first {
		t.Fatal("new job failed to rotate credential")
	}
	if _, ok := manager.ProgressStatus(first); ok {
		t.Fatal("previous job credential accepted")
	}
	if _, ok := manager.ProgressStatus(second); !ok {
		t.Fatal("new credential rejected")
	}
	if _, ok := NewManager(Config{}).ProgressStatus(second); ok {
		t.Fatal("credential survived manager restart")
	}
	manager.progressMu.Lock()
	manager.progressExpiresAt = time.Now().Add(-time.Second)
	manager.progressMu.Unlock()
	if _, ok := manager.ProgressStatus(second); ok {
		t.Fatal("expired progress credential accepted")
	}

	manager.job.Finish(nil)
}
