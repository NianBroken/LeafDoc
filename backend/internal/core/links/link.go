// 链接模块提取页面内部引用并维护目标解析状态，这些结构按方向与失效状态组织接口结果。
package links

// LinkStatusResult 按方向和失效状态分组返回链接，Counts 对应各分组的统计信息。
type LinkStatusResult struct {
	Backlinks       []BacklinkResultItem `json:"backlinks"`
	BrokenIncoming  []BacklinkResultItem `json:"broken_incoming"`
	Outgoings       []OutgoingResultItem `json:"outgoings"`
	BrokenOutgoings []OutgoingResultItem `json:"broken_outgoings"`
	Counts          LinkStatusCounts     `json:"counts"`
}

// LinkStatusCounts 只保存各类数量，不包含具体链接记录。
type LinkStatusCounts struct {
	Backlinks       int `json:"backlinks"`
	BrokenIncoming  int `json:"broken_incoming"`
	Outgoings       int `json:"outgoings"`
	BrokenOutgoings int `json:"broken_outgoings"`
}
