// 本文件集中声明页面树错误及带上下文的错误结构，供业务层按错误链判断失败类型。
package tree

import (
	"errors"
	"fmt"
)

// 固定错误用于区分查找、结构约束、排序、漂移和版本冲突，定义处不执行任何检查。
var ErrPageNotFound = errors.New("page not found")
var ErrParentNotFound = errors.New("parent not found")
var ErrTreeNotLoaded = errors.New("tree not loaded")
var ErrPageHasChildren = errors.New("page has children")
var ErrPageAlreadyExists = errors.New("page already exists")
var ErrMovePageCircularReference = errors.New("circular reference detected")
var ErrPageCannotBeMovedToItself = errors.New("page cannot be moved to itself")
var ErrInvalidSortOrder = errors.New("invalid sort order")
var ErrFileNotFound = errors.New("file not found")
var ErrDrift = errors.New("drift detected")
var ErrInvalidOperation = errors.New("invalid operation")
var ErrConvertNotAllowed = errors.New("convert not allowed")
var ErrVersionConflict = errors.New("version conflict")
var ErrVersionRequired = errors.New("version required")

// VersionUnchecked 是内部操作显式跳过版本比较的特殊值，不等同于缺少版本的空字符串。
const VersionUnchecked = "\x00"

// DriftError 携带发生漂移的节点、类型、路径和原因，Error 输出全部这些字段。
type DriftError struct {
	NodeID string
	Kind   NodeKind
	Path   string
	Reason string
}

// Error 构造诊断文本，不改变结构字段或底层状态。
func (e *DriftError) Error() string {
	return "drift detected: nodeID=" + e.NodeID + ", kind=" + string(e.Kind) + ", path=" + e.Path + ", reason=" + e.Reason
}

// Unwrap 使 errors.Is 能把具体漂移错误识别为 ErrDrift。
func (e *DriftError) Unwrap() error {
	return ErrDrift
}

// InvalidOpError 记录操作名与拒绝原因，并通过错误链归类为 ErrInvalidOperation。
type InvalidOpError struct {
	Op     string
	Reason string
}

func (e *InvalidOpError) Error() string { return fmt.Sprintf("%s: %s", e.Op, e.Reason) }
func (e *InvalidOpError) Unwrap() error { return ErrInvalidOperation }

// PageAlreadyExistsError 记录发生目标冲突的路径，创建、移动或重命名流程可使用该结构。
type PageAlreadyExistsError struct {
	Path string
}

func (e *PageAlreadyExistsError) Error() string { return fmt.Sprintf("already exists: %s", e.Path) }
func (e *PageAlreadyExistsError) Unwrap() error { return ErrPageAlreadyExists }

// NotFoundError 保存资源类型、ID 和路径，Error 只输出资源类型与 ID。
// Unwrap 统一返回 ErrPageNotFound，Path 保留给结构化处理而不进入当前错误文本。
type NotFoundError struct {
	Resource string
	ID       string
	Path     string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

func (e *NotFoundError) Unwrap() error {
	return ErrPageNotFound
}

// ConvertNotAllowedError 说明不能执行的源类型、目标类型及原因，归类为类型转换拒绝。
type ConvertNotAllowedError struct {
	From   NodeKind
	To     NodeKind
	Reason string
}

func (e *ConvertNotAllowedError) Error() string {
	return fmt.Sprintf("cannot convert from %s to %s: %s", e.From, e.To, e.Reason)
}

func (e *ConvertNotAllowedError) Unwrap() error {
	return ErrConvertNotAllowed
}
