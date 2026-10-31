// 本文件定义数据库执行接口、事务借用规则以及通用记录表的读写。
// 业务存储可使用连接池或外部事务，事务所有权由 BeginExecutor 明确区分。
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
)

// Executor 统一连接池与事务提供的查询和预编译能力，保留带上下文及不带上下文的方法。
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

// Transaction 在执行接口上提供提交与回滚入口，借用视图不会结束外层事务。
type Transaction interface {
	Executor
	Commit() error
	Rollback() error
}

// borrowedTransaction 转交全部查询，只把结束事务的两个操作实现为空操作。
type borrowedTransaction struct{ Executor }

func (borrowedTransaction) Commit() error   { return nil }
func (borrowedTransaction) Rollback() error { return nil }

// BeginExecutor 仅对具体类型 *sql.DB 开启真实事务，nil 接口返回错误。
// 其他非空执行器一律包装为借用视图，不检查其底层是否确实为事务。
func BeginExecutor(e Executor) (Transaction, error) {
	if db, ok := e.(*sql.DB); ok {
		return db.Begin()
	}
	if e == nil {
		return nil, fmt.Errorf("database executor is required")
	}
	return borrowedTransaction{e}, nil
}

// Records 按命名空间和记录键存取不透明字节载荷，不负责载荷编码或业务校验。
// Executor 非空时共用外层执行器，否则通过 Dir 获取共享数据库连接。
type Records struct {
	Dir      string
	Executor Executor
}

// Connection 优先返回显式执行器，未绑定时读取目录对应的共享连接。
func (r Records) Connection() (Executor, error) {
	if r.Executor != nil {
		return r.Executor, nil
	}
	return Shared(r.Dir)
}

// Read 查询单个载荷，没有匹配记录时保留 sql.ErrNoRows，不合成空记录。
func (r Records) Read(namespace, key string) ([]byte, error) {
	e, err := r.Connection()
	if err != nil {
		return nil, err
	}
	var raw []byte
	err = e.QueryRow("SELECT payload FROM content_records WHERE namespace=? AND record_key=?", namespace, key).Scan(&raw)
	return raw, err
}

// Write 按命名空间与记录键插入载荷，唯一键存在时覆盖其 payload。
func (r Records) Write(namespace, key string, raw []byte) error {
	e, err := r.Connection()
	if err != nil {
		return err
	}
	_, err = e.Exec("INSERT INTO content_records(namespace,record_key,payload) VALUES(?,?,?) ON DUPLICATE KEY UPDATE payload=VALUES(payload)", namespace, key, raw)
	return err
}

// Delete 删除指定记录，不检查受影响行数，因此不存在的记录也视为成功。
func (r Records) Delete(namespace, key string) error {
	e, err := r.Connection()
	if err != nil {
		return err
	}
	_, err = e.Exec("DELETE FROM content_records WHERE namespace=? AND record_key=?", namespace, key)
	return err
}

// List 读取命名空间下全部记录并以记录键建立映射，没有行时返回空映射。
// 扫描或遍历失败返回错误，命名返回值还会接收延迟关闭结果集时的错误。
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

// rollback 执行回滚，把已提交或已回滚的 sql.ErrTxDone 当作正常结束。
// 其他回滚失败原样返回，借用事务的回滚则始终成功。
func rollback(tx Transaction) error {
	err := tx.Rollback()
	if errors.Is(err, sql.ErrTxDone) {
		return nil
	}
	return err
}

// Rollback 把回滚失败与调用方已有错误合并，不覆盖此前失败原因。
// resultErr 必须是有效指针，通常在带命名错误返回值的函数中延迟调用。
func Rollback(tx Transaction, resultErr *error) {
	*resultErr = errors.Join(*resultErr, rollback(tx))
}

// CloseResource 关闭资源，并把关闭错误合并到已有返回错误中。
// 资源与结果指针由调用方保证有效，此函数不忽略关闭失败。
func CloseResource(resource io.Closer, resultErr *error) {
	*resultErr = errors.Join(*resultErr, resource.Close())
}
