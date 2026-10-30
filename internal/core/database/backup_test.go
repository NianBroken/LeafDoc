package database_test

import (
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/database"
)

func TestMalformedDumpRejected(t *testing.T) {
	for _, raw := range []string{`{}`, `{"format":2}`, `{"format":1,"tables":[{"name":"users;DROP","columns":["id"]}]}`} {
		if _, err := database.ValidateDump([]byte(raw)); err == nil {
			t.Fatal("malformed dump accepted")
		}
	}
}
