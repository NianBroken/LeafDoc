// 安装校验使用回滚事务实际执行读写，避免把仅能连接的账号当作可运行账号。
package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// VerifyWritable 检查表结构和通用记录的增查改删权限，不保留探测业务记录。
func VerifyWritable(ctx context.Context, dir string) (result error) {
	db, err := Open(dir)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, db.Close()) }()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, tx.Rollback()) }()
	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return err
	}
	records := Records{Executor: tx}
	namespace := "installation_probe"
	id := hex.EncodeToString(key)
	if err = records.Write(namespace, id, []byte("initial")); err != nil {
		return err
	}
	if err = records.Write(namespace, id, []byte("verified")); err != nil {
		return err
	}
	content, err := records.Read(namespace, id)
	if err != nil {
		return err
	}
	if string(content) != "verified" {
		return fmt.Errorf("数据库探测读写不一致")
	}
	return records.Delete(namespace, id)
}
