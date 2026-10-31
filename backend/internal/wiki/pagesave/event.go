package pagesave

import "github.com/NianBroken/LeafDoc/internal/core/tree"

// PageOperationType 标识触发保存事件的页面操作，处理器按此值选择具体分支。
type PageOperationType string

const (
	PageOperationCreate  PageOperationType = "create"
	PageOperationUpdate  PageOperationType = "update"
	PageOperationMove    PageOperationType = "move"
	PageOperationDelete  PageOperationType = "delete"
	PageOperationRestore PageOperationType = "restore"
)

// PageSaveEvent 在页面操作与派生处理之间传递页面状态和变化标记。
// 字段由调用方按实际操作填写，此结构不复制页面指针，也不自动判断哪些字段变化。
type PageSaveEvent struct {
	Operation PageOperationType
	UserID    string

	// Before 供需要旧状态的处理分支使用，创建事件不要求提供。
	Before *tree.Page
	// After 供正文、标签和属性索引等使用，删除事件不要求提供。
	After *tree.Page

	// 变化标记控制内容历史、结构历史和路径链接处理，不在处理器中统一重新比较。
	ContentChanged bool
	SlugChanged    bool
	TitleChanged   bool

	// OldPath 提前保存操作前路由，因为页面指针关联的节点在移动后会计算出新路径。
	OldPath string
	// OldTitle 为标题变化时的原文字，供按标题解析的链接重新匹配。
	OldTitle string

	// AffectedPages 提供操作涉及的页面列表，例如移动子树或删除子树。
	// 单页处理器也能直接读取 After，并非所有事件都必须填入此列表。
	AffectedPages []*tree.Page

	// Summary 传给正文历史记录入口，结构历史分支不使用此文字。
	Summary string
}
