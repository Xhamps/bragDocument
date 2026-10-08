package main

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
)

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply pending database migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := boot()
			if err != nil {
				return err
			}
			if err := postgres.Migrate(cfg.DatabaseURL); err != nil {
				return err
			}
			slog.InfoContext(cmd.Context(), "migrations applied")
			return nil
		},
	}
}
