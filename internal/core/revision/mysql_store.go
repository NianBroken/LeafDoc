package revision

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

type MySQLStore struct {
	executor   database.Executor
	storageDir string
	log        *slog.Logger
}

func NewMySQLStore(storageDir string, logger *slog.Logger) *MySQLStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &MySQLStore{
		storageDir: storageDir,
		log:        logger.With("component", "MySQLStore"),
	}
}

// SaveContentBlob stores content-addressed revision text exclusively in MySQL.
func (s *MySQLStore) SaveContentBlob(pageID string, content []byte) (string, error) {
	if err := validateStorageID(pageID); err != nil {
		return "", err
	}
	hash := sha256HexBytes(content)
	return hash, s.records().Write("revision_content", pageID+":"+hash, content)
}

func (s *MySQLStore) SaveAssetBlobFromPath(srcPath string) (string, int64, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", 0, fmt.Errorf("open live asset %s: %w", srcPath, err)
	}
	defer func() { _ = src.Close() }()

	tmpDir := filepath.Join(s.baseDir(), "tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", 0, fmt.Errorf("ensure tmp dir: %w", err)
	}

	tmp, err := os.CreateTemp(tmpDir, "asset-blob-*")
	if err != nil {
		return "", 0, fmt.Errorf("create temp asset blob: %w", err)
	}

	cleanupTmp := func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), src)
	if err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("copy asset to temp blob: %w", err)
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	dst := s.AssetBlobPath(hash)

	if err := tmp.Chmod(0o644); err != nil {
		cleanupTmp()
		return "", 0, fmt.Errorf("chmod temp asset blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", 0, fmt.Errorf("close temp asset blob: %w", err)
	}

	if fileExists(dst) {
		_ = os.Remove(tmp.Name())
		return hash, written, nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		_ = os.Remove(tmp.Name())
		return "", 0, fmt.Errorf("ensure asset blob dir: %w", err)
	}

	if err := os.Rename(tmp.Name(), dst); err != nil {
		if fileExists(dst) {
			_ = os.Remove(tmp.Name())
			return hash, written, nil
		}
		_ = os.Remove(tmp.Name())
		return "", 0, fmt.Errorf("move asset blob into place: %w", err)
	}

	return hash, written, nil
}

// SaveAssetManifest stores the canonical resource manifest in MySQL.
func (s *MySQLStore) SaveAssetManifest(items []AssetRef) (string, error) {
	raw, err := json.Marshal(assetManifest{Items: cloneAndSortAssetRefs(items)})
	if err != nil {
		return "", err
	}
	hash := sha256HexBytes(raw)
	return hash, s.records().Write("revision_manifests", hash, raw)
}

// SaveRevision persists one immutable revision descriptor.
func (s *MySQLStore) SaveRevision(rev *Revision) error {
	if rev == nil {
		return fmt.Errorf("revision is required")
	}
	if err := validateStorageID(rev.PageID); err != nil {
		return err
	}
	if err := validateStorageID(rev.ID); err != nil {
		return err
	}
	if rev.CreatedAt.IsZero() {
		return fmt.Errorf("created_at is required")
	}
	raw, err := json.Marshal(rev)
	if err != nil {
		return err
	}
	return s.records().Write("revisions", rev.PageID+":"+rev.ID, raw)
}

func (s *MySQLStore) ListRevisions(pageID string) ([]*Revision, error) {
	revisions, _, err := s.ListRevisionsPage(pageID, "", 0)
	if err != nil {
		return nil, err
	}
	return revisions, nil
}

// ListRevisionsPage orders database revisions newest first using stable revision cursors.
func (s *MySQLStore) ListRevisionsPage(pageID, cursor string, limit int) ([]*Revision, string, error) {
	if err := validateStorageID(pageID); err != nil {
		return nil, "", err
	}
	records, err := s.records().List("revisions")
	if err != nil {
		return nil, "", err
	}
	all := []*Revision{}
	for key, raw := range records {
		if !strings.HasPrefix(key, pageID+":") {
			continue
		}
		var rev Revision
		if err = json.Unmarshal(raw, &rev); err != nil {
			return nil, "", err
		}
		all = append(all, &rev)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].CreatedAt.Equal(all[j].CreatedAt) {
			return all[i].ID > all[j].ID
		}
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	start := 0
	if cursor != "" {
		start = len(all)
		for i, rev := range all {
			if rev.ID == cursor {
				start = i + 1
				break
			}
		}
	}
	end := len(all)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	next := ""
	if end < len(all) && end > start {
		next = all[end-1].ID
	}
	return all[start:end], next, nil
}

func (s *MySQLStore) GetLatestRevision(pageID string) (*Revision, error) {
	items, _, err := s.ListRevisionsPage(pageID, "", 1)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	return items[0], nil
}

func (s *MySQLStore) GetRevision(pageID, revisionID string) (*Revision, error) {
	if err := validateStorageID(pageID); err != nil {
		return nil, err
	}
	raw, err := s.records().Read("revisions", pageID+":"+revisionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	var rev Revision
	err = json.Unmarshal(raw, &rev)
	return &rev, err
}

// UpdateRevision persists the descriptor used by the coalescing policy.
func (s *MySQLStore) UpdateRevision(rev *Revision) error {
	if rev == nil {
		return fmt.Errorf("revision is required")
	}
	if _, err := s.GetRevision(rev.PageID, rev.ID); err != nil {
		return err
	}
	return s.SaveRevision(rev)
}

// PruneRevisions removes the oldest revision descriptors beyond keepCount.
// Descriptors are ordered by creation time and revision ID.
// Content blobs and asset manifests are NOT deleted — they are content-addressed and
// may be shared across multiple revisions.
func (s *MySQLStore) PruneRevisions(pageID string, keepCount int) (resultErr error) {
	if keepCount <= 0 {
		return nil
	}
	items, err := s.ListRevisions(pageID)
	if err != nil {
		return err
	}
	if len(items) <= keepCount {
		return nil
	}
	db, err := s.records().Connection()
	if err != nil {
		return err
	}
	tx, err := database.BeginExecutor(db)
	if err != nil {
		return err
	}
	defer database.Rollback(tx, &resultErr)
	for _, rev := range items[keepCount:] {
		if _, err = tx.Exec("DELETE FROM content_records WHERE namespace='revisions' AND record_key=?", pageID+":"+rev.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *MySQLStore) ReadContentBlob(pageID, hash string) ([]byte, error) {
	if err := validateStorageID(pageID); err != nil {
		return nil, err
	}
	if hash == "" {
		return []byte{}, nil
	}
	raw, err := s.records().Read("revision_content", pageID+":"+hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, os.ErrNotExist
	}
	return raw, err
}

// OpenContentBlob returns a streaming reader for the content blob.
// The caller is responsible for closing the returned ReadCloser.
// Use this instead of ReadContentBlob when you don't need the full content in memory.
// OpenContentBlob streams the revision text loaded from MySQL.
func (s *MySQLStore) OpenContentBlob(pageID, hash string) (io.ReadCloser, error) {
	raw, err := s.ReadContentBlob(pageID, hash)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(raw)), nil
}

// DeleteContentBlobIfUnreferenced deletes the content blob for the given hash if no
// revision of pageID still references it. Errors are non-fatal — callers should log them.
func (s *MySQLStore) DeleteContentBlobIfUnreferenced(pageID, hash string) error {
	items, err := s.ListRevisions(pageID)
	if err != nil {
		return err
	}
	for _, rev := range items {
		if rev.ContentHash == hash {
			return nil
		}
	}
	return s.records().Delete("revision_content", pageID+":"+hash)
}

func (s *MySQLStore) LoadAssetManifest(hash string) ([]AssetRef, error) {
	if hash == "" {
		return []AssetRef{}, nil
	}
	raw, err := s.records().Read("revision_manifests", hash)
	if err != nil {
		return nil, err
	}
	var manifest assetManifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return nil, err
	}
	return cloneAndSortAssetRefs(manifest.Items), nil
}

func (s *MySQLStore) ReadAssetBlob(hash string) ([]byte, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil, fmt.Errorf("asset hash is required")
	}

	raw, err := os.ReadFile(s.AssetBlobPath(hash))
	if err != nil {
		return nil, fmt.Errorf("read asset blob: %w", err)
	}
	return raw, nil
}

// OpenAssetBlob returns an open file handle for the given asset blob.
// The caller is responsible for closing the returned file.
func (s *MySQLStore) OpenAssetBlob(hash string) (*os.File, error) {
	hash = strings.TrimSpace(hash)
	if hash == "" {
		return nil, fmt.Errorf("asset hash is required")
	}
	f, err := os.Open(s.AssetBlobPath(hash))
	if err != nil {
		return nil, fmt.Errorf("open asset blob: %w", err)
	}
	return f, nil
}

// CopyAssetBlobToPath streams the asset blob identified by hash to dstPath,
// verifying hash and size during the copy. The write is atomic (temp + rename).
func (s *MySQLStore) CopyAssetBlobToPath(hash string, expectedSize int64, dstPath string) error {
	hash = strings.ToLower(strings.TrimSpace(hash))
	src, err := s.OpenAssetBlob(hash)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	tmpDir := filepath.Dir(dstPath)
	tmp, err := os.CreateTemp(tmpDir, "asset-restore-*")
	if err != nil {
		return fmt.Errorf("create temp restore file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = tmp.Close(); _ = os.Remove(tmpName) }

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hasher), src)
	if err != nil {
		cleanup()
		return fmt.Errorf("stream asset blob to %s: %w", dstPath, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		cleanup()
		return fmt.Errorf("chmod restored asset: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp restore file: %w", err)
	}
	if computedHash := hex.EncodeToString(hasher.Sum(nil)); computedHash != hash {
		_ = os.Remove(tmpName)
		return fmt.Errorf("asset blob hash mismatch: computed %s, want %s", computedHash, hash)
	}
	if written != expectedSize {
		_ = os.Remove(tmpName)
		return fmt.Errorf("asset blob size mismatch: got %d, want %d", written, expectedSize)
	}
	if err := os.Rename(tmpName, dstPath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("move restored asset into place: %w", err)
	}
	return nil
}

func (s *MySQLStore) DeletePageRevisions(pageID string) error {
	items, err := s.ListRevisions(pageID)
	if err != nil {
		return err
	}
	for _, rev := range items {
		if err = s.records().Delete("revisions", pageID+":"+rev.ID); err != nil {
			return err
		}
	}
	records, err := s.records().List("revision_content")
	if err != nil {
		return err
	}
	for key := range records {
		if strings.HasPrefix(key, pageID+":") {
			if err = s.records().Delete("revision_content", key); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *MySQLStore) baseDir() string {
	return filepath.Join(s.storageDir, ".leafdoc")
}

func (s *MySQLStore) contentBlobPath(pageID, hash string) string {
	return "mysql:revision_content:" + pageID + ":" + hash
}

func (s *MySQLStore) AssetBlobPath(hash string) string {
	return filepath.Join(s.baseDir(), "blobs", "assets", "sha256", shardHash(hash), hash)
}

func (s *MySQLStore) AssetManifestExists(hash string) bool {
	if hash == "" {
		return false
	}
	_, err := s.records().Read("revision_manifests", hash)
	return err == nil
}

func (s *MySQLStore) assetManifestPath(hash string) string {
	return "mysql:revision_manifests:" + hash
}

// validateStorageID checks that an ID is safe to use as a single file path component.
// Rejects empty strings, path separators, and dot-only segments like "." or "..".
func validateStorageID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("id must not be empty")
	}
	if strings.ContainsAny(id, "/\\") {
		return fmt.Errorf("id must not contain path separators")
	}
	if id == "." || id == ".." {
		return fmt.Errorf("id must not be a dot component")
	}
	return nil
}
func shardHash(hash string) string {
	if len(hash) < 2 {
		return "00"
	}
	return hash[:2]
}

func sha256HexBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func cloneAndSortAssetRefs(items []AssetRef) []AssetRef {
	cloned := make([]AssetRef, len(items))
	copy(cloned, items)

	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Name == cloned[j].Name {
			return cloned[i].SHA256 < cloned[j].SHA256
		}
		return cloned[i].Name < cloned[j].Name
	})

	return cloned
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *MySQLStore) records() database.Records {
	return database.Records{Dir: s.storageDir, Executor: s.executor}
}
