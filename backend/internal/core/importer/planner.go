package importer

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/ignore"
	"github.com/NianBroken/LeafDoc/internal/core/markdown"
	"github.com/NianBroken/LeafDoc/internal/core/shared"
	"github.com/NianBroken/LeafDoc/internal/core/tree"
)

// PlanAction 表示计划项的处理选择，执行器只直接处理创建和跳过两种值。
type PlanAction string

const (
	PlanActionCreate PlanAction = "create" // 建立目标路径并写入导入正文。
	PlanActionUpdate PlanAction = "update" // 声明更新选择，当前执行器将此值归入未知动作。
	PlanActionSkip   PlanAction = "skip"   // 保留已有目标，不写入源文件正文。
)

// ImportMDFile 标识导入来源中的一个 Markdown 文件，文件内容在分析和执行时读取。
type ImportMDFile struct {
	SourcePath string // 相对于解压目录的源文件路径，用于定位磁盘文件和解析相对链接。
}

// PlanItem 保存一个源文件的页面映射、现有目标状态和计划动作。
// Notes 记录标题读取等非致命问题，Conflicts 是结果中的冲突列表字段。
type PlanItem struct {
	SourcePath  string        `json:"source_path"`
	TargetPath  string        `json:"target_path"`
	Title       string        `json:"title"`
	DesiredSlug string        `json:"desired_slug"`
	Kind        tree.NodeKind `json:"kind"`
	Exists      bool          `json:"exists"`
	ExistingID  *string       `json:"existing_id"`

	Action    PlanAction `json:"action"`
	Conflicts []string   `json:"conflicts"`
	Notes     []string   `json:"notes"`
}

// PlanOptions 指定读取源文件的目录和目标页面树中的基础路径。
type PlanOptions struct {
	SourceBasePath string // 导入源根目录，SourcePath 在此目录下解释。
	TargetBasePath string // 目标路径的公共前缀，不是本机磁盘目录。
}

// PlanResult 保存可执行的计划项与分析失败信息，并记录生成计划时的页面树摘要。
// 单项分析错误进入 Errors，不使整份计划直接返回错误。
type PlanResult struct {
	ID       string     `json:"id"`
	TreeHash string     `json:"tree_hash"` // 计划生成时的页面树摘要，用于执行前检查结构变化。
	Items    []PlanItem `json:"items"`
	Errors   []string   `json:"errors"`
}

// Planner 读取源文件元信息，将文件目录映射到页面路径并查询目标是否存在。
// 规划阶段不创建页面，也不为同级页面执行唯一名称分配。
type Planner struct {
	log         *slog.Logger
	wiki        ImporterWiki
	slugger     *tree.SlugService
	storageDir  string
	ignoreCache *ignore.Cache
	// loadMarkdown 提供标题读取入口，空值使用实际 Markdown 文件加载函数。
	// 文件存在性与非目录检查仍先于此入口执行，测试替身不会绕过该检查。
	loadMarkdown func(string) (*markdown.MarkdownFile, error)
}

// NewPlanner 绑定页面接口、路径名称生成器和存储根目录，忽略缓存由独立方法设置。
func NewPlanner(wiki ImporterWiki, slugger *tree.SlugService, storageDir string) *Planner {
	return &Planner{
		log:        slog.Default().With("component", "Planner"),
		wiki:       wiki,
		slugger:    slugger,
		storageDir: storageDir,
	}
}

// SetIgnoreCache 设置目标路径过滤使用的忽略规则缓存，传入空值表示不查询规则。
func (p *Planner) SetIgnoreCache(ignoreCache *ignore.Cache) {
	p.ignoreCache = ignoreCache
}

// getIgnoreForDir 获取指定目录对应的已编译规则，未配置缓存时返回空值。
func (p *Planner) getIgnoreForDir(dir string) *ignore.IgnoreFile {
	if p.ignoreCache == nil {
		return nil
	}
	return p.ignoreCache.Get(dir)
}

// CreatePlan 生成计划标识并按输入顺序分析每个文件，收集未被忽略的结果。
// 单项失败会记录告警及错误文字后继续处理，只有计划标识生成失败直接返回错误。
func (p *Planner) CreatePlan(entries []ImportMDFile, options PlanOptions) (*PlanResult, error) {
	// 此标识属于整份导入计划，不是目标页面的标识。
	id, err := shared.GenerateUniqueID()
	if err != nil {
		return nil, fmt.Errorf("could not generate unique ID: %w", err)
	}
	result := &PlanResult{
		ID:       id,
		Items:    []PlanItem{},
		Errors:   []string{},
		TreeHash: p.wiki.TreeHash(),
	}
	for _, entry := range entries {
		resEntry, err := p.analyzeEntry(entry, options)
		if err != nil {
			p.log.Warn("could not import resource", "source_path", entry.SourcePath, "error", err)
			result.Errors = append(result.Errors, err.Error())
			continue
		}

		// 按目标页面的父目录读取忽略规则，目录类型由 index.md 映射结果确定。
		// 命中忽略规则的项不进入 Items，也不计入 Errors。
		targetRel := filepath.ToSlash(resEntry.TargetPath)
		isDir := resEntry.Kind == tree.NodeKindSection
		dir := filepath.Join(p.storageDir, "root", filepath.Dir(targetRel))
		if ig := p.getIgnoreForDir(dir); ig != nil {
			if ig.Matches(targetRel, isDir) {
				p.log.Debug("skipping import entry matching .leafdocignore", "target", targetRel, "source", entry.SourcePath)
				continue
			}
		}

		result.Items = append(result.Items, *resEntry)
	}
	return result, nil
}

// analyzeEntry 验证单个源文件，推导页面类型、路径、标题及默认动作。
// 目录形式输入被拒绝；index.md 映射为所属目录的分区，其余文件映射为页面。
func (p *Planner) analyzeEntry(mdFile ImportMDFile, options PlanOptions) (*PlanItem, error) {
	// 磁盘读取路径与后面的页面路由分别计算，二者使用各自的路径规则。
	sourcePath := filepath.Join(options.SourceBasePath, filepath.FromSlash(mdFile.SourcePath))

	// Stat 失败直接返回，目录也被拒绝，标题读取之前必须通过这项检查。
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("source path is a directory, expected a file: " + mdFile.SourcePath)
	}

	// 源路径转换为斜杠形式，再拆分目录和文件名以构造页面路由。
	rel := filepath.ToSlash(strings.TrimSpace(mdFile.SourcePath))
	rel = strings.TrimPrefix(rel, "/")

	filenameLower := strings.ToLower(path.Base(rel))
	sourceDir := path.Dir(rel)
	if sourceDir == "." {
		sourceDir = ""
	}

	// 每个源目录段先转换为有效路由段，转换失败使该文件的分析失败。
	// 此阶段不检查同级名称唯一性，实际页面建立由执行器调用 Wiki 接口完成。
	normalizedSourceDir, err := p.slugger.NormalizePathToValidSlugs(sourceDir)
	if err != nil {
		return nil, err
	}
	normalizedSourceDir = strings.Trim(normalizedSourceDir, "/")

	// 目标基础路径去掉两端空白和斜杠后，与源文件的目录映射组合。
	targetBase := strings.Trim(strings.TrimSpace(options.TargetBasePath), "/")

	kind := tree.NodeKindPage
	var wikiPath string

	if filenameLower == "index.md" {
		kind = tree.NodeKindSection
		wikiPath = strings.Trim(path.Join(targetBase, normalizedSourceDir), "/")
	} else {
		// 文件名先带扩展名转换，再去掉扩展名，得到页面最后一级路径段。
		normalizedFilename, err := p.slugger.NormalizeFilenameToValidSlug(filenameLower)
		if err != nil {
			return nil, err
		}
		baseSlug := strings.TrimSuffix(normalizedFilename, path.Ext(normalizedFilename))
		wikiPath = strings.Trim(path.Join(targetBase, normalizedSourceDir, baseSlug), "/")
	}

	// 目标存在性来自页面路径查询，不根据同名磁盘文件作判断。
	result, err := p.wiki.LookupPagePath(wikiPath)
	if err != nil {
		return nil, err
	}

	// 标题读取失败记录在备注中，路径已通过验证的文件仍能形成计划项。
	var notes []string
	loader := p.loadMarkdown
	if loader == nil {
		loader = markdown.LoadMarkdownFile
	}
	md, err := loader(sourcePath)
	if err != nil {
		notes = append(notes, fmt.Sprintf("Failed to load markdown file for title extraction: %v", err))
	}

	// 没有可读取的 Markdown 对象时，标题使用目标路径末段。
	title := path.Base(wikiPath)
	if wikiPath == "" {
		// 空目标路径使用源文件名去除扩展名后的文字，仍为空时使用 root。
		title = strings.TrimSuffix(filenameLower, path.Ext(filenameLower))
		if title == "" {
			title = "root"
		}
	}

	if md != nil {
		var titleErr error
		title, titleErr = md.GetTitle()
		if titleErr != nil {
			notes = append(notes, fmt.Sprintf("Failed to extract title from file: %v", titleErr))
			title = "unknown" // 标题提取失败时写入明确的回退文字，不沿用路径标题。
		}
	}

	if !result.Exists {
		// 目标尚不存在时默认创建，期望路径段取目标路由末段。
		slug := ""
		if wikiPath != "" {
			segs := strings.Split(wikiPath, "/")
			slug = segs[len(segs)-1]
		}

		return &PlanItem{
			SourcePath:  mdFile.SourcePath,
			TargetPath:  wikiPath,
			Title:       title,
			DesiredSlug: slug,
			Kind:        kind,
			Exists:      false,
			Action:      PlanActionCreate,
			Notes:       notes,
		}, nil
	}

	if len(result.Segments) == 0 {
		return nil, errors.New("invalid lookup result with zero segments for existing path")
	}

	// 目标已存在时默认跳过，保留查询得到的页面标识和实际路径段。
	last := result.Segments[len(result.Segments)-1]
	return &PlanItem{
		SourcePath:  mdFile.SourcePath,
		TargetPath:  wikiPath,
		Title:       title,
		DesiredSlug: last.Slug,
		Exists:      true,
		ExistingID:  last.ID,
		Kind:        kind,
		Action:      PlanActionSkip,
		Notes:       notes,
	}, nil
}
