// 本文件持久化页面链接索引。正常、失效和标题歧义分别保存为状态零、一和二，
// 普通路径与 wikilink: 标题标记共用目标字段，正文内容不在此读取或修改。
package links

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

// LinksStore 可使用自己打开的连接或外部执行器。各实例的读写锁只协调本实例，
// WithExecutor 生成的对象拥有独立锁，外层事务的生命周期由调用方控制。
type LinksStore struct {
	executor   database.Executor
	mu         sync.RWMutex
	storageDir string
	db         *sql.DB
}

// 多页面出链查询每批最多绑定九百个 ID，避免单条 IN 条件不断增长。
// 资源关闭失败使用固定日志文本，日志内容属于运行输出。
const (
	maxOutgoingLinksQueryArgs = 900
	logCloseRowsFailed        = "could not close rows"
	logCloseStatementFailed   = "could not close statement"
)

// PageLinkUpdate 包含一个来源页的完整新出链及该页面自身的当前路径。
// ToPath 用于恢复其他页面指向此页的失效入链，不是所有 Targets 的共同目标。
type PageLinkUpdate struct {
	FromPageID string
	FromTitle  string
	ToPath     string
	Targets    []TargetLink
}

// BrokenLink 保存失效引用的来源与目标，FromPath 由服务层读取页面树后填充。
type BrokenLink struct {
	FromPageID string `json:"from_page_id"`
	FromTitle  string `json:"from_title"`
	FromPath   string `json:"from_path"`
	ToPath     string `json:"to_path"`
}

// NewLinksStore 打开链接索引所用的数据库，失败时不返回存储对象。
func NewLinksStore(storageDir string) (*LinksStore, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &LinksStore{storageDir: storageDir, db: db}, nil
}

// Connect 在数据库指针为空时按存储目录打开连接，不检查绑定执行器。
// 此方法自身不加锁，正常查询也不会自动调用它。
func (s *LinksStore) Connect() error {
	// 已有连接直接复用，不在此额外执行连通性探测。
	if s.db != nil {
		return nil
	}
	// 打开失败保留空指针，调用方收到原始错误。
	db, err := database.Open(s.storageDir)
	if err != nil {
		return err
	}
	s.db = db
	return nil
}

// DeleteOutgoingLinks 删除指定页面作为来源的全部链接，不处理指向它的入链。
// 没有匹配记录也返回 SQL 执行结果，不把零行更新当作业务错误。
func (s *LinksStore) DeleteOutgoingLinks(fromPageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`DELETE FROM links WHERE from_page_id = ?`, fromPageID)
	return err
}

// MarkIncomingLinksBroken 只处理当前已解析且目标 ID 匹配的记录，
// 清空目标 ID、置为失效，但保留来源和目标路径供后续展示或恢复。
func (s *LinksStore) MarkIncomingLinksBroken(toPageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`
		UPDATE links
		SET to_page_id = NULL,
		    state     = 1
		WHERE to_page_id = ?
		  AND state = 0
	`, toPageID)

	return err
}

// MarkLinksBrokenForPath 按精确目标路径将已解析记录置为失效，不处理歧义或既有失效行。
func (s *LinksStore) MarkLinksBrokenForPath(toPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`
		UPDATE links
		SET to_page_id = NULL,
		    state     = 1
		WHERE to_path = ?
		  AND state   = 0
	`, toPath)

	return err
}

// MarkLinksBrokenForPrefix 使用等值及前缀加斜杠的 LIKE 条件匹配已解析记录。
// 输入直接作为参数，不在此归一化，也不转义输入中已有的 LIKE 通配符。
func (s *LinksStore) MarkLinksBrokenForPrefix(oldPrefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`
		UPDATE links
		SET to_page_id = NULL,
		    state     = 1
		WHERE state = 0
		  AND (
		    to_path = ?
		    OR to_path LIKE CONCAT(?, '/%')
		  )
	`, oldPrefix, oldPrefix)

	return err
}

// AddLinks 先删除来源页全部出链，再逐条写入新目标；空目标列表会清空该页出链。
// 自有连接会开启真实事务，外部执行器由 BeginExecutor 借用，提交与回滚不结束外层事务。
func (s *LinksStore) AddLinks(fromPageID string, fromTitle string, toLinks []TargetLink) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := database.BeginExecutor(s.queryExecutor())
	if err != nil {
		return err
	}

	// 整页替换避免正文移除的链接仍留在索引中，删除失败时尝试回滚并合并错误。
	_, err = tx.Exec(`DELETE FROM links WHERE from_page_id = ?`, fromPageID)
	if err != nil {
		rbErr := tx.Rollback()
		base := fmt.Errorf("failed to clear existing links for page %s", fromPageID)

		if rbErr != nil {
			return errors.Join(base, err, rbErr)
		}
		return errors.Join(base, err)
	}

	stmt, err := tx.Prepare(`REPLACE INTO links(from_page_id, to_page_id, to_path, from_title, state) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		rbErr := tx.Rollback()
		base := fmt.Errorf("failed to prepare insert statement for links from page %s", fromPageID)

		if rbErr != nil {
			return errors.Join(base, err, rbErr)
		}
		return errors.Join(base, err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			slog.Default().Error(logCloseStatementFailed, "error", err)
		}
	}()

	// 同一预编译语句逐条写入，REPLACE 的覆盖范围由数据库唯一约束决定。
	for _, link := range toLinks {
		_, err := stmt.Exec(fromPageID, link.TargetPageID, link.TargetPagePath, fromTitle, int(link.State))
		if err != nil {
			rbErr := tx.Rollback()
			base := fmt.Errorf("failed to insert link from %s to %s", fromPageID, link.TargetPageID)

			if rbErr != nil {
				return errors.Join(base, err, rbErr)
			}
			return errors.Join(base, err)
		}
	}

	return tx.Commit()
}

// ReplaceLinksAndHeal 在一个自有事务或外层执行器视图中替换多页出链并恢复失效入链。
// 处理失败尝试回滚并保留回滚错误，成功返回提交结果。
func (s *LinksStore) ReplaceLinksAndHeal(updates []PageLinkUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := database.BeginExecutor(s.queryExecutor())
	if err != nil {
		return err
	}

	if err := s.replaceLinksAndHealTx(tx, updates); err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil {
			return errors.Join(err, rbErr)
		}
		return err
	}

	return tx.Commit()
}

// replaceLinksAndHealTx 准备删除、写入、恢复三种语句。先替换所有来源页记录，
// 再逐页恢复指向其精确路径的失效入链，避免随后整页删除覆盖刚恢复的结果。
// 每条语句均延迟关闭，关闭失败记录日志，不覆盖主要执行错误。
func (s *LinksStore) replaceLinksAndHealTx(tx database.Executor, updates []PageLinkUpdate) error {
	deleteStmt, err := tx.Prepare(`DELETE FROM links WHERE from_page_id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement for batched link update: %w", err)
	}
	defer func() {
		if err := deleteStmt.Close(); err != nil {
			slog.Default().Error(logCloseStatementFailed, "error", err)
		}
	}()

	insertStmt, err := tx.Prepare(`REPLACE INTO links(from_page_id, to_page_id, to_path, from_title, state) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement for batched link update: %w", err)
	}
	defer func() {
		if err := insertStmt.Close(); err != nil {
			slog.Default().Error(logCloseStatementFailed, "error", err)
		}
	}()

	healStmt, err := tx.Prepare(`
		UPDATE links
		SET to_page_id = ?, state = 0
		WHERE to_path = ? AND state = 1
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare heal statement for batched link update: %w", err)
	}
	defer func() {
		if err := healStmt.Close(); err != nil {
			slog.Default().Error(logCloseStatementFailed, "error", err)
		}
	}()

	for _, update := range updates {
		if _, err := deleteStmt.Exec(update.FromPageID); err != nil {
			return fmt.Errorf("failed to clear existing links for page %s: %w", update.FromPageID, err)
		}

		for _, link := range update.Targets {
			if _, err := insertStmt.Exec(update.FromPageID, link.TargetPageID, link.TargetPagePath, update.FromTitle, int(link.State)); err != nil {
				return fmt.Errorf("failed to insert link from %s to %s: %w", update.FromPageID, link.TargetPageID, err)
			}
		}
	}

	for _, update := range updates {
		if _, err := healStmt.Exec(update.FromPageID, update.ToPath); err != nil {
			return fmt.Errorf("failed to heal links for path %s: %w", update.ToPath, err)
		}
	}

	return nil
}

// GetBacklinksForPage 只查询目标 ID 匹配且状态为已解析的记录，不包含歧义双链。
// SQL 没有排序条件，空目标 ID 转为空文本；扫描或遍历失败不返回部分列表。
func (s *LinksStore) GetBacklinksForPage(pageID string) ([]Backlink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows, err := s.queryExecutor().Query(`SELECT from_page_id, to_page_id, from_title FROM links WHERE to_page_id = ? AND state = 0`, pageID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var backlinks []Backlink
	for rows.Next() {
		var b Backlink
		var toPageID sql.NullString
		if err := rows.Scan(&b.FromPageID, &toPageID, &b.FromTitle); err != nil {
			return nil, err
		}
		if toPageID.Valid {
			b.ToPageID = toPageID.String
		} else {
			b.ToPageID = ""
		}
		backlinks = append(backlinks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return backlinks, nil
}

// GetOutgoingLinksForPage 读取指定来源的全部状态，保存目标路径及可空目标 ID。
// 不检查目标页面是否仍在树中，也不在此转换成界面结果。
func (s *LinksStore) GetOutgoingLinksForPage(pageID string) ([]Outgoing, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queryExecutor().Query(`
        SELECT from_page_id, to_page_id, to_path, from_title, state
        FROM links
        WHERE from_page_id = ?
    `, pageID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var outgoings []Outgoing
	for rows.Next() {
		var o Outgoing
		var toPageID sql.NullString
		var stateInt int

		if err := rows.Scan(&o.FromPageID, &toPageID, &o.ToPath, &o.FromTitle, &stateInt); err != nil {
			return nil, err
		}

		if toPageID.Valid {
			o.ToPageID = toPageID.String
		} else {
			o.ToPageID = ""
		}

		o.State = LinkState(stateInt)
		outgoings = append(outgoings, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return outgoings, nil
}

// GetOutgoingLinksForPages 将输入按批查询，以来源 ID 分组累积出链。
// 空输入返回空映射，无出链页面不专门写入映射键；批次失败丢弃已收集的结果。
func (s *LinksStore) GetOutgoingLinksForPages(pageIDs []string) (map[string][]Outgoing, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(pageIDs) == 0 {
		return map[string][]Outgoing{}, nil
	}

	outgoingByPageID := make(map[string][]Outgoing, len(pageIDs))
	// 输入不去重，跨批次重复的 ID 会让对应记录重复追加到结果中。
	for start := 0; start < len(pageIDs); start += maxOutgoingLinksQueryArgs {
		end := start + maxOutgoingLinksQueryArgs
		if end > len(pageIDs) {
			end = len(pageIDs)
		}

		if err := s.appendOutgoingLinksForPageBatch(outgoingByPageID, pageIDs[start:end]); err != nil {
			return nil, err
		}
	}

	return outgoingByPageID, nil
}

// appendOutgoingLinksForPageBatch 仅把占位符拼入 SQL，实际 ID 通过参数绑定。
// 调用方保证批次非空并持有读锁；查询按来源 ID 排序，同页出链没有额外顺序约束。
func (s *LinksStore) appendOutgoingLinksForPageBatch(outgoingByPageID map[string][]Outgoing, pageIDs []string) error {
	placeholders := strings.TrimRight(strings.Repeat("?,", len(pageIDs)), ",")
	args := make([]any, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		args = append(args, pageID)
	}

	rows, err := s.queryExecutor().Query(`
        SELECT from_page_id, to_page_id, to_path, from_title, state
        FROM links
        WHERE from_page_id IN (`+placeholders+`)
        ORDER BY from_page_id
    `, args...)
	if err != nil {
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	for rows.Next() {
		var outgoing Outgoing
		var toPageID sql.NullString
		var stateInt int

		if err := rows.Scan(&outgoing.FromPageID, &toPageID, &outgoing.ToPath, &outgoing.FromTitle, &stateInt); err != nil {
			return err
		}

		if toPageID.Valid {
			outgoing.ToPageID = toPageID.String
		}
		outgoing.State = LinkState(stateInt)
		outgoingByPageID[outgoing.FromPageID] = append(outgoingByPageID[outgoing.FromPageID], outgoing)
	}

	return rows.Err()
}

// GetRefactorMatchesForPrefix 查询路径本身及下级范围，所有状态都可成为重构候选。
// 只把 LinkBroken 映射为 Broken，不返回目标 ID，也不转义输入中的 LIKE 通配符。
func (s *LinksStore) GetRefactorMatchesForPrefix(oldPrefix string) ([]RefactorLinkMatch, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queryExecutor().Query(`
		SELECT from_page_id, from_title, to_path, state
		FROM links
		WHERE to_path = ? OR to_path LIKE ?
	`, oldPrefix, oldPrefix+"/%")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var matches []RefactorLinkMatch
	for rows.Next() {
		var match RefactorLinkMatch
		var stateInt int
		if err := rows.Scan(&match.FromPageID, &match.FromTitle, &match.ToPath, &stateInt); err != nil {
			return nil, err
		}
		match.Broken = LinkState(stateInt) == LinkBroken
		matches = append(matches, match)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}

// GetRefactorSourcePageIDsForPrefix 用 DISTINCT 查询引用路径范围的来源 ID，
// 不限定状态或输出顺序，不读取页面正文。
func (s *LinksStore) GetRefactorSourcePageIDsForPrefix(oldPrefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queryExecutor().Query(`
		SELECT DISTINCT from_page_id
		FROM links
		WHERE to_path = ? OR to_path LIKE ?
	`, oldPrefix, oldPrefix+"/%")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var pageIDs []string
	for rows.Next() {
		var pageID string
		if err := rows.Scan(&pageID); err != nil {
			return nil, err
		}
		pageIDs = append(pageIDs, pageID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pageIDs, nil
}

// GetRefactorSourcePageIDsForWikiLinkTitle 比较小写后的标题标记并去重来源。
// 此入口不裁去 title 两边空白，也不只筛选已解析或歧义状态。
func (s *LinksStore) GetRefactorSourcePageIDsForWikiLinkTitle(title string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queryExecutor().Query(`
		SELECT DISTINCT from_page_id
		FROM links
		WHERE LOWER(to_path) = ?
	`, strings.ToLower(wikilinkSentinelPrefix+title))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var pageIDs []string
	for rows.Next() {
		var pageID string
		if err := rows.Scan(&pageID); err != nil {
			return nil, err
		}
		pageIDs = append(pageIDs, pageID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pageIDs, nil
}

// GetBrokenIncomingForPath 按精确路径读取状态为失效的入链，按来源标题升序返回。
// 每条结果显式标为 Broken，目标 ID 为空时保留空文本。
func (s *LinksStore) GetBrokenIncomingForPath(toPath string) ([]Backlink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queryExecutor().Query(`
		SELECT from_page_id, to_page_id, from_title
		FROM links
		WHERE to_path = ? AND state = 1
		ORDER BY from_title ASC
	`, toPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var backlinks []Backlink
	for rows.Next() {
		var b Backlink
		var toPageID sql.NullString
		if err := rows.Scan(&b.FromPageID, &toPageID, &b.FromTitle); err != nil {
			return nil, err
		}
		if toPageID.Valid {
			b.ToPageID = toPageID.String
		} else {
			b.ToPageID = ""
		}
		b.Broken = true
		backlinks = append(backlinks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return backlinks, nil
}

// HealLinksForPath 将精确路径对应的失效记录绑定到页面 ID 并置为已解析。
// 不影响已解析或歧义行，不验证 pageID 是否存在于页面树。
func (s *LinksStore) HealLinksForPath(toPath string, pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`
		UPDATE links
		SET to_page_id = ?, state = 0
		WHERE to_path = ? AND state = 1
	`, pageID, toPath)

	return err
}

// sentinelKey 裁去标题边缘空白，加上双链标记后转为小写，供 SQL 的 LOWER 比较使用。
// 字符转换由 Go 与数据库分别执行，此函数不检查数据库排序规则。
func sentinelKey(title string) string {
	return strings.ToLower(wikilinkSentinelPrefix + strings.TrimSpace(title))
}

// ReconcileWikiLinksForTitle 将调用方计算的状态写入全部同标题标记记录。
// 已解析状态使用传入 pageID，其余状态把目标 ID 写为 SQL NULL。
// 标题查询使用小写比较，不在存储层重新统计匹配页面。
func (s *LinksStore) ReconcileWikiLinksForTitle(title string, state LinkState, pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var target any
	if state == LinkResolved {
		target = pageID
	}

	_, err := s.queryExecutor().Exec(`
		UPDATE links
		SET to_page_id = ?, state = ?
		WHERE LOWER(to_path) = ?
	`, target, int(state), sentinelKey(title))

	return err
}

// GetAmbiguousIncomingForTitle 查询状态为歧义的同标题标记来源，按来源标题升序排列。
// 查询只填来源 ID 和标题，具体目标由服务层针对当前匹配页面填写。
func (s *LinksStore) GetAmbiguousIncomingForTitle(title string) ([]Backlink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queryExecutor().Query(`
		SELECT from_page_id, from_title
		FROM links
		WHERE LOWER(to_path) = ? AND state = 2
		ORDER BY from_title ASC
	`, sentinelKey(title))
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			slog.Default().Error(logCloseRowsFailed, "error", err)
		}
	}()

	var backlinks []Backlink
	for rows.Next() {
		var b Backlink
		if err := rows.Scan(&b.FromPageID, &b.FromTitle); err != nil {
			return nil, err
		}
		backlinks = append(backlinks, b)
	}
	return backlinks, rows.Err()
}

// GetBrokenLinks 返回全部失效索引，依次按目标路径、来源标题排序。
// 不查询来源当前路径，延迟关闭游标时的错误未纳入返回值。
func (s *LinksStore) GetBrokenLinks() ([]BrokenLink, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.queryExecutor().Query(`
        SELECT from_page_id, from_title, to_path
        FROM links
        WHERE state = 1
        ORDER BY to_path, from_title
    `)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var result []BrokenLink

	for rows.Next() {
		var link BrokenLink

		if err := rows.Scan(
			&link.FromPageID,
			&link.FromTitle,
			&link.ToPath,
		); err != nil {
			return nil, err
		}

		result = append(result, link)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// Clear 删除全部索引行，不修改页面内容，也不依据受影响行数产生错误。
func (s *LinksStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`DELETE FROM links`)
	return err
}

// Close 只关闭本对象的自有数据库连接，不关闭绑定的外部执行器。
// 成功后清空连接指针，失败时保留指针并返回关闭错误。
func (s *LinksStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		err := s.db.Close()
		if err != nil {
			return err
		}
		s.db = nil
	}
	return nil
}

// WithExecutor 创建使用外部执行器的独立视图，不复制自有连接或当前锁状态。
func (s *LinksStore) WithExecutor(executor database.Executor) *LinksStore {
	return &LinksStore{executor: executor, storageDir: s.storageDir}
}

// queryExecutor 优先选择绑定执行器，否则直接返回自有连接，不自动重连。
func (s *LinksStore) queryExecutor() database.Executor {
	if s.executor != nil {
		return s.executor
	}
	return s.db
}
