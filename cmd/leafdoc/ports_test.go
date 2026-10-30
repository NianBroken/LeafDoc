package main

import (
	"context"
	"github.com/urfave/cli/v3"
	"io"
	"testing"
)

// TestFixedPortFlags ensures invalid CLI and environment overrides never reach startup.
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
