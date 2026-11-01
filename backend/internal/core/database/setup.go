// 安装数据库操作绕过普通 Open 的自动建表行为，检查阶段只读取目标库。
package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/NianBroken/LeafDoc/internal/core/installation"
	"github.com/go-sql-driver/mysql"
)

// Inspection 是目标库当前结构的只读快照，指纹绑定格式化授权。
type Inspection struct {
	State       string   `json:"state"`
	Differences []string `json:"differences"`
	Objects     []string `json:"objects"`
	Fingerprint string   `json:"fingerprint"`
}

var identifierPattern = regexp.MustCompile(`^[\p{L}\p{N}_$-]{1,64}$`)
var integerWidth = regexp.MustCompile(`\b(tinyint|smallint|mediumint|int|bigint)\(\d+\)`)
var spaces = regexp.MustCompile(`\s+`)

// ValidateDatabaseName 保护系统库并限制标识符，不限制账号和密码中的合法特殊字符。
func ValidateDatabaseName(name string) error {
	if !identifierPattern.MatchString(name) {
		return errors.New("数据库名须为 1 至 64 个字母、数字、下划线、连字符或美元符号")
	}
	switch strings.ToLower(name) {
	case "mysql", "sys", "information_schema", "performance_schema":
		return errors.New("不能使用 MySQL 系统数据库")
	}
	return nil
}

// ConnectSetup 不创建表，selectDatabase 为 false 时允许数据库尚不存在。
func ConnectSetup(ctx context.Context, c installation.Config, selectDatabase bool) (*sql.DB, error) {
	if !selectDatabase {
		c.Database = ""
	}
	cfg, err := c.MySQL()
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(0)
	if err = db.PingContext(ctx); err != nil {
		return nil, errors.Join(err, db.Close())
	}
	return db, nil
}

type setupQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// InspectSetup 直接读取元数据，不通过 Open 或初始化缓存。
func InspectSetup(ctx context.Context, c installation.Config) (result Inspection, err error) {
	if err = ValidateDatabaseName(c.Database); err != nil {
		return result, err
	}
	db, err := ConnectSetup(ctx, c, false)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	return inspectSetup(ctx, db, c.Database)
}

// splitDefinitions 按最外层逗号拆分列与索引，保留类型长度和复合索引。
func splitDefinitions(value string) []string {
	var parts []string
	depth, start := 0, 0
	quoted := false
	for i, r := range value {
		if r == '\'' {
			quoted = !quoted
		}
		if quoted {
			continue
		}
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, strings.TrimSpace(value[start:i]))
				start = i + 1
			}
		}
	}
	return append(parts, strings.TrimSpace(value[start:]))
}

// normalizedSchema 将项目 DDL 与服务器返回的 DDL 归一化为可比较定义。
// 归一化处理整数显示宽度、隐式 NULL、主键非空及 MySQL 默认值显示差异。
func normalizedSchema(ddl string) map[string]string {
	ddl = strings.ToLower(strings.ReplaceAll(ddl, "`", ""))
	ddl = strings.ReplaceAll(ddl, "/*!50100 ", "")
	ddl = strings.ReplaceAll(ddl, "*/", "")
	begin, end := strings.Index(ddl, "("), strings.LastIndex(ddl, ")")
	result := map[string]string{}
	if begin < 0 || end < begin {
		return result
	}
	tableCollation := "utf8mb4_unicode_ci"
	if strings.Contains(ddl[end:], "collate=utf8mb4_bin") {
		tableCollation = "utf8mb4_bin"
	}
	result["表属性"] = "innodb " + tableCollation
	if !strings.Contains(ddl[end:], "engine=innodb") {
		result["表属性"] = strings.TrimSpace(ddl[end+1:])
	}
	definitions := splitDefinitions(ddl[begin+1 : end])
	var primary []string
	for _, part := range definitions {
		p := spaces.ReplaceAllString(strings.TrimSpace(part), " ")
		p = integerWidth.ReplaceAllString(p, "$1")
		p = strings.ReplaceAll(p, "integer", "int")
		p = strings.ReplaceAll(p, "boolean", "tinyint")
		p = strings.ReplaceAll(p, "default false", "default '0'")
		p = strings.ReplaceAll(p, "default true", "default '1'")
		p = strings.ReplaceAll(p, "default 0", "default '0'")
		p = strings.ReplaceAll(p, "default 1", "default '1'")
		p = strings.ReplaceAll(p, "index ", "key ")
		p = strings.ReplaceAll(p, " using btree", "")
		p = strings.ReplaceAll(p, ", ", ",")
		p = strings.ReplaceAll(p, " (", "(")
		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}
		switch fields[0] {
		case "primary", "unique", "key", "fulltext", "constraint", "check":
			result["索引 "+strings.Split(p, "(")[0]] = p
			if fields[0] == "primary" {
				a, b := strings.Index(p, "("), strings.Index(p, ")")
				if a >= 0 && b > a {
					primary = strings.Split(p[a+1:b], ",")
				}
			}
		default:
			name := fields[0]
			rest := strings.TrimSpace(strings.TrimPrefix(p, name))
			if strings.Contains(rest, "primary key") {
				primary = append(primary, name)
				result["索引 primary key"] = "primary key(" + name + ")"
				rest = strings.ReplaceAll(rest, " primary key", "")
			}
			if strings.HasSuffix(rest, " unique") {
				result["索引 unique key "+name] = "unique key " + name + "(" + name + ")"
				rest = strings.TrimSuffix(rest, " unique")
			}
			rest = strings.ReplaceAll(rest, " default null", "")
			if !strings.Contains(rest, "not null") {
				rest = strings.TrimSuffix(rest, " null")
			}
			// 字符列必须继承或明确使用同一排序规则，服务器显式字符集不改变语义。
			rest = strings.ReplaceAll(rest, " character set utf8mb4", "")
			rest = strings.ReplaceAll(rest, " collate "+tableCollation, "")
			result["字段 "+name] = rest
		}
	}
	for _, name := range primary {
		key := "字段 " + name
		if value, ok := result[key]; ok && !strings.Contains(value, "not null") {
			result[key] = value + " not null"
		}
	}
	return result
}

// inspectSetup 记录所有对象及表定义，额外对象保留但纳入授权指纹。
func inspectSetup(ctx context.Context, q setupQuerier, name string) (result Inspection, err error) {
	result.Differences = []string{}
	result.Objects = []string{}
	var found string
	err = q.QueryRowContext(ctx, "SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?", name).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		result.State = "missing"
		result.Fingerprint = "missing"
		return result, nil
	}
	if err != nil {
		return result, err
	}
	rows, err := q.QueryContext(ctx, `SELECT object_name, object_type, definition FROM (
 SELECT TABLE_NAME object_name,TABLE_TYPE object_type,'' definition FROM information_schema.TABLES WHERE TABLE_SCHEMA=?
 UNION ALL SELECT ROUTINE_NAME,ROUTINE_TYPE,COALESCE(ROUTINE_DEFINITION,'') FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA=?
 UNION ALL SELECT EVENT_NAME,'EVENT',CONCAT(COALESCE(EVENT_DEFINITION,''),'|',STATUS,'|',COALESCE(CAST(EXECUTE_AT AS CHAR),''),'|',COALESCE(INTERVAL_VALUE,''),'|',COALESCE(INTERVAL_FIELD,'')) FROM information_schema.EVENTS WHERE EVENT_SCHEMA=?
 UNION ALL SELECT TRIGGER_NAME,'TRIGGER',CONCAT(EVENT_OBJECT_TABLE,'|',ACTION_TIMING,'|',EVENT_MANIPULATION,'|',ACTION_STATEMENT) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=?
 UNION ALL SELECT TABLE_NAME,'VIEW DEFINITION',COALESCE(VIEW_DEFINITION,'') FROM information_schema.VIEWS WHERE TABLE_SCHEMA=?
 ) objects ORDER BY object_type,object_name`, name, name, name, name, name)
	if err != nil {
		return result, err
	}
	tables := map[string]string{}
	definitions := map[string]string{}
	triggerTables := map[string]bool{}
	for rows.Next() {
		var object, kind, definition string
		if err = rows.Scan(&object, &kind, &definition); err != nil {
			break
		}
		definitions[kind+":"+object] = definition
		if kind != "VIEW DEFINITION" {
			result.Objects = append(result.Objects, kind+": "+object)
		}
		if kind == "TRIGGER" {
			triggerTables[strings.SplitN(definition, "|", 2)[0]] = true
		}
		if kind == "BASE TABLE" || kind == "VIEW" {
			tables[object] = kind
		}
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return result, err
	}
	snapshots := map[string]string{}
	for table, kind := range tables {
		var returned, ddl string
		if kind == "VIEW" {
			snapshots[table] = "VIEW"
			continue
		}
		err = q.QueryRowContext(ctx, "SHOW CREATE TABLE "+quoteIdentifier(name)+"."+quoteIdentifier(table)).Scan(&returned, &ddl)
		if err != nil {
			return result, err
		}
		snapshots[table] = ddl
	}
	encoded, err := json.Marshal(struct {
		Objects     []string
		Tables      map[string]string
		Definitions map[string]string
	}{result.Objects, snapshots, definitions})
	if err != nil {
		return result, err
	}
	hash := sha256.Sum256(encoded)
	result.Fingerprint = hex.EncodeToString(hash[:])
	if len(result.Objects) == 0 {
		result.State = "empty"
		return result, nil
	}
	for _, ddl := range SchemaStatements() {
		if !strings.HasPrefix(ddl, "CREATE TABLE") {
			continue
		}
		table := strings.Fields(ddl)[5]
		if triggerTables[table] {
			result.Differences = append(result.Differences, table+" / 存在项目未定义的触发器，无法保证写入行为")
		}
		actual, ok := snapshots[table]
		if !ok || tables[table] != "BASE TABLE" {
			result.Differences = append(result.Differences, "缺少数据表："+table)
			continue
		}
		expected, observed := normalizedSchema(ddl), normalizedSchema(actual)
		for key, value := range expected {
			if observed[key] != value {
				result.Differences = append(result.Differences, table+" / "+key+"：需要 "+value+"；实际 "+observed[key])
			}
		}
		for key := range observed {
			if _, ok := expected[key]; !ok {
				result.Differences = append(result.Differences, table+" / 存在额外定义："+key)
			}
		}
	}
	sort.Strings(result.Differences)
	result.State = "compatible"
	if len(result.Differences) > 0 {
		result.State = "incompatible"
	}
	return result, nil
}

func quoteIdentifier(name string) string { return "`" + strings.ReplaceAll(name, "`", "``") + "`" }

// PrepareSetup 在数据库级命名锁内重新检查并执行创建或用户已授权的完整重建。
// DDL 自动提交，错误消息保留操作阶段，不承诺回滚已执行的结构变更。
func PrepareSetup(ctx context.Context, c installation.Config, checked Inspection, format bool) (err error) {
	if err = ValidateDatabaseName(c.Database); err != nil {
		return err
	}
	db, err := ConnectSetup(ctx, c, false)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, db.Close()) }()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, conn.Close()) }()
	hash := sha256.Sum256([]byte(c.Database))
	lockName := "leafdoc-setup-" + hex.EncodeToString(hash[:20])
	var locked int
	if err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", lockName).Scan(&locked); err != nil {
		return err
	}
	if locked != 1 {
		return errors.New("该数据库正在被另一个安装流程使用，请稍后重试")
	}
	defer func() {
		var released int
		releaseErr := conn.QueryRowContext(context.Background(), "SELECT RELEASE_LOCK(?)", lockName).Scan(&released)
		err = errors.Join(err, releaseErr)
	}()
	current, err := inspectSetup(ctx, conn, c.Database)
	if err != nil {
		return err
	}
	if current.Fingerprint != checked.Fingerprint {
		return errors.New("数据库结构已变化，请返回数据库步骤重新检查并确认")
	}
	if current.State == "incompatible" {
		if !format {
			return errors.New("数据库结构不兼容，尚未确认格式化，现有数据已保留")
		}
		if _, err = conn.ExecContext(ctx, "DROP DATABASE "+quoteIdentifier(c.Database)); err != nil {
			return fmt.Errorf("清空数据库失败，请检查 DROP 权限: %w", err)
		}
	}
	if current.State == "missing" || current.State == "incompatible" {
		if _, err = conn.ExecContext(ctx, "CREATE DATABASE "+quoteIdentifier(c.Database)+" CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
			return fmt.Errorf("创建数据库失败，请检查 CREATE 权限: %w", err)
		}
	}
	if _, err = conn.ExecContext(ctx, "USE "+quoteIdentifier(c.Database)); err != nil {
		return err
	}
	if current.State != "compatible" {
		for _, ddl := range SchemaStatements() {
			if _, err = conn.ExecContext(ctx, ddl); err != nil {
				return fmt.Errorf("初始化结构失败，已创建的对象会保留，请重新检查: %w", err)
			}
		}
	}
	verified, err := inspectSetup(ctx, conn, c.Database)
	if err != nil {
		return err
	}
	if verified.State != "compatible" {
		return fmt.Errorf("结构复查失败: %s", strings.Join(verified.Differences, "；"))
	}
	schemaMu.Lock()
	initialized = map[string]bool{}
	schemaMu.Unlock()
	return nil
}

// SetupError 返回可操作的分类说明，调用方还会移除当前会话中的敏感值。
func SetupError(err error) string {
	var driverErr *mysql.MySQLError
	if errors.As(err, &driverErr) {
		switch driverErr.Number {
		case 1045:
			return "数据库账号或密码错误，请检查账号、密码和来源主机授权"
		case 1044, 1142, 1227:
			return "数据库权限不足，请检查目标库授权。" + err.Error()
		case 1049:
			return "数据库不存在或对当前账号不可见，请检查数据库名与授权"
		}
	}
	return err.Error()
}
