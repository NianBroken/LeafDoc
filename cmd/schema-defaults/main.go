// Command schema-defaults exports the authoritative schema and initial values.
// Run from the repository root. -check reports drift without rewriting files.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/auth"
	"github.com/NianBroken/LeafDoc/internal/core/branding"
	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/defaults"
	"github.com/NianBroken/LeafDoc/internal/core/snapshot"
	"golang.org/x/crypto/bcrypt"
)

var tableNames = regexp.MustCompile(`CREATE TABLE IF NOT EXISTS ([a-z_]+)`)
var savedHash = regexp.MustCompile(`\$2[aby]\$[0-9]{2}\$[./A-Za-z0-9]{53}`)

// Descriptions state actual relationships; the schema intentionally has no SQL foreign keys.
var descriptions = map[string]string{
	"content_records":   "业务文档和设置。namespace 与 record_key 组成主键，payload 为 UTF-8 JSON 或正文二进制。settings 使用稳定设置键。page_nodes 使用页面 ID，包含正文、父节点、排序和元信息。revisions、revision_content、revision_manifests 保存历史和索引。instance_state 保存教程初始化标记。",
	"auth_state":        "认证代次。会话验证使用 generation 使恢复前的认证状态失效。",
	"schema_migrations": "已应用的结构版本及应用时间。",
	"users":             "用户账号。用户名和邮箱分别唯一。password 仅保存 bcrypt 散列，role 使用应用角色定义。",
	"sessions":          "访问和刷新会话。user_id 在应用层关联 users.id，时间采用 Unix 秒，revoked_at 为空表示未撤销。",
	"api_keys":          "API 密钥元数据及散列。user_id、created_by 关联 users.id，prefix 唯一，密钥明文不存储。",
	"email_tokens":      "邮件操作令牌散列。user_id 关联 users.id，时间采用 Unix 秒，consumed_at 为空表示未消费。",
	"favorites":         "用户收藏，user_id 关联 users.id，page_id 关联 page_nodes 记录，复合主键防止重复收藏。",
	"links":             "页面出入链接派生索引。from_page_id 和 to_page_id 关联页面 ID，未解析目标为 NULL，to_path 保存目标路径。",
	"page_tags":         "页面标签，page_id 关联页面 ID，复合主键防止同一页面重复标签。",
	"page_meta":         "页面摘要，page_id 关联页面 ID。",
	"page_properties":   "页面属性，page_id 关联页面 ID，property_key 保持精确大小写，value_type 默认为 text。",
	"search_documents":  "可重建的页面全文搜索索引，pageID 关联页面 ID，ngram 解析器支持中文搜索。",
}

// Field meanings complement the exact type, nullability, keys and defaults in each DDL.
var fieldDescriptions = map[string]string{
	"namespace": "业务命名空间", "record_key": "命名空间内唯一记录键", "payload": "完整业务记录的 JSON 或二进制正文", "id": "记录唯一标识", "generation": "随机认证代次标识", "version": "结构迁移版本号", "applied_at": "迁移应用时间，数据库当前 UTC 时间", "username": "唯一登录用户名", "password": "bcrypt 密码散列", "email": "唯一账号邮箱", "role": "权限角色 admin、editor 或 viewer", "created_at": "记录创建时间，日期列使用 UTC，BIGINT 列使用 Unix 秒", "must_set_password": "是否必须先设置密码", "must_change_password": "是否必须先更改初始密码", "user_id": "所属用户 ID，应用层关联 users.id", "token_type": "访问或刷新令牌类型", "expires_at": "失效时刻，Unix 秒；仅允许 NULL 的 API 密钥字段以 NULL 表示不设到期时间", "revoked_at": "撤销时刻，Unix 秒，NULL 表示未撤销", "name": "用户填写的显示名称", "prefix": "公开密钥前缀，唯一", "key_hash": "API 密钥散列", "created_by": "创建者用户 ID，应用层关联 users.id", "last_used_at": "最后使用时刻，Unix 秒，NULL 表示未使用", "token_hash": "邮件令牌散列", "purpose": "邮件令牌用途", "consumed_at": "消费时刻，Unix 秒，NULL 表示未消费", "page_id": "页面 ID，应用层关联 page_nodes 记录", "from_page_id": "来源页面 ID", "to_page_id": "目标页面 ID，NULL 表示目标尚未解析", "to_path": "目标页面路径", "from_title": "来源页面标题", "state": "链接状态枚举，0 为已解析，1 为链接失效，2 为目标不唯一", "tag": "标签文本", "excerpt": "派生页面摘要", "property_key": "属性名称", "property_value": "属性值的文本表示", "value_type": "属性类型，默认 text", "pageID": "页面 ID，与 page_nodes 记录关联", "path": "页面访问路径", "filepath": "页面在应用存储层的逻辑路径", "kind": "页面或章节类型", "title": "页面标题", "headings": "派生标题检索文本", "content": "派生正文检索文本",
}

func describeFields(out *strings.Builder, statement string) {
	lines := strings.Split(statement, "\n")
	if len(lines) == 1 { // Compact definitions use the same column matcher as multiline definitions.
		statement = strings.ReplaceAll(statement, " (", " (\n")
		statement = strings.ReplaceAll(statement, ", ", ",\n")
		lines = strings.Split(statement, "\n")
	}
	for _, line := range lines {
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) < 2 {
			continue
		}
		if description, ok := fieldDescriptions[parts[0]]; ok {
			fmt.Fprintf(out, "--   %s：%s。完整定义见 DDL。\n", parts[0], description)
		}
	}
}

func quote(v string) string {
	return "'" + strings.ReplaceAll(strings.ReplaceAll(v, "\\", "\\\\"), "'", "''") + "'"
}
func mustJSON(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func main() {
	check := flag.Bool("check", false, "verify generated files without writing")
	flag.Parse()
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		panic("run schema-defaults from the repository root")
	}
	target := filepath.Join(root, "internal/core/database/default.sql")
	existing, err := os.ReadFile(target)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	hash := savedHash.Find(existing)
	if bcrypt.CompareHashAndPassword(hash, []byte(defaults.AdminPassword)) != nil {
		hash, err = bcrypt.GenerateFromPassword([]byte(defaults.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			panic(err)
		}
	}
	var out strings.Builder
	out.WriteString("-- LeafDoc MySQL 8.0 完整结构和初始化默认值。\n-- 生成来源：node scripts/container.mjs schema-write，核验：node scripts/container.mjs schema。\n-- 在选定的空业务数据库执行，不包含 CREATE DATABASE、DROP 或服务器连接凭据。\n-- 未声明 DEFAULT 的非空业务字段必须由应用提供，不以空字符串冒充业务默认值。\n-- NULL 表示明确的未发生或未解析状态，时间戳精度和字段长度均以以下 DDL 为准。\n-- 全部逻辑关系由应用维护，当前无数据库外键、视图、存储过程、触发器或事件。\n-- schema_migrations 未显式声明字符集，继承目标数据库设置；其字段均为数字和时间，不存储文本。\n-- content_records 的 settings 记录键为下方五个 JSON 文件名，payload 是配置对象。\n-- page_nodes 的键为页面 ID，payload 包含 node 元信息、parentId 和 raw 原始 Markdown，子节点与父引用不重复序列化。\n-- revisions 的键为 页面ID:版本ID，revision_content 的键为 页面ID:正文SHA256，revision_manifests 的键为清单SHA256。\n-- 历史附件内容保存在实例目录中，数据库保存历史清单关联，不将文件系统内容虚构为 SQL 表。\n-- instance_state 保存教程初始化状态，complete 为布尔值；初始化完成后不会在每次启动时重复建立教程。\nSET NAMES utf8mb4;\nSET time_zone = '+00:00';\n\n")
	for _, statement := range database.SchemaStatements() {
		if match := tableNames.FindStringSubmatch(statement); match != nil {
			fmt.Fprintf(&out, "-- %s：%s\n", match[1], descriptions[match[1]])
			describeFields(&out, statement)
		}
		out.WriteString(statement + ";\n\n")
	}
	settings := []struct {
		key   string
		value any
	}{
		{"preferences.json", map[string]any{"language": defaults.Language, "dateFormat": defaults.DateFormat, "timeFormat": defaults.TimeFormat}},
		{"public-access.json", map[string]any{"enabled": defaults.PublicAccess}},
		{"toc-display.json", map[string]any{"alwaysShow": false, "mode": defaults.TocMode}},
		{"snapshot-settings.json", snapshot.DefaultSettings(24*time.Hour, 10)},
		{"branding.json", branding.DefaultBrandingConfig()},
	}
	out.WriteString("-- 设置仅在缺失时插入，不覆盖管理员已保存的设置。\n")
	for _, setting := range settings {
		fmt.Fprintf(&out, "INSERT INTO content_records(namespace, record_key, payload) VALUES('settings', %s, %s) ON DUPLICATE KEY UPDATE record_key=content_records.record_key;\n", quote(setting.key), quote(mustJSON(setting.value)))
	}
	out.WriteString("\n-- 初始账号仅在尚无管理员时建立，ID 为 SQL 引导记录标识，密码仅存储 bcrypt。\n")
	fmt.Fprintf(&out, "INSERT INTO users(id, username, password, email, role, must_set_password, must_change_password) SELECT UUID(), %s, %s, %s, %s, FALSE, FALSE WHERE NOT EXISTS (SELECT 1 FROM users WHERE role=%s);\n", quote(defaults.AdminUsername), quote(string(hash)), quote(defaults.AdminEmail), quote(auth.RoleAdmin), quote(auth.RoleAdmin))
	out.WriteString("-- 教程、随机认证代次、页面 ID 和运行密钥由应用在首次启动时生成，不使用固定伪造值。\n")
	browser := map[string]any{"language": defaults.Language, "dateFormat": defaults.DateFormat, "timeFormat": defaults.TimeFormat, "tocMode": defaults.TocMode, "publicAccess": defaults.PublicAccess}
	browserJSON, err := json.MarshalIndent(browser, "", "  ")
	if err != nil {
		panic(err)
	}
	browserJSON = append(browserJSON, '\n')
	outputs := []struct {
		path string
		data []byte
	}{{target, []byte(out.String())}, {filepath.Join(root, "ui/src/shared/lib/instanceDefaults.json"), browserJSON}}
	for _, output := range outputs {
		previous, err := os.ReadFile(output.path)
		if err != nil && !os.IsNotExist(err) {
			panic(err)
		}
		if *check {
			if !bytes.Equal(previous, output.data) {
				panic("generated file is stale: " + output.path)
			}
		} else if err := os.WriteFile(output.path, output.data, 0644); err != nil {
			panic(err)
		}
	}
	fmt.Println("默认 SQL 和浏览器默认值一致性核验通过")
}
