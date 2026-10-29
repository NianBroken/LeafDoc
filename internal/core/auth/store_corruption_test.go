package auth

import (
	"github.com/NianBroken/LeafDoc/internal/testutil"
	"os"
	"testing"
)

func TestAuthStores_MySQLUnavailableFailsWithoutLocalFallback(t *testing.T) {
	testutil.ResetMySQL(t)
	t.Setenv("LEAFDOC_MYSQL_HOST", "127.0.0.1")
	t.Setenv("LEAFDOC_MYSQL_PORT", "1")
	for _, kind := range []string{"users", "sessions", "keys", "email"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			var err error
			switch kind {
			case "users":
				_, err = NewUserStore(dir)
			case "sessions":
				_, err = NewSessionStore(dir)
			case "keys":
				_, err = NewAPIKeyStore(dir)
			case "email":
				_, err = NewEmailTokenStore(dir)
			}
			if err == nil {
				t.Fatal("unavailable MySQL must reject store initialization")
			}
			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Fatalf("unexpected local fallback files: %v", entries)
			}
		})
	}
}

func TestUserStore_DataDirectoriesShareConfiguredMySQL(t *testing.T) {
	testutil.ResetMySQL(t)
	first, err := NewUserStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	user := &User{ID: "shared", Username: "中文账户", Email: "shared@example.test", Password: "hash", Role: RoleAdmin}
	if err := first.CreateUser(user); err != nil {
		t.Fatal(err)
	}
	second, err := NewUserStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	actual, err := second.GetUserByID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Username != user.Username {
		t.Fatalf("unexpected shared record: %#v", actual)
	}
}
