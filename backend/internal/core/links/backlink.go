// 本文件定义从其他页面指向目标页面的入链记录及接口结果。
package links

// Backlink 保存来源页面、目标页面、来源标题和失效标志，不直接携带当前来源路径。
type Backlink struct {
	FromPageID string
	ToPageID   string
	FromTitle  string
	Broken     bool
}

// BacklinkResult 将入链条目与转换后条目数量组合返回。
type BacklinkResult struct {
	Backlinks []BacklinkResultItem `json:"backlinks"`
	Count     int                  `json:"count"`
}

// BacklinkResultItem 包含接口所需来源路径，路径由结果转换时查询当前页面树获得。
type BacklinkResultItem struct {
	FromPageID string `json:"from_page_id"`
	FromTitle  string `json:"from_title"`
	FromPath   string `json:"from_path"`
	Broken     bool   `json:"broken"`
	ToPageID   string `json:"to_page_id"`
}
