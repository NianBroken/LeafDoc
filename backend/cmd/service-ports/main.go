// service-ports 命令向源码工具提供共享监听释放能力，直接使用统一端口策略。
package main

import (
	"fmt"
	"os"

	"github.com/NianBroken/LeafDoc/internal/core/serviceports"
)

// main 将可执行文件名后的全部参数作为服务名称交给 Release。
// 失败时原样写入标准错误并以状态一结束，成功时不输出内容。
func main() {
	if err := serviceports.Release(os.Args[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
