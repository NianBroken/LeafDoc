// Package properties 从页面前置字段提取可检索属性，并维护 MySQL 属性索引。
// 服务负责正文提取规则，存储层提供整页替换、按键值查询和批量读取。
package properties

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// PropertyEntry 表示一个属性值及其存储类型。
// 正文提取路径只产生 text 类型，直接存储接口原样接收调用方传入的数据。
type PropertyEntry struct {
	Value string
	Type  string // 正文提取结果为 text，此字段不在存储层执行类型校验。
}

// PropertyKeyCount 将属性键与拥有该键的不同页面数量组合为接口结果。
type PropertyKeyCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

const logCloseRowsFailed = "could not close rows"

// PropertiesStore 串行访问属性表，并允许为单次业务事务创建执行器视图。
// 每个视图有独立互斥锁，外部事务的并发使用规则仍由调用方负责。
type PropertiesStore struct {
	executor database.Executor
	mu       sync.Mutex
	db       *sql.DB
}

// NewPropertiesStore 打开配置对应的 MySQL 连接，打开失败不创建存储对象。
func NewPropertiesStore(storageDir string) (*PropertiesStore, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &PropertiesStore{db: db}, nil
}

// SetPropertiesForPage 先删除页面原属性，再逐项插入传入键值和类型。
// 本方法不筛除系统键、不转换类型，PropertiesService 的同名方法也只做转交。
// 普通连接创建并提交事务；绑定外部执行器时借用执行器，最终提交或回滚由外层负责。
func (s *PropertiesStore) SetPropertiesForPage(pageID string, props map[string]PropertyEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := database.BeginExecutor(s.queryExecutor())
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM page_properties WHERE page_id = ?`, pageID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to clear properties for page %s: %w", pageID, err)
	}

	// 空集合只执行删除。非空集合复用预编译语句，任一插入失败立即返回。
	if len(props) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO page_properties(page_id, property_key, property_value, value_type) VALUES (?, ?, ?, ?)`)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to prepare property insert: %w", err)
		}
		defer shared.LogClose(stmt.Close, "could not close statement")

		for k, e := range props {
			if _, err := stmt.Exec(pageID, k, e.Value, e.Type); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("failed to insert property %q for page %s: %w", k, pageID, err)
			}
		}
	}

	return tx.Commit()
}

// DeletePropertiesForPage 删除页面的全部属性，页面不存在时不单独报错。
func (s *PropertiesStore) DeletePropertiesForPage(pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`DELETE FROM page_properties WHERE page_id = ?`, pageID)
	return err
}

// Clear 清空整张属性索引表，不删除页面正文。
func (s *PropertiesStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`DELETE FROM page_properties`)
	return err
}

// GetAllPropertyKeys 按字面前缀筛选属性键，统计每个键关联的不同页面数。
// 结果按数量降序、键名升序排列，仅在 limit 大于零时限制行数。
func (s *PropertiesStore) GetAllPropertyKeys(filter string, limit int) ([]PropertyKeyCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		SELECT property_key, COUNT(DISTINCT page_id) AS count
		FROM page_properties
		WHERE property_key LIKE CONCAT(?, '%') ESCAPE '='
		GROUP BY property_key
		ORDER BY count DESC, property_key ASC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.queryExecutor().Query(query, escapeLikePrefix(filter))
	if err != nil {
		return nil, err
	}
	defer shared.LogClose(rows.Close, logCloseRowsFailed)

	var result []PropertyKeyCount
	for rows.Next() {
		var kc PropertyKeyCount
		if err := rows.Scan(&kc.Key, &kc.Count); err != nil {
			return nil, err
		}
		result = append(result, kc)
	}
	return result, rows.Err()
}

// GetPageIDsByProperty 用 SQL 等值条件匹配键和值，返回对应页面 ID。
// 字符串比较规则由数据库列的排序规则执行，此处不添加排序或类型条件。
func (s *PropertiesStore) GetPageIDsByProperty(key, value string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows, err := s.queryExecutor().Query(`
		SELECT page_id FROM page_properties
		WHERE property_key = ? AND property_value = ?
	`, key, value)
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

// GetPropertiesForPages 批量读取指定页面的属性，按页面 ID 和属性键组织两层映射。
// 输入为空时返回空映射；没有属性记录的页面不会单独出现在结果中。
func (s *PropertiesStore) GetPropertiesForPages(pageIDs []string) (map[string]map[string]PropertyEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(pageIDs) == 0 {
		return map[string]map[string]PropertyEntry{}, nil
	}

	// 仅占位符数量参与 SQL 拼接，实际页面 ID 全部通过参数传入。
	placeholders := strings.TrimRight(strings.Repeat("?,", len(pageIDs)), ",")
	args := make([]any, len(pageIDs))
	for i, id := range pageIDs {
		args[i] = id
	}

	rows, err := s.queryExecutor().Query(fmt.Sprintf(`
		SELECT page_id, property_key, property_value, value_type FROM page_properties
		WHERE page_id IN (%s)
		ORDER BY page_id, property_key ASC
	`, placeholders), args...)
	if err != nil {
		return nil, err
	}
	defer shared.LogClose(rows.Close, logCloseRowsFailed)

	result := make(map[string]map[string]PropertyEntry)
	for rows.Next() {
		var pageID, key, value, typ string
		if err := rows.Scan(&pageID, &key, &value, &typ); err != nil {
			return nil, err
		}
		if result[pageID] == nil {
			result[pageID] = make(map[string]PropertyEntry)
		}
		result[pageID][key] = PropertyEntry{Value: value, Type: typ}
	}
	return result, rows.Err()
}

// Close 只关闭本对象持有的数据库连接，外部执行器的资源由调用方释放。
func (s *PropertiesStore) Close() error {
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

// escapeLikePrefix 配合 SQL 的 ESCAPE '='，依次转义等号、百分号和下划线。
// 反斜杠在此转义约定下无需处理，输入中的通配符不会扩大前缀匹配范围。
func escapeLikePrefix(s string) string {
	s = strings.ReplaceAll(s, `=`, `==`)
	s = strings.ReplaceAll(s, `%`, `=%`)
	s = strings.ReplaceAll(s, `_`, `=_`)
	return s
}

// WithExecutor 创建仅绑定外部执行器的视图，不转移原连接所有权。
func (s *PropertiesStore) WithExecutor(executor database.Executor) *PropertiesStore {
	return &PropertiesStore{executor: executor}
}

// queryExecutor 优先返回外部执行器，未绑定时返回本对象的连接。
// 本方法不检查连接是否已关闭，调用方须遵守存储对象生命周期。
func (s *PropertiesStore) queryExecutor() database.Executor {
	if s.executor != nil {
		return s.executor
	}
	return s.db
}
