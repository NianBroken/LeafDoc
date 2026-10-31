// 本文件提供单页写入适配和批量删除，实际正文转换由 IndexPages 统一执行。
package search

import (
	"github.com/NianBroken/LeafDoc/internal/core/database"

	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// IndexPage 把单页输入交给批量入口，优先返回事务错误，再返回该页提取失败。
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

// RemovePages 用预编译语句逐页删除索引，空输入直接成功，不要求页面原来存在。
// 普通连接开启并提交事务，绑定执行器时借用外部事务，提交归外层所有者负责。
// 延迟回滚和语句关闭的错误在这里不单独返回。
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
