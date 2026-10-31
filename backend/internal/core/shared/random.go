// 本文件提供短标识生成和从固定字符集合抽取密码字符的共用函数。
package shared

import (
	"crypto/rand"

	"math/big"

	"github.com/teris-io/shortid"
)

// 密码的每个字节都从此 ASCII 集合独立选择，不保证每种字符类别都出现。
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"

// GenerateUniqueID 调用 shortid 生成短标识，将生成错误原样返回。
// 此函数自身不查询存储，也不进行已有标识的冲突检查。
func GenerateUniqueID() (string, error) {
	id, err := shortid.Generate()
	if err != nil {
		return "", err
	}

	return id, nil
}

// GenerateRandomPassword 生成指定字节长度的 ASCII 密码，随机来源为 crypto/rand。
// 长度为零时返回空字符串，负数长度会由切片分配触发 panic，本函数不校验长度。
// 任何一次随机读取失败都会返回空字符串和错误，不返回已生成的部分密码。
func GenerateRandomPassword(length int) (string, error) {
	password := make([]byte, length)
	// rand.Int 在字符下标范围内均匀取值，每轮只决定当前位置的一个字符。
	for i := range password {
		// 上界不包含字符集合长度，因此随机下标始终落在切片可访问范围内。
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[n.Int64()]
	}
	return string(password), nil
}
