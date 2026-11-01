// 安装资源独立嵌入，不依赖主应用构建产物或数据库。
package main

import (
	_ "embed"
	"strings"
)

//go:embed setup/page.html
var setupHTML string

//go:embed setup/page.css
var setupCSS string

//go:embed setup/page.js
var setupJS string

var installationPage = strings.NewReplacer("__STYLE__", setupCSS, "__SCRIPT__", setupJS).Replace(setupHTML)
