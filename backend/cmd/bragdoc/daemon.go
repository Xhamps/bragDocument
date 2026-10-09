package main

import (
	"log/slog"

	"github.com/spf13/cobra"
)

// daemonCmd builds the worker command, a placeholder that starts, logs, and
// waits for a signal; its loop arrives with PRD-0006.
func daemonCmd(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: "Run the " + name + " process",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if _, err := boot(); err != nil {
				return err
			}
			slog.InfoContext(ctx, name+" started")
			<-ctx.Done()
			slog.InfoContext(ctx, name+" stopped")
			return nil
		},
	}
}
