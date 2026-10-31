// 本文件检查参数和环境变量的命名对应关系，以及布尔开关前缀与例外清单。
package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// legacyBoolFlagNames 列出允许不使用统一前缀的四个现有布尔参数。
// 测试同时要求清单项确实存在且为布尔类型，避免无效例外残留。
var legacyBoolFlagNames = []string{
	"hide-link-metadata-section",
	"public-access",
	"smtp-insecure-skip-verify",
	"snapshot",
}

// boolFlagPrefixes 定义常规布尔开关的方向前缀，不适用于其他参数类型。
var boolFlagPrefixes = []string{"enable-", "disable-", "allow-"}

// TestServerFlags_EnvVarIsLeafdocPlusFlagName 要求每个参数恰好公开一个环境变量，
// 名称等于 LEAFDOC_ 加参数名的大写下划线形式，并检查帮助生成接口可读取该名称。
func TestServerFlags_EnvVarIsLeafdocPlusFlagName(t *testing.T) {
	for _, flag := range (&serverConfig{}).flags() {
		name := flag.Names()[0]
		t.Run(name, func(t *testing.T) {
			envFlag, ok := flag.(cli.DocGenerationFlag)
			if !ok {
				t.Fatalf("flag --%s (%T) does not expose its environment variables", name, flag)
			}
			want := "LEAFDOC_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
			if got := envFlag.GetEnvVars(); !slices.Equal(got, []string{want}) {
				t.Errorf("flag --%s env vars = %v, want [%s]", name, got, want)
			}
		})
	}
}

// TestServerFlags_BoolFlagsUseOptInOrOptOutPrefix 只检查布尔参数，
// 既拒绝没有方向前缀且不在例外清单的名称，也拒绝已合规却仍在清单的名称。
func TestServerFlags_BoolFlagsUseOptInOrOptOutPrefix(t *testing.T) {
	for _, flag := range (&serverConfig{}).flags() {
		if _, isBool := flag.(*cli.BoolFlag); !isBool {
			continue
		}
		name := flag.Names()[0]
		t.Run(name, func(t *testing.T) {
			hasPrefix := slices.ContainsFunc(boolFlagPrefixes, func(p string) bool {
				return strings.HasPrefix(name, p)
			})
			isLegacy := slices.Contains(legacyBoolFlagNames, name)
			switch {
			case !hasPrefix && !isLegacy:
				t.Errorf("bool flag --%s must start with one of %v (conventions.md#naming)", name, boolFlagPrefixes)
			case hasPrefix && isLegacy:
				t.Errorf("bool flag --%s follows the scheme now; remove it from legacyBoolFlagNames", name)
			}
		})
	}
}

// TestLegacyBoolFlagNames_NameExistingBoolFlags 建立全部参数的类型映射，
// 逐项验证例外清单没有列入不存在的参数或非布尔参数。
func TestLegacyBoolFlagNames_NameExistingBoolFlags(t *testing.T) {
	isBool := map[string]bool{}
	for _, flag := range (&serverConfig{}).flags() {
		_, isBool[flag.Names()[0]] = flag.(*cli.BoolFlag)
	}
	for _, legacy := range legacyBoolFlagNames {
		isBoolFlag, exists := isBool[legacy]
		switch {
		case !exists:
			t.Errorf("legacyBoolFlagNames lists --%s, which no longer exists; remove it", legacy)
		case !isBoolFlag:
			t.Errorf("legacyBoolFlagNames lists --%s, which is not a bool flag; remove it", legacy)
		}
	}
}
