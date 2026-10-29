package database_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/database"
	"github.com/NianBroken/LeafDoc/internal/testutil"
)

func TestMain(m *testing.M) { testutil.RunMySQLTests(m, "database") }

// Round-trip assertions cover binary bytes, SQL NULL and transaction rollback.
func TestDumpRoundTripAndRollback(t *testing.T) {
	testutil.ResetMySQL(t)
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE dump_roundtrip(id INTEGER PRIMARY KEY, content LONGBLOB NULL) ENGINE=InnoDB"); err != nil {
		t.Fatal(err)
	}
	defer db.Exec("DROP TABLE dump_roundtrip")
	original := []byte{0, 255, 1, 0, 128}
	if _, err := db.Exec("INSERT INTO dump_roundtrip VALUES(1,?),(2,NULL),(3,?)", original, []byte{}); err != nil {
		t.Fatal(err)
	}
	raw, err := database.Export(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.ValidateDump(raw); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("UPDATE dump_roundtrip SET content='changed' WHERE id=1"); err != nil {
		t.Fatal(err)
	}
	tx, err := database.Import(context.Background(), db, raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var content []byte
	if err := db.QueryRow("SELECT content FROM dump_roundtrip WHERE id=1").Scan(&content); err != nil || string(content) != "changed" {
		t.Fatalf("rollback %x %v", content, err)
	}
	tx, err = database.Import(context.Background(), db, raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT content FROM dump_roundtrip WHERE id=1").Scan(&content); err != nil || !bytes.Equal(content, original) {
		t.Fatalf("binary restore %x %v", content, err)
	}
	var null, empty bool
	if err := db.QueryRow("SELECT content IS NULL FROM dump_roundtrip WHERE id=2").Scan(&null); err != nil || !null {
		t.Fatal("SQL NULL lost", err)
	}
	if err := db.QueryRow("SELECT content IS NOT NULL AND OCTET_LENGTH(content)=0 FROM dump_roundtrip WHERE id=3").Scan(&empty); err != nil || !empty {
		t.Fatal("empty binary lost", err)
	}
}

func TestUnavailableMySQLHasNoFallback(t *testing.T) {
	t.Setenv("LEAFDOC_MYSQL_HOST", "127.0.0.1")
	t.Setenv("LEAFDOC_MYSQL_PORT", "1")
	dir := t.TempDir()
	if db, err := database.Open(dir); err == nil {
		db.Close()
		t.Fatal("unavailable database accepted")
	}
	files, err := os.ReadDir(dir)
	if err != nil || len(files) != 0 {
		t.Fatalf("local fallback files %v %v", files, err)
	}
}

func TestMalformedDumpRejected(t *testing.T) {
	for _, raw := range []string{`{}`, `{"format":2}`, `{"format":1,"tables":[{"name":"users;DROP","columns":["id"]}]}`} {
		if _, err := database.ValidateDump([]byte(raw)); err == nil {
			t.Fatal("malformed dump accepted")
		}
	}
}
