// 本文件定义页面出链的存储表示和接口表示，解析状态保存在存储记录中。
package links

// Outgoing 描述来源页面的一个目标引用，未解析目标的 ToPageID 可为空。
type Outgoing struct {
	FromPageID string
	ToPageID   string
	FromTitle  string
	ToPath     string    // 路由路径或带 wikilink: 前缀的标题标记，不保证目标存在。
	State      LinkState // 已解析、失效或标题歧义状态。
}

// OutgoingResult 汇总转换后的出链列表及其条目数。
type OutgoingResult struct {
	Outgoings []OutgoingResultItem `json:"outgoings"`
	Count     int                  `json:"count"`
}

// OutgoingResultItem 向接口提供目标显示信息，Broken 只对应 LinkBroken。
// 标题歧义不表示失效，目标标题在有可读目标页面时由当前树填入。
type OutgoingResultItem struct {
	ToPageID    string `json:"to_page_id"`
	ToPageTitle string `json:"to_page_title"`
	ToPath      string `json:"to_path"`
	Broken      bool   `json:"broken"`
	FromPageID  string `json:"from_page_id"`
}
