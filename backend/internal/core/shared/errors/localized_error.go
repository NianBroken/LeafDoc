// 本文件在普通错误之外保存翻译标识、模板和参数，供上层组织本地化响应。
package errors

import (
	stderrors "errors"
	"fmt"
)

// LocalizedError 将展示信息与底层错误保存在同一对象中。
// 本类型不执行翻译，Error 只使用 Message 和 Cause 组织文本。
type LocalizedError struct {
	// Code 是上层用于识别错误类别的稳定标识。
	Code string
	// Message 是 Error 使用的提示正文。
	Message string
	// Template 和 Args 保存模板及参数，具体替换规则由消费它们的代码决定。
	Template string
	Args     []string
	// Cause 保留被包装错误，使标准错误链查询能够继续向下查找。
	Cause error
}

// Error 对 nil 接收者返回空文本，有底层错误时以冒号连接正文与底层错误文本。
func (e *LocalizedError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap 暴露底层错误，nil 接收者或没有底层错误时返回 nil。
func (e *LocalizedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// NewLocalizedError 保存所有描述字段并复制参数切片。
// 调用方随后替换原切片中的元素不会改变该错误保存的参数。
func NewLocalizedError(code, message, template string, cause error, args ...string) *LocalizedError {
	return &LocalizedError{
		Code:     code,
		Message:  message,
		Template: template,
		Args:     append([]string(nil), args...),
		Cause:    cause,
	}
}

// AsLocalizedError 使用标准 errors.As 沿错误链查找可赋给此类型的对象。
// 没有匹配对象时返回 nil 和 false，不根据错误文本猜测类别。
func AsLocalizedError(err error) (*LocalizedError, bool) {
	var localized *LocalizedError
	if !stderrors.As(err, &localized) {
		return nil, false
	}
	return localized, true
}
