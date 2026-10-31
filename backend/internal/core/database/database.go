// Package database 管理 MySQL 配置、连接池、表结构及通用记录读写。
// 独立存储对象拥有各自连接池，通用记录操作还可使用进程级共享连接池。
package database

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
)

// connectionTimeout 同时限制建立连接和 Open 中连通性检查的等待时间。
const connectionTimeout = 10 * time.Second

// schemaMu 串行化进程内的首次表结构初始化，以完整连接配置作为初始化缓存键。
var schemaMu sync.Mutex
var initialized = map[string]bool{}

// recordPoolMu 保护共享连接池映射，创建连接期间仍持锁，避免同配置重复创建。
var recordPoolMu sync.Mutex
var recordPools = map[string]*sql.DB{}

// Shared 返回进程管理的共享连接池，调用方负责关闭结果集和结束事务。
// 连接池本身交给 ClosePools 统一释放，调用方不应单独关闭共享池。
func Shared(dir string) (*sql.DB, error) { return recordPool(dir) }

// recordPool 按完整连接配置复用连接池，没有缓存时通过 Open 建立并登记。
// 配置读取或打开连接失败时不写入缓存。
func recordPool(dir string) (*sql.DB, error) {
	configuration, err := Configuration(dir)
	if err != nil {
		return nil, err
	}
	key := configuration.FormatDSN()
	recordPoolMu.Lock()
	defer recordPoolMu.Unlock()
	if db := recordPools[key]; db != nil {
		return db, nil
	}
	db, err := Open(dir)
	if err != nil {
		return nil, err
	}
	recordPools[key] = db
	return db, nil
}

// ClosePools 持锁关闭所有共享池并删除缓存项，将全部关闭错误合并返回。
// 即使某个池关闭失败，仍继续处理其余池；结构初始化缓存不在此处清空。
func ClosePools() error {
	recordPoolMu.Lock()
	defer recordPoolMu.Unlock()
	var failures []error
	for key, db := range recordPools {
		failures = append(failures, db.Close())
		delete(recordPools, key)
	}
	return errors.Join(failures...)
}

// IsDuplicate 沿错误链查找 MySQL 错误码 1062，识别唯一键冲突而不匹配消息文本。
func IsDuplicate(err error) bool {
	var violation *mysql.MySQLError
	return errors.As(err, &violation) && violation.Number == 1062
}

// Configuration 从存储目录向父级寻找最近的 .env，并以进程环境变量逐项覆盖文件值。
// 环境变量即使为空也覆盖文件值。配置解析不创建数据库，不提供其他存储引擎回退。
func Configuration(storageDir string) (*mysql.Config, error) {
	values := map[string]string{}
	directory, err := filepath.Abs(storageDir)
	if err != nil {
		return nil, err
	}
	// 只读取遇到的第一个文件，不合并多个父目录配置。非文件不存在的打开错误直接返回。
	for {
		file, openErr := os.Open(filepath.Join(directory, ".env"))
		if openErr == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				// 每行按首个等号分隔，裁去键值边缘空白和引号，不执行变量展开或行内注释解析。
				key, value, ok := strings.Cut(line, "=")
				if ok {
					values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), "\"'")
				}
			}
			scanErr := scanner.Err()
			closeErr := file.Close()
			if scanErr != nil {
				return nil, scanErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			break
		}
		if !os.IsNotExist(openErr) {
			return nil, openErr
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
		directory = parent
	}
	// 查找环境变量是否存在，而非只判断内容是否非空，以保留显式空值的优先级。
	get := func(key string) string {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		return values[key]
	}
	// 主机、用户名和数据库名必填，端口为空使用 3306，密码允许为空。
	host, port := get("LEAFDOC_MYSQL_HOST"), get("LEAFDOC_MYSQL_PORT")
	if host == "" {
		return nil, fmt.Errorf("LEAFDOC_MYSQL_HOST is required")
	}
	if port == "" {
		port = "3306"
	}
	config := mysql.NewConfig()
	config.Net = "tcp"
	config.Addr = net.JoinHostPort(host, port)
	config.User = get("LEAFDOC_MYSQL_USER")
	config.Passwd = get("LEAFDOC_MYSQL_PASSWORD")
	config.DBName = get("LEAFDOC_MYSQL_DATABASE")
	if config.User == "" || config.DBName == "" {
		return nil, fmt.Errorf("MySQL username and database are required")
	}
	// 时间字段按 UTC 解析，连接使用 utf8mb4；读取与写入分别设置超时。
	// ClientFoundRows 使更新语句的受影响行数包含匹配但值未变化的行。
	config.ParseTime = true
	config.Loc = time.UTC
	config.Timeout = connectionTimeout
	config.ReadTimeout = 30 * time.Second
	config.WriteTimeout = 30 * time.Second
	config.Collation = "utf8mb4_unicode_ci"
	config.Params = map[string]string{"charset": "utf8mb4"}
	config.ClientFoundRows = true
	return config, nil
}

// Open 创建独立连接池，限制并发和空闲连接并检查数据库实际连通性。
// 当前进程尚未初始化同配置时，依次执行内容表和关系表定义，成功后登记初始化状态。
// 连通性或建表失败均关闭刚创建的池，并保留操作错误与关闭错误。
func Open(storageDir string) (*sql.DB, error) {
	config, err := Configuration(storageDir)
	if err != nil {
		return nil, err
	}
	dsn := config.FormatDSN()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	// 池最多同时打开四条连接，最多保留两条空闲连接，单连接最长复用三分钟。
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(3 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("MySQL connection failed: %w", errors.Join(err, db.Close()))
	}
	schemaMu.Lock()
	defer schemaMu.Unlock()
	if !initialized[dsn] {
		if _, err := db.Exec(contentSchema); err != nil {
			return nil, errors.Join(err, db.Close())
		}
		if err := EnsureStoreSchema(db); err != nil {
			return nil, errors.Join(err, db.Close())
		}
		initialized[dsn] = true
	}
	return db, nil
}

// ReadRecord 使用共享连接读取一条通用记录，保留没有记录时的 sql.ErrNoRows。
func ReadRecord(dir, namespace, key string) ([]byte, error) {
	return (Records{Dir: dir}).Read(namespace, key)
}

// WriteRecord 用单条插入或更新语句写入完整载荷，不解释载荷内容。
func WriteRecord(dir, namespace, key string, payload []byte) error {
	return (Records{Dir: dir}).Write(namespace, key, payload)
}

// DeleteRecord 删除指定记录，不存在的键也返回成功。
func DeleteRecord(dir, namespace, key string) error {
	return (Records{Dir: dir}).Delete(namespace, key)
}

// ListRecords 读取命名空间的全部载荷并返回独立结果映射，不提供固定遍历次序。
func ListRecords(dir, namespace string) (map[string][]byte, error) {
	return (Records{Dir: dir}).List(namespace)
}
