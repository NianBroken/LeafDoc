// 安装测试验证规范化和真实 MySQL 上的数据保护行为。
package database

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/NianBroken/LeafDoc/internal/core/installation"
)

func TestSetupSchemaNormalization(t *testing.T) {
	expected := normalizedSchema("CREATE TABLE IF NOT EXISTS sample (id INTEGER PRIMARY KEY, flag BOOLEAN NOT NULL DEFAULT FALSE, value VARCHAR(64) UNIQUE) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")
	actual := normalizedSchema("CREATE TABLE `sample` (`id` int NOT NULL, `flag` tinyint(1) NOT NULL DEFAULT '0', `value` varchar(64) DEFAULT NULL, PRIMARY KEY (`id`), UNIQUE KEY `value` (`value`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci")
	for key, value := range expected {
		if actual[key] != value {
			t.Fatalf("%s expected %q got %q", key, value, actual[key])
		}
	}
	for _, name := range []string{"mysql", "sys", "information_schema", "performance_schema", "bad`name", ""} {
		if ValidateDatabaseName(name) == nil {
			t.Fatalf("unsafe name %q", name)
		}
	}
}

// TestSetupMySQLIntegration 仅使用显式测试服务，目标名由测试进程实时生成。
func TestSetupMySQLIntegration(t *testing.T) {
	host := os.Getenv("LEAFDOC_SETUP_TEST_HOST")
	if host == "" {
		t.Skip("isolated MySQL not configured")
	}
	c := installation.Config{Host: host, Port: 3306, User: "root", Password: os.Getenv("LEAFDOC_SETUP_TEST_PASSWORD"), Database: "leafdoc_setup_test_" + strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "")}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	db, err := ConnectSetup(ctx, c, false)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()
	defer func() {
		if _, err := db.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+quoteIdentifier(c.Database)); err != nil {
			t.Error(err)
		}
	}()
	missing, err := InspectSetup(ctx, c)
	if err != nil || missing.State != "missing" {
		t.Fatalf("missing: %+v %v", missing, err)
	}
	if _, err = db.ExecContext(ctx, "CREATE DATABASE "+quoteIdentifier(c.Database)+" CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		t.Fatal(err)
	}
	empty, err := InspectSetup(ctx, c)
	if err != nil || empty.State != "empty" {
		t.Fatalf("empty: %+v %v", empty, err)
	}
	if _, err = db.ExecContext(ctx, "DROP DATABASE "+quoteIdentifier(c.Database)); err != nil {
		t.Fatal(err)
	}
	if err = PrepareSetup(ctx, c, missing, false); err != nil {
		t.Fatal(err)
	}
	correct, err := InspectSetup(ctx, c)
	if err != nil || correct.State != "compatible" {
		t.Fatalf("correct: %+v %v", correct, err)
	}
	if err = PrepareSetup(ctx, c, correct, false); err != nil {
		t.Fatal(err)
	}
	target := quoteIdentifier(c.Database) + "."
	for _, statement := range []string{"ALTER TABLE " + target + "sessions DROP INDEX sessions_expiry", "ALTER TABLE " + target + "users MODIFY email VARCHAR(100) NULL", "ALTER TABLE " + target + "search_documents DROP INDEX search_text", "CREATE TABLE " + target + "protected_data (value INT)", "INSERT INTO " + target + "protected_data VALUES (42)"} {
		if _, err = db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	broken, err := InspectSetup(ctx, c)
	if err != nil || broken.State != "incompatible" || len(broken.Differences) < 3 {
		t.Fatalf("broken: %+v %v", broken, err)
	}
	if err = PrepareSetup(ctx, c, broken, false); err == nil {
		t.Fatal("format without consent")
	}
	var value int
	if err = db.QueryRowContext(ctx, "SELECT value FROM "+target+"protected_data").Scan(&value); err != nil || value != 42 {
		t.Fatal("existing data changed", err)
	}
	if _, err = db.ExecContext(ctx, "CREATE TABLE "+target+"changed_after_check (value INT)"); err != nil {
		t.Fatal(err)
	}
	if err = PrepareSetup(ctx, c, broken, true); err == nil {
		t.Fatal("stale consent accepted")
	}
	broken, err = InspectSetup(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	if err = PrepareSetup(ctx, c, broken, true); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := InspectSetup(ctx, c)
	if err != nil || rebuilt.State != "compatible" {
		t.Fatalf("rebuilt: %+v %v", rebuilt, err)
	}
	for _, object := range rebuilt.Objects {
		if strings.Contains(object, "protected_data") {
			t.Fatal("old objects remain")
		}
	}
}
