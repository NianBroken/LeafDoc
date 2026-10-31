package importer

import "os"

// ZipWorkspace 保存压缩包解压目录，供导入计划扫描文件和读取附件。
// 目录的生存期由调用方管理，导入结束后通过 Cleanup 删除其中的临时内容。
type ZipWorkspace struct {
	// Root 是需要整体清理的目录路径，空路径表示没有可清理的工作区。
	Root string
}

// Cleanup 删除工作区及其全部子目录和文件，并将删除错误交给调用方。
// 空接收者和空路径直接返回，避免对尚未建立的工作区执行文件操作。
func (ws *ZipWorkspace) Cleanup() error {
	if ws == nil || ws.Root == "" {
		return nil
	}
	return os.RemoveAll(ws.Root)
}
