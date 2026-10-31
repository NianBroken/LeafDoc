// 本文件定义字段校验失败的响应结构，保存字段名称和对应的提示文本。
package errors

// FieldError 描述一个字段的一条错误，JSON 字段名与接口响应约定一致。
type FieldError struct {
	// Field 是调用方提供的字段标识，本类型不校验该字段是否真实存在。
	Field string `json:"field"`
	// Message 是已经组织好的提示文字，序列化时不进行翻译或格式化。
	Message string `json:"message"`
}

// NewFieldError 原样保存字段名和提示，返回独立的错误条目。
func NewFieldError(field, message string) *FieldError {
	return &FieldError{Field: field, Message: message}
}

// ValidationErrors 按添加顺序收集多条字段错误，并实现 error 接口。
// 详细内容由 Errors 提供，Error 的返回值不拼接各字段的提示。
type ValidationErrors struct {
	// Errors 以 fields 键输出，不在本类型中按字段去重。
	Errors []*FieldError `json:"fields"`
}

// NewValidationErrors 创建非 nil 的空切片，使空集合序列化为数组而非 null。
// 该保证针对本构造函数的结果，直接使用结构体零值时 Errors 仍为 nil。
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{Errors: []*FieldError{}}
}

// Add 在尾部加入一条错误，允许同一字段保留多条不同或相同的提示。
func (v *ValidationErrors) Add(field, message string) {
	v.Errors = append(v.Errors, NewFieldError(field, message))
}

// Error 返回固定分类文本，不泄露具体字段内容，也不判断集合是否为空。
func (v *ValidationErrors) Error() string {
	return "validation error"
}

// HasErrors 仅依据当前条目数量判断是否存在字段错误。
// 条目内容是否为空不参与判断，即使调用方直接放入 nil 条目也会计入数量。
func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}
