// 本文件定义路径逐段查找结果，供创建路径前展示已存在与待创建的部分。
package tree

// PathSegment 描述一个路径片段，节点不存在时可省略类型、标题与 ID。
// 指针字段区分缺失与对应类型的零值，Exists 单独表达查找结果。
type PathSegment struct {
	Slug   string    `json:"slug"`
	Exists bool      `json:"exists"`
	Kind   *NodeKind `json:"kind,omitempty"`
	Title  *string   `json:"title,omitempty"`
	ID     *string   `json:"id,omitempty"`
}

// PathLookup 组合完整路径、逐段结果、整体存在状态和创建许可判断。
// 结构本身只传递计算结果，不执行创建操作。
type PathLookup struct {
	Path      string        `json:"path"`
	Segments  []PathSegment `json:"segments"`
	Exists    bool          `json:"exists"`
	CanCreate bool          `json:"canCreate"`
}
