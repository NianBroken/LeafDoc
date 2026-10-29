package search

import (
	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

func (s *MySQLIndex) IndexPage(path string, filePath string, pageID string, title string, kind tree.NodeKind, raw string) error {
	failures, err := s.IndexPages([]IndexPageInput{{Path: path, FilePath: filePath, PageID: pageID, Title: title, Kind: kind, Raw: raw}})
	if err != nil {
		return err
	}
	if len(failures) > 0 {
		return failures[0].Err
	}
	return nil
}

// RemovePages removes page IDs from the index in a single transaction.
func (s *MySQLIndex) RemovePages(pageIDs []string) error {
	if len(pageIDs) == 0 {
		return nil
	}
	return s.withExecutor(func(db database.Executor) error {
		tx, err := database.BeginExecutor(db)
		if err != nil {
			return err
		}
		defer func() {
			_ = tx.Rollback()
		}()

		stmt, err := tx.Prepare(`DELETE FROM search_documents WHERE pageID = ?`)
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
