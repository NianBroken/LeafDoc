// Package search 使用 MySQL 保存 Markdown 派生文本，并提供字面匹配和全文检索。
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

// MySQLIndex 持有独立连接或外部执行器，普通读使用读锁，普通写与关闭使用写锁。
type MySQLIndex struct {
	executor   database.Executor
	mu         sync.RWMutex
	storageDir string
	db         *sql.DB
}

// NewMySQLIndex 打开统一配置对应的数据库，连接归该索引对象管理。
func NewMySQLIndex(storageDir string) (*MySQLIndex, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &MySQLIndex{storageDir: storageDir, db: db}, nil
}

// withDB 持写锁运行回调，连接字段为空时先重新打开数据库。
// 因此 Close 后的普通写操作能够重新建连，不代表对象永久不可用。
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

// withDBRead 持读锁运行回调，空连接直接返回 sql.ErrConnDone，不尝试重新建连。
func (s *MySQLIndex) withDBRead(fn func(*sql.DB) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.db == nil {
		return sql.ErrConnDone
	}
	return fn(s.db)
}

// Clear 删除全部搜索文档，不删除页面正文或标签索引。
func (s *MySQLIndex) Clear() error {
	return s.withExecutor(func(db database.Executor) error {
		_, err := db.Exec("DELETE FROM search_documents")
		return err
	})
}

// Ping 检查本对象连接的连通性，不通过外部执行器视图执行。
func (s *MySQLIndex) Ping() error {
	return s.withDBRead(func(db *sql.DB) error { return db.Ping() })
}

// Close 关闭并清空普通连接，即使关闭返回错误也清空字段，不关闭外部执行器。
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

// headingParser 为标题提取复用默认 Markdown 解析器。
var headingParser = goldmark.New()

// extractHeadings 遍历标题节点中的文本子节点，节点间插入空格、标题间插入换行。
// 收集结果再经过公共纯文本清理，非标题内容不参与此字段。
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

// IndexPageInput 组合一页索引的身份、路径、标题、类型和含前置字段的原始正文。
type IndexPageInput struct {
	Path     string
	FilePath string
	PageID   string
	Title    string
	Kind     tree.NodeKind
	Raw      string
}

// IndexFailure 记录正文准备阶段失败的页面及错误，与整批数据库写入错误分开返回。
type IndexFailure struct {
	PageID string
	Err    error
}

// IndexPages 在建事务前逐页解析前置字段、提取标题和搜索文本，并解码 HTML 实体。
// 解析失败的页记入 failures 并跳过，其他页使用删除后插入方式共同写入。
// 普通连接使用一笔事务，外部执行器不在此结束事务；写入或提交失败单独返回 error。
// 无有效页时直接返回准备失败列表，不打开写事务；失败页已有索引不在此清除。
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
	// 所有文本准备在存储锁和事务之外完成，避免把 Markdown 解析时间计入持锁写入。
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

		// 同页重复输入按输入顺序覆盖，普通事务最终只留下最后一次成功插入的内容。
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

// WithExecutor 创建外部执行器视图，读写都优先使用该执行器，不复制原连接。
func (s *MySQLIndex) WithExecutor(executor database.Executor) *MySQLIndex {
	return &MySQLIndex{storageDir: s.storageDir, executor: executor}
}

// withExecutor 优先使用外部执行器，普通连接路径按需重连并持写锁。
func (s *MySQLIndex) withExecutor(fn func(database.Executor) error) error {
	if s.executor != nil {
		return fn(s.executor)
	}
	return s.withDB(func(db *sql.DB) error { return fn(db) })
}

// withExecutorRead 优先使用外部执行器，普通连接路径只持读锁且不重连。
func (s *MySQLIndex) withExecutorRead(fn func(database.Executor) error) error {
	if s.executor != nil {
		return fn(s.executor)
	}
	return s.withDBRead(func(db *sql.DB) error { return fn(db) })
}
