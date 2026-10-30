// SQL initialization must produce the same tables and indexes as application initialization.
package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/core/defaults"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"golang.org/x/crypto/bcrypt"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "schema-defaults") }

func schemas(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	names := []string{}
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		if name != "test_database_owner" {
			names = append(names, name)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	result := map[string]string{}
	for _, name := range names {
		var table, ddl string
		if err := db.QueryRow("SHOW CREATE TABLE `"+name+"`").Scan(&table, &ddl); err != nil {
			t.Fatal(err)
		}
		result[name] = ddl
	}
	return result
}

func TestDefaultSQLMatchesRuntimeSchema(t *testing.T) {
	testutil.ResetMySQL(t)
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	before := schemas(t, db)
	for name := range before {
		if _, err := db.Exec("DROP TABLE `" + name + "`"); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(filepath.Join("../../internal/core/database/default.sql"))
	if err != nil {
		t.Fatal(err)
	}
	var executable strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			executable.WriteString(line + "\n")
		}
	}
	for pass := 0; pass < 2; pass++ {
		for _, statement := range strings.Split(executable.String(), ";") {
			if strings.TrimSpace(statement) != "" {
				if _, err := db.Exec(statement); err != nil {
					t.Fatalf("SQL statement failed: %v\n%s", err, statement)
				}
			}
		}
	}
	after := schemas(t, db)
	if len(before) != len(after) {
		t.Fatal("table count differs", len(before), len(after))
	}
	for name, ddl := range before {
		if ddl != after[name] {
			t.Fatalf("schema differs for %s\n%s\n%s", name, ddl, after[name])
		}
	}
	var username, email, password string
	if err := db.QueryRow("SELECT username,email,password FROM users WHERE role='admin'").Scan(&username, &email, &password); err != nil {
		t.Fatal(err)
	}
	if username != defaults.AdminUsername || email != defaults.AdminEmail || bcrypt.CompareHashAndPassword([]byte(password), []byte(defaults.AdminPassword)) != nil {
		t.Fatal("initial admin values differ")
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM content_records WHERE namespace='settings'").Scan(&count); err != nil || count != 5 {
		t.Fatal("settings not initialized", count, err)
	}
}
