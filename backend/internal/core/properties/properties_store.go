package properties

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// PropertyEntry is a single property value with its type.
type PropertyEntry struct {
	Value string
	Type  string // currently always "text"
}

// PropertyKeyCount carries a property key and the number of pages that have it.
type PropertyKeyCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

const logCloseRowsFailed = "could not close rows"

type PropertiesStore struct {
	executor database.Executor
	mu       sync.Mutex
	db       *sql.DB
}

// NewPropertiesStore connects the domain store to the configured MySQL database.
func NewPropertiesStore(storageDir string) (*PropertiesStore, error) {
	db, err := database.Open(storageDir)
	if err != nil {
		return nil, err
	}
	return &PropertiesStore{db: db}, nil
}

// SetPropertiesForPage replaces all properties for the given page atomically.
// Keys and values are stored as-is — filtering of reserved keys (tags, title, leafdoc_*)
// and type coercion are the caller's responsibility. All write paths go through
// PropertiesService.SetPropertiesForPage or ExtractPropertiesFromContent, which enforce this.
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

func (s *PropertiesStore) DeletePropertiesForPage(pageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`DELETE FROM page_properties WHERE page_id = ?`, pageID)
	return err
}

func (s *PropertiesStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.queryExecutor().Exec(`DELETE FROM page_properties`)
	return err
}

// GetAllPropertyKeys returns distinct property keys with page count, optionally filtered by
// prefix. limit <= 0 means no limit.
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

// GetPageIDsByProperty returns page IDs where key = key AND value = value (exact match).
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

// GetPropertiesForPages returns a map of pageID → properties for the given page IDs.
func (s *PropertiesStore) GetPropertiesForPages(pageIDs []string) (map[string]map[string]PropertyEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(pageIDs) == 0 {
		return map[string]map[string]PropertyEntry{}, nil
	}

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

// escapeLikePrefix escapes LIKE special characters so that '%', '_', and '\'
// are treated as literals and not SQL wildcards.
func escapeLikePrefix(s string) string {
	s = strings.ReplaceAll(s, `=`, `==`)
	s = strings.ReplaceAll(s, `%`, `=%`)
	s = strings.ReplaceAll(s, `_`, `=_`)
	return s
}

// WithExecutor binds an independent store view to the caller-owned transaction.
func (s *PropertiesStore) WithExecutor(executor database.Executor) *PropertiesStore {
	return &PropertiesStore{executor: executor}
}

func (s *PropertiesStore) queryExecutor() database.Executor {
	if s.executor != nil {
		return s.executor
	}
	return s.db
}
