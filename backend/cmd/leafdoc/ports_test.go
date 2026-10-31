// 本文件用替换后的根动作验证监听参数校验，避免测试真正启动或释放服务端口。
package main

import (
	"context"
	"io"
	"testing"

	"github.com/urfave/cli/v3"
)

// TestFixedPortFlags 验证替代端口、临时端口和 Unix 套接字在解析阶段失败。
// 根动作一旦被调用就使测试失败，保证错误输入没有进入真实启动流程。
func TestFixedPortFlags(t *testing.T) {
	for _, args := range [][]string{{"--port=0"}, {"--port=8080"}, {"--metrics-port=0"}, {"--metrics-port=9090"}, {"--unix-socket=/tmp/leafdoc.sock"}} {
		t.Run(args[0], func(t *testing.T) {
			cmd := newRootCommand()
			cmd.Writer = io.Discard
			cmd.ErrWriter = io.Discard
			cmd.Action = func(context.Context, *cli.Command) error { t.Fatal("invalid listener reached startup"); return nil }
			if err := cmd.Run(context.Background(), append([]string{"leafdoc"}, args...)); err == nil {
				t.Fatal("accepted alternate listener")
			}
		})
	}
	for _, key := range []string{"LEAFDOC_PORT", "LEAFDOC_METRICS_PORT"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "0")
			cmd := newRootCommand()
			cmd.Writer = io.Discard
			cmd.ErrWriter = io.Discard
			cmd.Action = func(context.Context, *cli.Command) error { t.Fatal("invalid environment reached startup"); return nil }
			if err := cmd.Run(context.Background(), []string{"leafdoc"}); err == nil {
				t.Fatal("accepted ephemeral environment")
			}
		})
	}
}

// TestFixedListenerFlagsAcceptTheirAllocation 同时设置合法环境值并传入合法参数，
// 确认后端与指标固定分配都被命令接受。替换动作只返回成功，不建立监听。
func TestFixedListenerFlagsAcceptTheirAllocation(t *testing.T) {
	t.Setenv("LEAFDOC_PORT", "24692")
	t.Setenv("LEAFDOC_METRICS_PORT", "24694")
	for _, args := range [][]string{{"--port=24692"}, {"--metrics-port=24694"}} {
		t.Run(args[0], func(t *testing.T) {
			command := newRootCommand()
			command.Writer, command.ErrWriter = io.Discard, io.Discard
			command.Action = func(context.Context, *cli.Command) error { return nil }
			if err := command.Run(context.Background(), append([]string{"leafdoc"}, args...)); err != nil {
				t.Fatalf("fixed assignment rejected: %v", err)
			}
		})
	}
}
