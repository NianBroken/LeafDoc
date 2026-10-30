// Package search provides MySQL-backed Markdown indexing and retrieval.
package search

import (
	"bytes"
	"database/sql"
	"html"
	"sync"

	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/excerpt"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// MySQLIndex keeps one independently owned database handle for search.
type MySQLIndex struct {
	executor   database.Executor
	mu         sync.RWMutex
	storageDir string
	db         *sql.DB
}

// NewMySQLIndex connects search to the centrally configured schema.
func NewMySQLIndex(storageDir string) (*MySQLIndex, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &MySQLIndex{storageDir: storageDir, db: db}, nil
}

func (s *MySQLIndex) withDB(fn func(*sql.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		db, err := database.Open(s.storageDir)
		if err != nil {
			return err
		}
		s.db = db
	}
	return fn(s.db)
}

func (s *MySQLIndex) withDBRead(fn func(*sql.DB) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return sql.ErrConnDone
	}
	return fn(s.db)
}

func (s *MySQLIndex) Clear() error {
	return s.withExecutor(func(db database.Executor) error {
		_, err := db.Exec("DELETE FROM search_documents")
		return err
	})
}

func (s *MySQLIndex) Ping() error {
	return s.withDBRead(func(db *sql.DB) error { return db.Ping() })
}

func (s *MySQLIndex) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

var headingParser = goldmark.New()

func extractHeadings(src string) string {
	srcBytes := []byte(src)
	reader := text.NewReader(srcBytes)
	doc := headingParser.Parser().Parse(reader)

	var buf bytes.Buffer

	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if _, ok := n.(*ast.Heading); !ok {
			return ast.WalkContinue, nil
		}

		var headingText bytes.Buffer
		_ = ast.Walk(n, func(child ast.Node, childEntering bool) (ast.WalkStatus, error) {
			if !childEntering || child == n {
				return ast.WalkContinue, nil
			}
			if t, ok := child.(*ast.Text); ok {
				headingText.Write(t.Segment.Value(srcBytes))
				headingText.WriteByte(' ')
			}
			return ast.WalkContinue, nil
		})

		headingStr := strings.TrimSpace(headingText.String())
		if headingStr != "" {
			buf.WriteString(headingStr)
			buf.WriteByte('\n')
		}

		return ast.WalkSkipChildren, nil
	})

	return excerpt.PlainTextFromMarkdown(buf.String())
}

// IndexPageInput bundles one page's worth of IndexPages input.
type IndexPageInput struct {
	Path     string
	FilePath string
	PageID   string
	Title    string
	Kind     tree.NodeKind
	Raw      string
}

// IndexFailure pairs a failed input with why it failed, returned by
// IndexPages for inputs skipped due to a per-page problem (e.g.
// unparsable frontmatter) rather than a transaction-level failure.
type IndexFailure struct {
	PageID string
	Err    error
}

// IndexPages indexes multiple pages in a single transaction — used for
// subtree operations (recursive delete healing/move) where a naive
// one-IndexPage-call-per-page loop would pay for one commit per page.
// Load-tested: for a 200-page subtree this cut the search side effect's
// share of total Move/Delete cost from ~75-93% to a small fraction (see
// the Delete/Move/Rename load-test results).
//
// Inputs whose frontmatter fails to parse are skipped and reported via
// the returned []IndexFailure rather than aborting the whole batch — the
// rest still commit together. Only a transaction-level failure (e.g. the
// commit itself failing) returns a non-nil error, in which case none of
// the inputs were indexed. This preserves IndexPage's per-page fault
// isolation for the most likely failure mode (bad content) while still
// getting the throughput win for the common case (everything valid).
func (s *MySQLIndex) IndexPages(inputs []IndexPageInput) ([]IndexFailure, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	type prepared struct {
		path, filePath, pageID, title string
		kind                          tree.NodeKind
		headings, sanitizedBody       string
	}

	var failures []IndexFailure
	prepped := make([]prepared, 0, len(inputs))
	for _, in := range inputs {
		_, content, _, err := markdown.ParseFrontmatter(in.Raw)
		if err != nil {
			failures = append(failures, IndexFailure{PageID: in.PageID, Err: err})
			continue
		}
		content = excerpt.NormalizeMarkdownBody(content)
		prepped = append(prepped, prepared{
			path:          in.Path,
			filePath:      in.FilePath,
			pageID:        in.PageID,
			title:         in.Title,
			kind:          in.Kind,
			headings:      html.UnescapeString(extractHeadings(content)),
			sanitizedBody: html.UnescapeString(excerpt.PlainTextForSearch(content)),
		})
	}

	if len(prepped) == 0 {
		return failures, nil
	}

	err := s.withExecutor(func(db database.Executor) error {
		tx, err := database.BeginExecutor(db)
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()

		deleteStmt, err := tx.Prepare(`DELETE FROM search_documents WHERE pageID = ?`)
		if err != nil {
			return err
		}
		defer func() {
			_ = deleteStmt.Close()
		}()

		insertStmt, err := tx.Prepare(`
			INSERT INTO search_documents (path, filepath, pageID, kind, title, headings, content)
			VALUES (?, ?, ?, ?, ?, ?, ?);
		`)
		if err != nil {
			return err
		}
		defer func() {
			_ = insertStmt.Close()
		}()

		for _, p := range prepped {
			if _, err := deleteStmt.Exec(p.pageID); err != nil {
				return err
			}
			if _, err := insertStmt.Exec(p.path, p.filePath, p.pageID, string(p.kind), p.title, p.headings, p.sanitizedBody); err != nil {
				return err
			}
		}

		return tx.Commit()
	})

	return failures, err
}

// WithExecutor creates an index view whose writes share the caller's transaction.
func (s *MySQLIndex) WithExecutor(executor database.Executor) *MySQLIndex {
	return &MySQLIndex{storageDir: s.storageDir, executor: executor}
}

func (s *MySQLIndex) withExecutor(fn func(database.Executor) error) error {
	if s.executor != nil {
		return fn(s.executor)
	}
	return s.withDB(func(db *sql.DB) error { return fn(db) })
}

func (s *MySQLIndex) withExecutorRead(fn func(database.Executor) error) error {
	if s.executor != nil {
		return fn(s.executor)
	}
	return s.withDBRead(func(db *sql.DB) error { return fn(db) })
}
