// 本文件根据数据库中的教程状态记录，在空实例中建立内置教程页面。
package wiki

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/defaultdocs"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
	wikipages "github.com/NianBroken/LeafDoc/internal/wiki/pages"
)

// 初始化状态键与默认教程包保持一致，记录位于 instance_state 命名空间。
const defaultDocumentsStateFile = defaultdocs.StateFile

// defaultDocumentsState 通过完成标记区分尚未结束的初始化与已经完成的实例。
// 完成标记存在后不会因用户删除教程而重新建立页面。
type defaultDocumentsState struct {
	Complete bool `json:"complete"`
}

// EnsureDefaultDocuments 为尚未初始化的空实例建立教程，并允许未完成状态继续写入。
// 没有状态记录且已有页面时只写入完成标记，不向现有实例插入教程。
// 已有未完成记录时按教程路径复用系统创建的根页面，非系统页面占用路径则返回错误。
func (w *Wiki) EnsureDefaultDocuments() error {
	stateKey := defaultDocumentsStateFile
	state := defaultDocumentsState{}
	data, err := database.ReadRecord(w.storageDir, "instance_state", stateKey)
	// 读取成功但 JSON 不合法时直接报错，不把损坏状态当作首次启动。
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &state); err != nil {
			return fmt.Errorf("read default document state: %w", err)
		}
		if state.Complete {
			return nil
		}
	case !errors.Is(err, sql.ErrNoRows):
		return err
	default:
		// 只有记录不存在才进入首次初始化判断，其他数据库错误已在上一分支返回。
		if w.tree.HasPages() {
			return database.WriteRecord(w.storageDir, "instance_state", stateKey, []byte("{\"complete\":true}\n"))
		}
		if err := database.WriteRecord(w.storageDir, "instance_state", stateKey, []byte("{\"complete\":false}\n")); err != nil {
			return err
		}
	}
	// 状态准备完毕后读取嵌入教程，不修改 Markdown 正文或教程排列顺序。
	documents, err := defaultdocs.Load()
	if err != nil {
		return err
	}
	kind := tree.NodeKindPage
	orchestrator := w.newPageOrchestrator()
	// 每篇教程依次定位或创建页面，再使用当前版本更新正文，任一步失败即结束。
	for _, document := range documents {
		var id string
		// 查找仅覆盖根节点直接子项，按 slug 判断是否已有同路径教程。
		for _, node := range w.tree.GetTree().Children {
			if node.Slug == document.Slug {
				if node.Metadata.CreatorID != SYSTEM_USER_ID {
					return fmt.Errorf("default document path is occupied: %s", document.Slug)
				}
				id = node.ID
				break
			}
		}
		if id == "" {
			out, err := wikipages.NewCreatePageUseCase(w.tree, w.slug, orchestrator, w.log, w.metrics).Execute(context.Background(), wikipages.CreatePageInput{
				UserID: SYSTEM_USER_ID, Title: document.Title, Slug: document.Slug, Kind: &kind,
			})
			if err != nil {
				return err
			}
			id = out.Page.ID
		}
		// 创建与复用都读取当前页面版本，更新通过既有业务用例执行保存及相关联动。
		page, err := w.tree.GetPage(id)
		if err != nil {
			return err
		}
		if _, err := wikipages.NewUpdatePageUseCase(w.tree, w.slug, orchestrator, w.log, w.metrics).Execute(context.Background(), wikipages.UpdatePageInput{
			UserID: SYSTEM_USER_ID, ID: id, Version: page.Version(), Title: document.Title, Slug: document.Slug, Content: &document.Content, Kind: &kind,
		}); err != nil {
			return err
		}
	}
	// 所有页面成功处理后才写完成状态，失败保留未完成标记供下次调用继续。
	return database.WriteRecord(w.storageDir, "instance_state", stateKey, []byte("{\"complete\":true}\n"))
}
