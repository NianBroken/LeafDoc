// installation 管理安装实例配置。候选配置只在初始化期间使用，成功记录以原子替换持久保存。
package installation

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

const FileName = "installation.json"

// Config 同时记录连接信息与安装完成状态，密码不经字符串裁剪或环境变量展开。
type Config struct {
	Complete      bool   `json:"complete"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	Database      string `json:"database"`
	User          string `json:"user"`
	Password      string `json:"password"`
	TLS           bool   `json:"tls"`
	CA            string `json:"ca"`
	Admin         string `json:"admin"`
	Email         string `json:"email"`
	AdminPassword string `json:"adminPassword,omitempty"`
	Secret        string `json:"secret"`
}

var candidates = struct {
	sync.RWMutex
	values map[string]*Config
}{values: map[string]*Config{}}

// Candidate 为当前进程安装事务提供配置，不写出未完成记录。
func Candidate(dir string, c *Config) {
	dir = filepath.Clean(dir)
	if absolute, err := filepath.Abs(dir); err == nil {
		dir = absolute
	}
	candidates.Lock()
	defer candidates.Unlock()
	if c == nil {
		delete(candidates.values, dir)
	} else {
		candidates.values[dir] = c
	}
}

// Load 优先返回进程内候选值；文件损坏是错误，不能被当作全新安装。
func Load(dir string) (*Config, error) {
	dir = filepath.Clean(dir)
	if absolute, err := filepath.Abs(dir); err == nil {
		dir = absolute
	}
	candidates.RLock()
	c := candidates.values[dir]
	candidates.RUnlock()
	if c != nil {
		return c, nil
	}
	b, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		return nil, err
	}
	var result Config
	if err = json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	if !result.Complete {
		return nil, errors.New("安装记录未完成，请检查实例配置")
	}
	return &result, nil
}

// MySQL 将结构化配置转换为驱动配置，TLS 始终校验证书和服务器名称。
func (c *Config) MySQL() (*mysql.Config, error) {
	if c.Host == "" || c.User == "" || c.Database == "" || c.Port < 1 || c.Port > 65535 {
		return nil, errors.New("数据库地址、端口、账号和数据库名不能为空或无效")
	}
	m := mysql.NewConfig()
	m.Net = "tcp"
	m.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	m.User = c.User
	m.Passwd = c.Password
	m.DBName = c.Database
	m.ParseTime = true
	m.Loc = time.UTC
	m.Timeout = 10 * time.Second
	m.ReadTimeout = 30 * time.Second
	m.WriteTimeout = 30 * time.Second
	m.Collation = "utf8mb4_unicode_ci"
	m.Params = map[string]string{"charset": "utf8mb4"}
	m.ClientFoundRows = true
	if c.TLS {
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		if c.CA != "" && !roots.AppendCertsFromPEM([]byte(c.CA)) {
			return nil, errors.New("数据库 CA 证书无效")
		}
		t := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: c.Host}
		if err = mysql.RegisterTLSConfig("leafdoc-installation", t); err != nil {
			return nil, err
		}
		m.TLSConfig = "leafdoc-installation"
	}
	return m, nil
}

// Save 先同步临时文件再替换正式记录，目录和文件均限制访问权限。
func Save(dir string, c *Config) error {
	if !c.Complete {
		return errors.New("拒绝保存未完成的安装")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".installation-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = os.Rename(name, filepath.Join(dir, FileName)); err != nil {
		return fmt.Errorf("保存安装配置: %w", err)
	}
	return nil
}

// Find 从存储子目录向上寻找同一实例配置，覆盖导入任务等独立存储目录。
func Find(dir string) (*Config, error) {
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	for {
		c, err := Load(absolute)
		if err == nil {
			return c, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
		}
		parent := filepath.Dir(absolute)
		if parent == absolute {
			return nil, err
		}
		absolute = parent
	}
}
