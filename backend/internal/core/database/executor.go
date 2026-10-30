package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
)

// Executor binds domain queries to either a pool or one request transaction.
type Executor interface {
	Exec(string, ...any) (sql.Result, error)
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
	Prepare(string) (*sql.Stmt, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	PrepareContext(context.Context, string) (*sql.Stmt, error)
}

// Transaction preserves the owning unit of work when domain methods compose.
type Transaction interface {
	Executor
	Commit() error
	Rollback() error
}
type borrowedTransaction struct{ Executor }

func (borrowedTransaction) Commit() error   { return nil }
func (borrowedTransaction) Rollback() error { return nil }

// BeginExecutor owns pool transactions and borrows transactions supplied by callers.
func BeginExecutor(e Executor) (Transaction, error) {
	if db, ok := e.(*sql.DB); ok {
		return db.Begin()
	}
	if e == nil {
		return nil, fmt.Errorf("database executor is required")
	}
	return borrowedTransaction{e}, nil
}

// Records provides opaque domain records through a request-bound executor.
type Records struct {
	Dir      string
	Executor Executor
}

func (r Records) Connection() (Executor, error) {
	if r.Executor != nil {
		return r.Executor, nil
	}
	return Shared(r.Dir)
}
func (r Records) Read(namespace, key string) ([]byte, error) {
	e, err := r.Connection()
	if err != nil {
		return nil, err
	}
	var raw []byte
	err = e.QueryRow("SELECT payload FROM content_records WHERE namespace=? AND record_key=?", namespace, key).Scan(&raw)
	return raw, err
}
func (r Records) Write(namespace, key string, raw []byte) error {
	e, err := r.Connection()
	if err != nil {
		return err
	}
	_, err = e.Exec("INSERT INTO content_records(namespace,record_key,payload) VALUES(?,?,?) ON DUPLICATE KEY UPDATE payload=VALUES(payload)", namespace, key, raw)
	return err
}
func (r Records) Delete(namespace, key string) error {
	e, err := r.Connection()
	if err != nil {
		return err
	}
	_, err = e.Exec("DELETE FROM content_records WHERE namespace=? AND record_key=?", namespace, key)
	return err
}
func (r Records) List(namespace string) (result map[string][]byte, resultErr error) {
	e, err := r.Connection()
	if err != nil {
		return nil, err
	}
	rows, err := e.Query("SELECT record_key,payload FROM content_records WHERE namespace=?", namespace)
	if err != nil {
		return nil, err
	}
	defer CloseResource(rows, &resultErr)
	out := map[string][]byte{}
	for rows.Next() {
		var key string
		var raw []byte
		if err = rows.Scan(&key, &raw); err != nil {
			return nil, err
		}
		out[key] = raw
	}
	return out, rows.Err()
}

// rollback releases an unfinished SQL transaction without treating commit as a failure.
func rollback(tx Transaction) error {
	err := tx.Rollback()
	if errors.Is(err, sql.ErrTxDone) {
		return nil
	}
	return err
}

// Rollback joins cleanup failures with the operation's existing error.
func Rollback(tx Transaction, resultErr *error) {
	*resultErr = errors.Join(*resultErr, rollback(tx))
}

// CloseResource preserves both the operation failure and the resource release failure.
func CloseResource(resource io.Closer, resultErr *error) {
	*resultErr = errors.Join(*resultErr, resource.Close())
}
