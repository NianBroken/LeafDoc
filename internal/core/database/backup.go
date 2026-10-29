package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// MaintenanceMu serializes backups and restores that share resource directories.
var MaintenanceMu sync.Mutex
var identifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// TableDump preserves SQL NULL separately from empty text and binary data.
type TableDump struct {
	Name    string     `json:"name"`
	Columns []string   `json:"columns"`
	Rows    [][][]byte `json:"rows"`
}

// Dump is a versioned, credential-free snapshot of every application table.
type Dump struct {
	Format int         `json:"format"`
	Tables []TableDump `json:"tables"`
}

// Export reads all tables in a repeatable-read transaction while writes are gated.
func Export(ctx context.Context, dir string) ([]byte, error) {
	db, err := Open(dir)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	names, err := tableNames(ctx, db)
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	dump := Dump{Format: 1}
	for _, name := range names {
		rows, err := tx.QueryContext(ctx, "SELECT * FROM `"+name+"`")
		if err != nil {
			return nil, err
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			return nil, err
		}
		table := TableDump{Name: name, Columns: columns, Rows: make([][][]byte, 0)}
		for rows.Next() {
			values := make([]any, len(columns))
			scan := make([]any, len(columns))
			for i := range values {
				scan[i] = &values[i]
			}
			if err := rows.Scan(scan...); err != nil {
				rows.Close()
				return nil, err
			}
			cells := make([][]byte, len(columns))
			for i, value := range values {
				switch v := value.(type) {
				case nil:
					cells[i] = nil
				case []byte:
					cells[i] = append([]byte{}, v...)
				case time.Time:
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

func tableNames(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_TYPE='BASE TABLE' ORDER BY TABLE_NAME")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

// ValidateDump rejects unknown formats, duplicate tables and malformed rows.
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

// Import stages all table replacement inside a transaction; the caller owns commit.
// The current schema must match the backup, so archives cannot write arbitrary tables.
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
	fail := func(err error) (*sql.Tx, error) { tx.Rollback(); return nil, err }
	for _, table := range dump.Tables {
		if _, err := tx.ExecContext(ctx, "DELETE FROM `"+table.Name+"`"); err != nil {
			return fail(err)
		}
		// Sessions are deliberately empty after restoration.
		if table.Name == "sessions" {
			continue
		}
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
				stmt.Close()
				return fail(err)
			}
		}
		if err := stmt.Close(); err != nil {
			return fail(err)
		}
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO auth_state(id,generation) VALUES(1,UUID()) ON DUPLICATE KEY UPDATE generation=UUID()"); err != nil {
		return fail(err)
	}
	return tx, nil
}
