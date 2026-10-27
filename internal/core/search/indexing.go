package search

import (
	"database/sql"

	"github.com/perber/wiki/internal/core/tree"

	_ "modernc.org/sqlite"
)

func (s *SQLiteIndex) IndexPage(path string, filePath string, pageID string, title string, kind tree.NodeKind, raw string) error {
	failures, err := s.IndexPages([]IndexPageInput{{Path: path, FilePath: filePath, PageID: pageID, Title: title, Kind: kind, Raw: raw}})
	if err != nil {
		return err
	}
	if len(failures) > 0 {
		return failures[0].Err
	}
	return nil
}

// RemovePages removes multiple pages from the index in a single
// transaction — the batch counterpart to RemovePage, used for subtree
// deletes for the same reason IndexPages exists (one commit instead of
// one per page).
func (s *SQLiteIndex) RemovePages(pageIDs []string) error {
	if len(pageIDs) == 0 {
		return nil
	}
	return s.withDB(func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()

		stmt, err := tx.Prepare(`DELETE FROM pages WHERE pageID = ?`)
		if err != nil {
			return err
		}
		defer func() {
			_ = stmt.Close()
		}()

		for _, pageID := range pageIDs {
			if _, err := stmt.Exec(pageID); err != nil {
				return err
			}
		}

		return tx.Commit()
	})
}

func (s *SQLiteIndex) RemovePage(pageID string) error {
	return s.withDB(func(db *sql.DB) error {
		_, err := db.Exec(`DELETE FROM pages WHERE pageID = ?`, pageID)
		return err
	})
}

func (s *SQLiteIndex) RemovePageByFilePath(filePath string) (int64, error) {
	var rows int64
	err := s.withDB(func(db *sql.DB) error {
		res, err := db.Exec(`DELETE FROM pages WHERE filepath = ?`, filePath)
		if err != nil {
			return err
		}
		r, err := res.RowsAffected()
		if err != nil {
			return err
		}
		rows = r
		return nil
	})
	return rows, err
}
