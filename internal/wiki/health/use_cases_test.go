package health

import (
	"testing"

	"github.com/NianBroken/LeafDoc/internal/core/search"
	"github.com/NianBroken/LeafDoc/internal/testutil"
)

func TestHealthReportsMySQLAndSearchReadiness(t *testing.T) {
	testutil.ResetMySQL(t)
	directory := t.TempDir()
	index, err := search.NewMySQLIndex(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := index.Close(); err != nil {
			t.Errorf("close MySQL index: %v", err)
		}
	}()
	status := search.NewIndexingStatus()
	status.Success()
	status.Finish()
	service := NewHealthUseCase(index, status, directory)
	healthy, checks := service.Execute()
	if !healthy || checks["mysql"] != "ok" || checks["search"] != "ok" || checks["data_dir"] != "ok" {
		t.Fatalf("unexpected health: %v %v", healthy, checks)
	}
	if _, exists := checks["sqlite"]; exists {
		t.Fatal("health exposes a removed database engine")
	}
	if _, exists := checks["mysql"]; !exists {
		t.Fatal("health must expose the MySQL connection")
	}
	if err := index.Close(); err != nil {
		t.Fatal(err)
	}
	healthy, checks = service.Execute()
	if healthy || checks["mysql"] != "failed" {
		t.Fatalf("closed database reported healthy: %v", checks)
	}
}
