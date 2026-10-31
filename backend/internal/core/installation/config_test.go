// 安装记录测试覆盖凭据原样保存、损坏文件和候选配置隔离。
package installation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadPreservesCredentials(t *testing.T) {
	dir := t.TempDir()
	c := &Config{Complete: true, Host: "db.example.test", Port: 3306, Database: "docs", User: "user", Password: "  'a=b\"中文  ", Secret: "test-secret"}
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Password != c.Password {
		t.Fatal("密码发生变化")
	}
	if loaded.Complete != true {
		t.Fatal("完成状态丢失")
	}
	m, err := loaded.MySQL()
	if err != nil {
		t.Fatal(err)
	}
	if m.Passwd != c.Password {
		t.Fatal("驱动配置密码变化")
	}
}

func TestCorruptRecordIsNotFreshInstall(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil || os.IsNotExist(err) {
		t.Fatal("损坏配置被当作未安装")
	}
}

func TestCandidateDoesNotPersistAndCannotSaveIncomplete(t *testing.T) {
	dir := t.TempDir()
	c := &Config{Host: "database", Port: 3306, User: "user", Database: "docs"}
	Candidate(dir, c)
	loaded, err := Load(dir)
	if err != nil || loaded != c {
		t.Fatal("候选配置不可用")
	}
	Candidate(dir, nil)
	if _, err = Load(dir); !os.IsNotExist(err) {
		t.Fatal("候选配置被持久化")
	}
	if err = Save(dir, c); err == nil {
		t.Fatal("接受了未完成的配置")
	}
}

func TestMySQLRejectsInvalidAddressAndCA(t *testing.T) {
	c := &Config{Host: "db", Port: 65536, User: "user", Database: "docs"}
	if _, err := c.MySQL(); err == nil {
		t.Fatal("接受了无效端口")
	}
	c.Port = 3306
	c.TLS = true
	c.CA = "invalid"
	if _, err := c.MySQL(); err == nil {
		t.Fatal("接受了无效 CA")
	}
}

// 子存储必须取得同一安装记录，不能退回缺失的环境配置。
func TestFindResolvesChildStorage(t *testing.T) {
	dir := t.TempDir()
	c := &Config{Complete: true, Host: "database", Port: 3306, User: "user", Database: "docs"}
	Candidate(dir, c)
	defer Candidate(dir, nil)
	loaded, err := Find(filepath.Join(dir, "imports", "jobs"))
	if err != nil || loaded != c {
		t.Fatalf("子目录未取得候选配置: %v", err)
	}
	Candidate(dir, nil)
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	loaded, err = Find(filepath.Join(dir, "imports", "jobs"))
	if err != nil || loaded.Host != c.Host {
		t.Fatalf("子目录未取得持久化配置: %v", err)
	}
}
