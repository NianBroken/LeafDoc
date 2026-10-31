// 本文件维护 page_tags 与 page_meta 表，标签和摘要均是能够由正文重建的索引。
package tags

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

const (
	logCloseRowsFailed = "could not close rows"
	sqlLimitFmt        = " LIMIT %d"
)

// TagsStore 用互斥锁串行化单个存储实例的访问，可绑定外层事务执行器。
// 执行器视图不持有原连接，其锁也不与原实例共享。
type TagsStore struct {
	executor database.Executor
	mu       sync.Mutex
	db       *sql.DB
}

// TagCount 表示一个标签及拥有该标签的不同页面数量。
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// NewTagsStore 打开配置对应的 MySQL 连接，成功后由存储对象负责关闭。
func NewTagsStore(storageDir string) (*TagsStore, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &TagsStore{db: db}, nil
}

// SetTagsForPage 删除页面原标签后逐项插入传入标签，重复唯一键不改变已有值。
// 本方法及服务同名方法均不转小写或裁去空白，正文提取路径单独执行这些规则。
// 普通连接自行提交事务，外部执行器的提交和回滚仍归其所有者管理。
func (s *TagsStore) SetTagsForPage(pageID string, tags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := database.BeginExecutor(s.queryExecutor())
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM page_tags WHERE page_id = ?`, pageID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to clear tags for page %s: %w", pageID, err)
	}

	if len(tags) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO page_tags(page_id, tag) VALUES (?, ?) ON DUPLICATE KEY UPDATE tag=page_tags.tag`)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to prepare tag insert: %w", err)
		}
		defer shared.LogClose(stmt.Close, "could not close statement")

		for _, tag := range tags {
			if _, err := stmt.Exec(pageID, tag); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("failed to insert tag %q for page %s: %w", tag, pageID, err)
			}
		}
	}

	return tx.Commit()
}

// DeleteTagsForPage 只删除标签关系，保留该页在 page_meta 中的摘要。
func (s *TagsStore) DeleteTagsForPage(pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`DELETE FROM page_tags WHERE page_id = ?`, pageID)
	return err
}

// SetPageIndex 在同一执行器中替换页面标签并写入摘要。
// 标签原样存储，摘要无论是否为空都会写入 page_meta；外部事务由外层最终提交。
func (s *TagsStore) SetPageIndex(pageID string, tags []string, excerpt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := database.BeginExecutor(s.queryExecutor())
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM page_tags WHERE page_id = ?`, pageID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to clear tags for page %s: %w", pageID, err)
	}

	if len(tags) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO page_tags(page_id, tag) VALUES (?, ?) ON DUPLICATE KEY UPDATE tag=page_tags.tag`)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to prepare tag insert: %w", err)
		}
		defer shared.LogClose(stmt.Close, "could not close statement")

		for _, tag := range tags {
			if _, err := stmt.Exec(pageID, tag); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("failed to insert tag %q for page %s: %w", tag, pageID, err)
			}
		}
	}

	// 已存在页面元数据时只覆盖摘要字段，没有记录时插入一条。
	if _, err := tx.Exec(
		`INSERT INTO page_meta(page_id, excerpt) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE excerpt = ?`,
		pageID, excerpt, excerpt,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to upsert excerpt for page %s: %w", pageID, err)
	}

	return tx.Commit()
}

// DeletePageIndex 依次删除单页标签和摘要，两步共享一个事务或外部执行器。
// 普通事务任一步失败均尝试回滚，并把原执行错误交给调用方。
func (s *TagsStore) DeletePageIndex(pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := database.BeginExecutor(s.queryExecutor())
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM page_tags WHERE page_id = ?`, pageID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM page_meta WHERE page_id = ?`, pageID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// DeletePageIndexes 用一组预编译语句逐页清理标签与摘要，空输入直接成功。
// 所有页面共享一个事务，循环失败立即返回，延迟回滚释放未提交的普通事务。
// 提交后的回滚结果被忽略，绑定外部执行器时这些提交与回滚调用不结束外层事务。
func (s *TagsStore) DeletePageIndexes(pageIDs []string) error {
	if len(pageIDs) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := database.BeginExecutor(s.queryExecutor())
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	tagsStmt, err := tx.Prepare(`DELETE FROM page_tags WHERE page_id = ?`)
	if err != nil {
		return err
	}
	defer shared.LogClose(tagsStmt.Close, "could not close statement")

	metaStmt, err := tx.Prepare(`DELETE FROM page_meta WHERE page_id = ?`)
	if err != nil {
		return err
	}
	defer shared.LogClose(metaStmt.Close, "could not close statement")

	for _, pageID := range pageIDs {
		if _, err := tagsStmt.Exec(pageID); err != nil {
			return fmt.Errorf("failed to delete tags for page %s: %w", pageID, err)
		}
		if _, err := metaStmt.Exec(pageID); err != nil {
			return fmt.Errorf("failed to delete meta for page %s: %w", pageID, err)
		}
	}

	return tx.Commit()
}

// GetExcerptsForPages 通过参数化 IN 查询批量读取摘要，以页面 ID 建立映射。
// 空输入返回空映射，没有元数据记录的页面不生成映射项。
func (s *TagsStore) GetExcerptsForPages(pageIDs []string) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(pageIDs) == 0 {
		return map[string]string{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(pageIDs)), ",")
	args := make([]any, len(pageIDs))
	for i, id := range pageIDs {
		args[i] = id
	}

	rows, err := s.queryExecutor().Query(fmt.Sprintf(
		`SELECT page_id, excerpt FROM page_meta WHERE page_id IN (%s)`,
		placeholders,
	), args...)
	if err != nil {
		return nil, err
	}
	defer shared.LogClose(rows.Close, logCloseRowsFailed)

	result := make(map[string]string)
	for rows.Next() {
		var pageID, excerpt string
		if err := rows.Scan(&pageID, &excerpt); err != nil {
			return nil, err
		}
		result[pageID] = excerpt
	}
	return result, rows.Err()
}

// Clear 在同一事务或外部执行器中清空标签表与摘要表，不删除原始页面。
func (s *TagsStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := database.BeginExecutor(s.queryExecutor())
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM page_tags`); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM page_meta`); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// GetAllTags 对字面前缀匹配的标签统计不同页面数量。
// 结果按数量降序、标签升序排列，limit 大于零时才添加行数限制。
func (s *TagsStore) GetAllTags(filter string, limit int) ([]TagCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		SELECT tag, COUNT(DISTINCT page_id) AS count
		FROM page_tags
		WHERE tag LIKE CONCAT(?, '%') ESCAPE '='
		GROUP BY tag
		ORDER BY count DESC, tag ASC
	`
	if limit > 0 {
		query += fmt.Sprintf(sqlLimitFmt, limit)
	}

	rows, err := s.queryExecutor().Query(query, escapeLikePrefix(filter))
	if err != nil {
		return nil, err
	}
	defer shared.LogClose(rows.Close, logCloseRowsFailed)

	var result []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		result = append(result, tc)
	}
	return result, rows.Err()
}

// GetAllTagsForSelection 找出同时拥有全部已选标签的页面，再统计这些页面的其他标签。
// 候选还需满足字面前缀且不能属于已选标签，数量仅统计当前匹配页面集合。
// selected 为空时转入已持锁的普通查询；非空输入不去重，重复标签会影响计数条件。
func (s *TagsStore) GetAllTagsForSelection(filter string, selected []string, limit int) ([]TagCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(selected) == 0 {
		return s.getAllTagsLocked(filter, limit)
	}

	// 参数依次对应已选标签、所需不同标签数、前缀和需要排除的已选标签。
	filterArg := escapeLikePrefix(filter)
	selectionPlaceholders := strings.TrimRight(strings.Repeat("?,", len(selected)), ",")
	args := make([]any, 0, len(selected)*2+2)
	for _, tag := range selected {
		args = append(args, tag)
	}
	args = append(args, len(selected), filterArg)
	for _, tag := range selected {
		args = append(args, tag)
	}

	query := fmt.Sprintf(`
		WITH matching_pages AS (
			SELECT page_id
			FROM page_tags
			WHERE tag IN (%s)
			GROUP BY page_id
			HAVING COUNT(DISTINCT tag) = ?
		)
		SELECT pt.tag, COUNT(DISTINCT pt.page_id) AS count
		FROM page_tags pt
		JOIN matching_pages mp ON mp.page_id = pt.page_id
		WHERE pt.tag LIKE CONCAT(?, '%%') ESCAPE '='
		  AND pt.tag NOT IN (%s)
		GROUP BY pt.tag
		ORDER BY count DESC, tag ASC
	`, selectionPlaceholders, selectionPlaceholders)
	if limit > 0 {
		query += fmt.Sprintf(sqlLimitFmt, limit)
	}

	rows, err := s.queryExecutor().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer shared.LogClose(rows.Close, logCloseRowsFailed)

	var result []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		result = append(result, tc)
	}
	return result, rows.Err()
}

// GetPageIDsByTags 用分组计数实现标签交集，空标签输入返回 nil。
// 比较不同标签数量与输入长度，因此调用方须避免重复标签；结果未额外排序。
func (s *TagsStore) GetPageIDsByTags(tags []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(tags) == 0 {
		return nil, nil
	}

	args := make([]any, 0, len(tags)+1)
	for _, t := range tags {
		args = append(args, t)
	}
	args = append(args, len(tags))

	placeholders := strings.TrimRight(strings.Repeat("?,", len(tags)), ",")
	rows, err := s.queryExecutor().Query(fmt.Sprintf(`
		SELECT page_id
		FROM page_tags
		WHERE tag IN (%s)
		GROUP BY page_id
		HAVING COUNT(DISTINCT tag) = ?
	`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer shared.LogClose(rows.Close, logCloseRowsFailed)

	var pageIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		pageIDs = append(pageIDs, id)
	}
	return pageIDs, rows.Err()
}

// getAllTagsLocked 在调用方已持锁时执行普通标签统计，避免再次获取同一互斥锁。
func (s *TagsStore) getAllTagsLocked(filter string, limit int) ([]TagCount, error) {
	query := `
		SELECT tag, COUNT(DISTINCT page_id) AS count
		FROM page_tags
		WHERE tag LIKE CONCAT(?, '%') ESCAPE '='
		GROUP BY tag
		ORDER BY count DESC, tag ASC
	`
	if limit > 0 {
		query += fmt.Sprintf(sqlLimitFmt, limit)
	}

	rows, err := s.queryExecutor().Query(query, escapeLikePrefix(filter))
	if err != nil {
		return nil, err
	}
	defer shared.LogClose(rows.Close, logCloseRowsFailed)

	var result []TagCount
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		result = append(result, tc)
	}
	return result, rows.Err()
}

// GetTagsForPages 批量读取标签并按页面 ID 分组，各页标签按数据库升序结果追加。
// 空输入返回空映射，没有标签记录的页面不会出现在映射中。
func (s *TagsStore) GetTagsForPages(pageIDs []string) (map[string][]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(pageIDs) == 0 {
		return map[string][]string{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(pageIDs)), ",")
	args := make([]any, len(pageIDs))
	for i, id := range pageIDs {
		args[i] = id
	}

	rows, err := s.queryExecutor().Query(fmt.Sprintf(`
		SELECT page_id, tag FROM page_tags
		WHERE page_id IN (%s)
		ORDER BY page_id, tag ASC
	`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer shared.LogClose(rows.Close, logCloseRowsFailed)

	result := make(map[string][]string)
	for rows.Next() {
		var pageID, tag string
		if err := rows.Scan(&pageID, &tag); err != nil {
			return nil, err
		}
		result[pageID] = append(result[pageID], tag)
	}
	return result, rows.Err()
}

// escapeLikePrefix 配合 ESCAPE '=' 依次转义等号、百分号与下划线。
// 反斜杠不在此处执行替换，百分号与下划线输入按字面字符参与前缀查询。
func escapeLikePrefix(s string) string {
	s = strings.ReplaceAll(s, `=`, `==`)
	s = strings.ReplaceAll(s, `%`, `=%`)
	s = strings.ReplaceAll(s, `_`, `=_`)
	return s
}

// Close 只关闭本实例拥有的连接，成功后置空，外部执行器由其所有者释放。
func (s *TagsStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			return err
		}
		s.db = nil
	}
	return nil
}

// WithExecutor 创建使用外部执行器的独立视图，不复制连接或转移连接所有权。
func (s *TagsStore) WithExecutor(executor database.Executor) *TagsStore {
	return &TagsStore{executor: executor}
}

// queryExecutor 优先采用外部执行器，否则返回本实例连接，不自行检查关闭状态。
func (s *TagsStore) queryExecutor() database.Executor {
	if s.executor != nil {
		return s.executor
	}
	return s.db
}
