// 本文件定义链接重构的索引候选和重写规则，不执行正文解析或数据库操作。
package links

// RefactorLinkMatch 保存引用来源及目标路径，Broken 只表示索引中的失效状态。
// 来源路径由上层结合页面树读取，这里只携带来源 ID 和索引标题。
type RefactorLinkMatch struct {
	FromPageID string
	FromTitle  string
	ToPath     string
	Broken     bool
}

// RewriteRule 将旧路径映射到新路径，并可提供标题更名信息。
// 各重写入口按自己的匹配规则使用这些字段，结构本身不验证路径或标题。
type RewriteRule struct {
	OldPath  string
	NewPath  string
	OldTitle string // 非空时供双链标题更名匹配使用，与路径规则分别处理。
	NewTitle string
}
