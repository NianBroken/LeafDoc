package testutil

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"regexp"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

var testDatabasePattern = regexp.MustCompile(`^leafdoc_test_migration_[0-9a-f]{24}$`)
var testTablePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// RunMySQLTests owns exactly one isolated database for a package test process.
// The recorded owner token is checked before cleanup so unrelated databases
// cannot be removed. Every failure is reported and contributes to the exit code.
func RunMySQLTests(m *testing.M, packageName string) {
	code := runMySQLTests(m, packageName)
	os.Exit(code)
}

func runMySQLTests(m *testing.M, packageName string) int {
	directory, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	configuration, err := database.Configuration(directory)
	if err != nil {
		fmt.Fprintln(os.Stderr, "MySQL test configuration:", err)
		return 1
	}
	token := make([]byte, 12)
	if _, err := rand.Read(token); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	owner := hex.EncodeToString(token)
	name := "leafdoc_test_migration_" + owner
	configuration.DBName = ""
	admin, err := sql.Open("mysql", configuration.FormatDSN())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer admin.Close()
	admin.SetMaxOpenConns(1)
	if _, err := admin.Exec("CREATE DATABASE `" + name + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		fmt.Fprintln(os.Stderr, "create MySQL test database:", err)
		return 1
	}
	cleanup := func() error {
		if !testDatabasePattern.MatchString(name) {
			return fmt.Errorf("refusing to remove unowned test database")
		}
		var actual string
		if err := admin.QueryRow("SELECT owner_token FROM `" + name + "`.test_database_owner LIMIT 1").Scan(&actual); err != nil {
			return err
		}
		if actual != owner {
			return fmt.Errorf("MySQL test database ownership changed")
		}
		_, err := admin.Exec("DROP DATABASE `" + name + "`")
		return err
	}
	if _, err := admin.Exec("CREATE TABLE `" + name + "`.test_database_owner(owner_token VARCHAR(24) PRIMARY KEY, package_name VARCHAR(191) NOT NULL)"); err != nil {
		// This database was created by this process and has not been shared.
		admin.Exec("DROP DATABASE `" + name + "`")
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if _, err := admin.Exec("INSERT INTO `"+name+"`.test_database_owner VALUES(?,?)", owner, packageName); err != nil {
		admin.Exec("DROP DATABASE `" + name + "`")
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	configuration.DBName = name
	// Addr was produced by net.JoinHostPort, so use the standard parser for IPv6.
	host, port, splitErr := net.SplitHostPort(configuration.Addr)
	if splitErr != nil {
		cleanup()
		fmt.Fprintln(os.Stderr, splitErr)
		return 1
	}
	for key, value := range map[string]string{
		"LEAFDOC_MYSQL_HOST":     host,
		"LEAFDOC_MYSQL_PORT":     port,
		"LEAFDOC_MYSQL_USER":     configuration.User,
		"LEAFDOC_MYSQL_PASSWORD": configuration.Passwd,
		"LEAFDOC_MYSQL_DATABASE": name,
	} {
		if err := os.Setenv(key, value); err != nil {
			cleanup()
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	db, err := database.Open(directory)
	if err != nil {
		cleanup()
		fmt.Fprintln(os.Stderr, "initialize MySQL test schema:", err)
		return 1
	}
	db.Close()
	code := m.Run()
	if err := database.ClosePools(); err != nil {
		fmt.Fprintln(os.Stderr, "close MySQL record pools:", err)
		code = 1
	}
	if err := cleanup(); err != nil {
		fmt.Fprintln(os.Stderr, "clean MySQL test database:", err)
		code = 1
	}
	return code
}

// ResetMySQL clears application records between sequential tests without creating
// more databases. It refuses the production schema and preserves the owner row.
func ResetMySQL(t *testing.T) {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	config, err := database.Configuration(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !testDatabasePattern.MatchString(config.DBName) {
		t.Fatal("refusing to clear a non-test MySQL database")
	}
	db, err := database.Open(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if name != "test_database_owner" {
			names = append(names, name)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, name := range names {
		if !testTablePattern.MatchString(name) {
			t.Fatalf("invalid test table name: %s", name)
		}
		if _, err := tx.Exec("DELETE FROM `" + name + "`"); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}
