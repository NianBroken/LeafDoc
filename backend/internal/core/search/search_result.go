// 本文件定义搜索响应、单页结果和标签统计的接口结构。
package search

// SearchResult 同时返回分页参数、匹配总数和当前页条目，标签统计由上层查询流程填充。
type SearchResult struct {
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
	Count     int                `json:"count"`
	Items     []SearchResultItem `json:"items"`
	TagFacets []SearchTagFacet   `json:"tag_facets"`
}

// SearchResultItem 保存页面身份、排序分数及已转义并可带 b 高亮标签的标题和摘要。
// Tags 为空时不输出该字段，不能从字段缺失推断页面正文没有标签。
type SearchResultItem struct {
	PageID  string   `json:"page_id"`
	Title   string   `json:"title"`
	Path    string   `json:"path"`
	Kind    string   `json:"kind"`
	Rank    float64  `json:"rank"`
	Excerpt string   `json:"excerpt"`
	Tags    []string `json:"tags,omitempty"`
}

// SearchTagFacet 表示标签与当前查询范围中的页面计数。
type SearchTagFacet struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}
