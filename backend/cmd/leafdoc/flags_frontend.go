// 本文件声明界面展示、资源上传和修订参数，启动入口将它们分发给路由与 Wiki。
package main

import (
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/defaults"

	"github.com/urfave/cli/v3"
)

// frontendOptions 保存界面选项、功能开关与修订保留策略。
// 解析只填写字段，具体服务和路由读取这些值后决定实际功能行为。
type frontendOptions struct {
	defaultLanguage         string
	injectCodeInHeader      string
	customStylesheet        string
	hideLinkMetadataSection bool
	maxAssetUploadSize      string
	enableRevision          bool
	enableLinkRefactor      bool
	enableAPIKeyManagement  bool
	maxRevisionHistory      int
	revisionCoalesceWindow  time.Duration
}

// Flags 声明环境来源与默认值，上传大小连同默认值接受字节解析校验，
// 时长由 cli 库解析，布尔环境值支持公共来源定义的文字形式。
func (o *frontendOptions) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name: "default-language",
			// 此处提供启动选项，路由还会读取全局持久化语言偏好。
			Destination: &o.defaultLanguage,
			Value:       defaults.Language,
			Category:    catFrontend,
			Usage:       "default UI language code (e.g. en, de); ignored unless the frontend ships it",
			Sources:     envVars("LEAFDOC_DEFAULT_LANGUAGE"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "inject-code-in-header",
			// 内容作为可信头部片段传入模板，此处不清洗 HTML 或脚本。
			Destination: &o.injectCodeInHeader,
			Category:    catFrontend,
			Usage:       "raw HTML/JS injected into <head> (analytics, custom CSS); trusted code only, it is not sanitized",
			Sources:     envVars("LEAFDOC_INJECT_CODE_IN_HEADER"),
			Config:      trimmed,
		},
		&cli.StringFlag{
			Name: "custom-stylesheet",
			// 路由按数据目录解析并提供 custom.css，此处仅裁剪空白。
			Destination: &o.customStylesheet,
			Category:    catFrontend,
			Usage:       "a .css file inside the data dir, served as /custom.css (under --base-path when set)",
			Sources:     envVars("LEAFDOC_CUSTOM_STYLESHEET"),
			Config:      trimmed,
		},
		&cli.BoolFlag{
			Name:        "hide-link-metadata-section",
			Destination: &o.hideLinkMetadataSection,
			Category:    catFrontend,
			Usage:       "hide the link metadata section in the frontend UI",
			Sources:     envBoolVars("LEAFDOC_HIDE_LINK_METADATA_SECTION"),
		},
		&cli.StringFlag{
			Name:             "max-asset-upload-size",
			Destination:      &o.maxAssetUploadSize,
			Category:         catFrontend,
			Usage:            "maximum size for asset uploads (e.g. 50MiB, 50MB, 52428800)",
			Value:            "50MiB",
			Sources:          envVars("LEAFDOC_MAX_ASSET_UPLOAD_SIZE"),
			Validator:        validateByteSizeValue,
			ValidateDefaults: true,
			Config:           trimmed,
		},
		&cli.BoolFlag{
			Name: "enable-revision",
			// 历史服务与前端功能开关共用此值，默认未启用。
			Destination: &o.enableRevision,
			Category:    catFeatures,
			Usage:       "enable the revision / page history feature",
			Sources:     envBoolVars("LEAFDOC_ENABLE_REVISION"),
		},
		&cli.BoolFlag{
			Name:        "enable-link-refactor",
			Destination: &o.enableLinkRefactor,
			Category:    catFeatures,
			Usage:       "enable the link refactoring dialog and rewrite flow",
			Sources:     envBoolVars("LEAFDOC_ENABLE_LINK_REFACTOR"),
		},
		&cli.BoolFlag{
			Name:        "enable-api-key-management",
			Destination: &o.enableAPIKeyManagement,
			Category:    catFeatures,
			Usage:       "enable the experimental API key management feature",
			Sources:     envBoolVars("LEAFDOC_ENABLE_API_KEY_MANAGEMENT"),
		},
		&cli.IntFlag{
			Name: "max-revision-history",
			// 此上限与同一作者短时间保存的合并窗口一起传给修订服务。
			Destination: &o.maxRevisionHistory,
			Category:    catRevisions,
			Usage:       "maximum revisions kept per page; 0 = unlimited",
			Value:       100,
			Sources:     envVars("LEAFDOC_MAX_REVISION_HISTORY"),
		},
		&cli.DurationFlag{
			Name:        "revision-coalesce-window",
			Destination: &o.revisionCoalesceWindow,
			Category:    catRevisions,
			Usage:       "coalesce rapid successive saves by the same author (e.g. 5m); 0 = disabled",
			Value:       5 * time.Minute,
			Sources:     envVars("LEAFDOC_REVISION_COALESCE_WINDOW"),
			DefaultText: "5m",
		},
	}
}
