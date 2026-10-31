// 本文件声明页首元信息解析的分类错误，供调用方通过 errors.Is 判断。
package markdown

import "errors"

// ErrFrontmatterParse 与底层 YAML 错误共同组成错误链，不保存具体文件路径或字段。
var ErrFrontmatterParse = errors.New("frontmatter parse error")
