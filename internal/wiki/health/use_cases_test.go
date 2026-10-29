package health

import (
	"github.com/NianBroken/LeafDoc/internal/core/search"
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"testing"
)

func TestHealthReportsMySQLAndSearchReadiness(t *testing.T) {
	testutil.ResetMySQL(t)
	directory := t.TempDir()
	index, err := search.NewMySQLIndex(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
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
	if err := index.Close(); err != nil {
		t.Fatal(err)
	}
	healthy, checks = service.Execute()
	if healthy || checks["mysql"] != "failed" {
		t.Fatalf("closed database reported healthy: %v", checks)
	}
}
