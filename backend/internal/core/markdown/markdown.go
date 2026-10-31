// 本文件提供带路径的 Markdown 内存对象，将正文、系统元信息和写盘操作联系起来。
package markdown

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/shared"
)

// MarkdownFile 保存目标路径、去掉页首元信息后的正文及解析出的字段。
// 设置方法只改变内存状态，只有 WriteToFile 执行磁盘写入。
type MarkdownFile struct {
	path    string
	content string
	fm      Frontmatter
}

// LoadMarkdownFile 检查扩展名后读取整个文件，再将原文交给内存构造函数解析。
// 扩展名按大小写不敏感方式识别 .md，读取和解析错误直接返回，不创建部分对象。
func LoadMarkdownFile(filePath string) (*MarkdownFile, error) {
	if !strings.EqualFold(filepath.Ext(filePath), ".md") {
		return nil, errors.New("file is not a markdown file")
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return NewMarkdownFileFromRaw(filePath, string(raw))
}

// NewMarkdownFileFromRaw 解析已读取的字符串，不再次访问文件系统或校验路径扩展名。
// 无元信息时保留完整正文，有解析错误时返回 nil，路径仅保存供标题回退和写盘使用。
func NewMarkdownFileFromRaw(filePath string, raw string) (*MarkdownFile, error) {
	fm, content, has, err := ParseFrontmatter(raw)
	if err != nil {
		return nil, err
	}
	if !has {
		fm = Frontmatter{}
	}

	return &MarkdownFile{
		path:    filePath,
		content: content,
		fm:      fm,
	}, nil
}

// NewMarkdownFile 直接保存调用方给出的路径、正文和字段，不执行解析或校验。
// Frontmatter 按值复制，其中 ExtraFields 的 map 仍与传入对象共享底层数据。
func NewMarkdownFile(filePath string, content string, fm Frontmatter) *MarkdownFile {
	return &MarkdownFile{
		path:    filePath,
		content: content,
		fm:      fm,
	}
}

// WriteToFile 组装当前字段和正文，再使用同目录临时文件替换目标。
// 目标存在且 Stat 成功时沿用它的模式，否则使用 0644，父目录不会在此创建。
// 元信息编码失败时不会开始文件写入，写入失败原样交给调用方。
func (mf *MarkdownFile) WriteToFile() error {
	fmContent, err := BuildMarkdownWithFrontmatter(mf.fm, mf.content)
	if err != nil {
		return err
	}

	mode := os.FileMode(0o644)
	if st, err := os.Stat(mf.path); err == nil {
		mode = st.Mode()
	}

	return shared.WriteFileAtomic(mf.path, []byte(fmContent), mode)
}

// GetTitle 依次使用非空系统标题、正文中的首个一级标题、去掉扩展名的文件名。
// 系统标题判断在去空白之前，因此仅含空白的标题会返回空串而不继续回退。
// 当前实现的所有返回分支均返回 nil 错误。
func (mf *MarkdownFile) GetTitle() (string, error) {
	if mf.fm.LeafDocTitle != "" {
		return strings.TrimSpace(mf.fm.LeafDocTitle), nil
	}

	title, err := mf.extractTitleFromFirstHeading()
	if err == nil && title != "" {
		return title, nil
	}

	base := path.Base(strings.ReplaceAll(mf.path, `\`, "/"))
	name := strings.TrimSuffix(base, path.Ext(base))
	return name, nil
}

// extractTitleFromFirstHeading 按行查找去掉两端空白后以 # 加空格开头的文本。
// 不解析 Markdown 语法树，因此不会主动排除代码块内符合该文本形式的行。
func (mf *MarkdownFile) extractTitleFromFirstHeading() (string, error) {
	lines := strings.Split(mf.content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# ")), nil
		}
	}
	return "", errors.New("no heading found")
}

// GetContent 返回当前正文，不包含由本对象单独保存的页首元信息。
func (mf *MarkdownFile) GetContent() string {
	return mf.content
}

// SetContent 直接替换正文，不解析传入文本中的元信息，也不改变已有字段。
func (mf *MarkdownFile) SetContent(content string) {
	mf.content = content
}

// SetExtraFields 直接保存 map 引用，调用方随后修改同一 map 会反映到对象中。
func (mf *MarkdownFile) SetExtraFields(fields map[string]interface{}) {
	mf.fm.ExtraFields = fields
}

// SetRawContentPreservingManagedFrontmatter 解析输入后只接收正文与附加字段。
// 输入中的系统字段不会覆盖对象已有系统元信息；没有元信息时只改正文，附加字段也保留。
// 解析失败在任何赋值之前返回，原内存状态不变。
func (mf *MarkdownFile) SetRawContentPreservingManagedFrontmatter(raw string) error {
	incomingFM, body, has, err := ParseFrontmatter(raw)
	if err != nil {
		return err
	}

	if !has {
		mf.content = raw
		return nil
	}

	mf.content = body
	mf.fm.ExtraFields = incomingFM.ExtraFields
	return nil
}

// GetPath 返回构造时保存的路径，不进行绝对路径转换或存在性检查。
func (mf *MarkdownFile) GetPath() string {
	return mf.path
}

// GetFrontmatter 返回结构副本，ExtraFields 仍共享 map，不能据此认为获得了深拷贝。
func (mf *MarkdownFile) GetFrontmatter() Frontmatter {
	return mf.fm
}

// setFrontmatterID 原样保存系统 ID，不生成或校验标识。
func (mf *MarkdownFile) setFrontmatterID(id string) {
	mf.fm.LeafDocID = id
}

// setFrontmatterTitle 原样保存系统标题，写盘时的空白处理由组装函数完成。
func (mf *MarkdownFile) setFrontmatterTitle(title string) {
	mf.fm.LeafDocTitle = title
}

// SetLeafDocFrontmatter 同时保存 ID 和标题，不改变作者、时间、置顶与附加字段。
func (mf *MarkdownFile) SetLeafDocFrontmatter(id string, title string) {
	mf.setFrontmatterID(id)
	mf.setFrontmatterTitle(title)
}

// SetLeafDocMetadata 去掉四个输入的两端空白后保存，不解析日期或验证用户是否存在。
func (mf *MarkdownFile) SetLeafDocMetadata(createdAt string, updatedAt string, creatorID string, lastAuthorID string) {
	mf.fm.LeafDocCreatedAt = strings.TrimSpace(createdAt)
	mf.fm.LeafDocUpdatedAt = strings.TrimSpace(updatedAt)
	mf.fm.LeafDocCreatorID = strings.TrimSpace(creatorID)
	mf.fm.LeafDocLastAuthorID = strings.TrimSpace(lastAuthorID)
}

// SetLeafDocPinned 保存置顶布尔值，具体排序和展示由页面模块处理。
func (mf *MarkdownFile) SetLeafDocPinned(pinned bool) {
	mf.fm.LeafDocPinned = pinned
}
