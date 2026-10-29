// Package database owns MySQL configuration, connections and durable records.
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

const connectionTimeout = 10 * time.Second

var schemaMu sync.Mutex
var initialized = map[string]bool{}
var recordPoolMu sync.Mutex
var recordPools = map[string]*sql.DB{}

// Shared returns the process-owned pool for transactional domain operations.
// Callers close rows and transactions, while final shutdown closes the pool.
func Shared(dir string) (*sql.DB, error) { return recordPool(dir) }

// recordPool keeps short record operations on reusable TCP connections.
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

// ClosePools releases process-owned record connections during final shutdown.
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

// IsDuplicate classifies MySQL unique-key violations without matching log text.
func IsDuplicate(err error) bool {
	var violation *mysql.MySQLError
	return errors.As(err, &violation) && violation.Number == 1062
}

// Configuration reads environment overrides before the nearest private .env file.
// Configuration never creates a database or falls back to a local storage engine.
func Configuration(storageDir string) (*mysql.Config, error) {
	values := map[string]string{}
	directory, err := filepath.Abs(storageDir)
	if err != nil {
		return nil, err
	}
	for {
		file, openErr := os.Open(filepath.Join(directory, ".env"))
		if openErr == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
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
	get := func(key string) string {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
		return values[key]
	}
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

// Open returns an owned connection pool. Callers close it when their store closes.
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
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(3 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("MySQL connection failed: %w", err)
	}
	schemaMu.Lock()
	defer schemaMu.Unlock()
	if !initialized[dsn] {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS content_records (namespace VARCHAR(64) NOT NULL, record_key VARCHAR(255) NOT NULL, payload LONGBLOB NOT NULL, PRIMARY KEY(namespace,record_key)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin`); err != nil {
			db.Close()
			return nil, err
		}
		if err := EnsureStoreSchema(db); err != nil {
			db.Close()
			return nil, err
		}
		initialized[dsn] = true
	}
	return db, nil
}

// ReadRecord retrieves one opaque domain record, preserving sql.ErrNoRows.
func ReadRecord(dir, namespace, key string) ([]byte, error) {
	return (Records{Dir: dir}).Read(namespace, key)
}

// WriteRecord atomically stores the complete current domain value.
func WriteRecord(dir, namespace, key string, payload []byte) error {
	return (Records{Dir: dir}).Write(namespace, key, payload)
}

// DeleteRecord is idempotent for absent values.
func DeleteRecord(dir, namespace, key string) error {
	return (Records{Dir: dir}).Delete(namespace, key)
}

// ListRecords returns a detached copy of every value in a domain namespace.
func ListRecords(dir, namespace string) (map[string][]byte, error) {
	return (Records{Dir: dir}).List(namespace)
}
