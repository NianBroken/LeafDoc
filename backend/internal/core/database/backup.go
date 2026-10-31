// 本文件以 JSON 导出和导入当前数据库的业务表。表与列名经过标识符检查，
// 单元格按字节载荷保存，SQL NULL 与空内容分别表示；导入事务交由上层决定提交。
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// MaintenanceMu 供备份与恢复调用方串行维护共享资源，本文件的 Export 和 Import
// 自身不取得这把锁，也不自行暂停业务写入。
var MaintenanceMu sync.Mutex

// 标识符只接受 ASCII 字母或下划线开头，后续可含数字，排除 SQL 结构字符。
var identifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// TableDump 保存列顺序与每行单元格，nil 字节切片表示 SQL NULL，
// 非空指针语义的空切片表示空内容，JSON 编码将字节切片表示为 Base64 文本。
type TableDump struct {
	Name    string     `json:"name"`
	Columns []string   `json:"columns"`
	Rows    [][][]byte `json:"rows"`
}

// Dump 保存格式版本和导出表，不包含数据库连接配置。业务表的账号密码摘要、
// 令牌摘要及其他字段仍按表内容导出，不能把整个载荷视为不含敏感业务信息。
type Dump struct {
	Format int         `json:"format"`
	Tables []TableDump `json:"tables"`
}

// Export 先读取当前库的基础表名称，再在只读可重复读事务中逐表读取全部列和行。
// 调用方负责写入屏障；此函数只把上下文交给数据库操作，不自行取得维护锁。
// 事务提交后编码 JSON，延迟回滚和连接关闭的错误合并到返回值。
func Export(ctx context.Context, dir string) (result []byte, resultErr error) {
	db, err := Open(dir)
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, db.Close()) }()
	// 表清单在事务开启前读取，导出内容随后按表名顺序追加。
	names, err := tableNames(ctx, db)
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, rollback(tx)) }()
	dump := Dump{Format: 1}
	for _, name := range names {
		rows, err := tx.QueryContext(ctx, "SELECT * FROM `"+name+"`")
		if err != nil {
			return nil, err
		}
		columns, err := rows.Columns()
		if err != nil {
			return nil, errors.Join(err, rows.Close())
		}
		table := TableDump{Name: name, Columns: columns, Rows: make([][][]byte, 0)}
		// 每行按驱动提供的值类型转换，字节值复制后再保存，避免引用游标复用缓冲区。
		for rows.Next() {
			values := make([]any, len(columns))
			scan := make([]any, len(columns))
			for i := range values {
				scan[i] = &values[i]
			}
			if err := rows.Scan(scan...); err != nil {
				return nil, errors.Join(err, rows.Close())
			}
			cells := make([][]byte, len(columns))
			for i, value := range values {
				switch v := value.(type) {
				case nil:
					cells[i] = nil
				case []byte:
					cells[i] = append([]byte{}, v...)
				case time.Time:
					// 时间统一为 UTC 的 SQL 文本，最多保留微秒，不保存原时区标识。
					cells[i] = []byte(v.UTC().Format("2006-01-02 15:04:05.999999"))
				default:
					cells[i] = []byte(fmt.Sprint(v))
				}
			}
			table.Rows = append(table.Rows, cells)
		}
		rowErr := rows.Err()
		closeErr := rows.Close()
		if rowErr != nil {
			return nil, rowErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		dump.Tables = append(dump.Tables, table)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return json.Marshal(dump)
}

// tableNames 只选当前数据库的基础表，按表名排序，并核对可用于标识符拼接的字符。
// 视图不在清单内，游标关闭错误与查询或扫描错误合并返回。
func tableNames(ctx context.Context, db *sql.DB) (result []string, resultErr error) {
	rows, err := db.QueryContext(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_TYPE='BASE TABLE' ORDER BY TABLE_NAME")
	if err != nil {
		return nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, rows.Close()) }()
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if !identifier.MatchString(name) {
			return nil, fmt.Errorf("unsupported table identifier")
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// ValidateDump 检查格式一、合法且不重复的表列名、非空列集合及每行的列数。
// 至少需要 users 和 content_records 两张表，不在此校验列类型、业务字段值、
// 当前数据库实际结构或必须存在的其余表。
func ValidateDump(raw []byte) (*Dump, error) {
	var dump Dump
	if err := json.Unmarshal(raw, &dump); err != nil {
		return nil, err
	}
	if dump.Format != 1 {
		return nil, fmt.Errorf("unsupported MySQL backup format")
	}
	seen := map[string]bool{}
	for _, table := range dump.Tables {
		if !identifier.MatchString(table.Name) || seen[table.Name] || len(table.Columns) == 0 {
			return nil, fmt.Errorf("invalid backup table")
		}
		seen[table.Name] = true
		columns := map[string]bool{}
		for _, column := range table.Columns {
			if !identifier.MatchString(column) || columns[column] {
				return nil, fmt.Errorf("invalid backup column")
			}
			columns[column] = true
		}
		for _, row := range table.Rows {
			if len(row) != len(table.Columns) {
				return nil, fmt.Errorf("invalid backup row in %s", table.Name)
			}
		}
	}
	if !seen["users"] || !seen["content_records"] {
		return nil, fmt.Errorf("backup is missing application tables")
	}
	return &dump, nil
}

// Import 先验证载荷与当前基础表名称集合，再开启事务逐表清空并导入记录。
// 表名集合必须完全相同，列名合法性由载荷校验负责，实际列兼容性由 SQL 执行检查。
// 任一步失败回滚；成功返回仍未提交的事务，调用方负责提交或回滚。
func Import(ctx context.Context, db *sql.DB, raw []byte) (*sql.Tx, error) {
	dump, err := ValidateDump(raw)
	if err != nil {
		return nil, err
	}
	names, err := tableNames(ctx, db)
	if err != nil {
		return nil, err
	}
	expected := map[string]bool{}
	for _, name := range names {
		expected[name] = true
	}
	if len(names) != len(dump.Tables) {
		return nil, fmt.Errorf("backup schema does not match this server")
	}
	for _, table := range dump.Tables {
		if !expected[table.Name] {
			return nil, fmt.Errorf("unknown table in backup")
		}
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	fail := func(err error) (*sql.Tx, error) { return nil, errors.Join(err, rollback(tx)) }
	for _, table := range dump.Tables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM `"+table.Name+"`"); err != nil {
			return fail(err)
		}
		// 会话表只清空，不导入备份中的刷新会话记录。
		if table.Name == "sessions" {
			continue
		}
		// 表列标识符已经过字符约束，单元格值全部通过占位符绑定。
		quoted := "`" + strings.Join(table.Columns, "`,`") + "`"
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(table.Columns)), ",")
		stmt, err := tx.PrepareContext(ctx, "INSERT INTO `"+table.Name+"` ("+quoted+") VALUES ("+placeholders+")")
		if err != nil {
			return fail(err)
		}
		for _, row := range table.Rows {
			values := make([]any, len(row))
			for i, cell := range row {
				if cell != nil {
					values[i] = cell
				}
			}
			if _, err := stmt.ExecContext(ctx, values...); err != nil {
				return fail(errors.Join(err, stmt.Close()))
			}
		}
		if err := stmt.Close(); err != nil {
			return fail(err)
		}
	}
	// 导入后在同一事务中生成认证代次，使提交后的认证状态不同于备份中保存的值。
	if _, err := tx.ExecContext(ctx, "INSERT INTO auth_state(id,generation) VALUES(1,UUID()) ON DUPLICATE KEY UPDATE generation=UUID()"); err != nil {
		return fail(err)
	}
	return tx, nil
}
