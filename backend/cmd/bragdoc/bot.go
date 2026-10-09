package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/xhamps/bragdocument/backend/internal/adapters/llm"
	"github.com/xhamps/bragdocument/backend/internal/adapters/postgres"
	"github.com/xhamps/bragdocument/backend/internal/adapters/redis"
	"github.com/xhamps/bragdocument/backend/internal/adapters/telegram"
	"github.com/xhamps/bragdocument/backend/internal/app"
	"github.com/xhamps/bragdocument/backend/internal/ports"
	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// botLLMTimeout keeps a bot reply under PRD-0003 NFR-1 (3 s).
const botLLMTimeout = 2 * time.Second

func botCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bot",
		Short: "Run the Telegram bot (long polling)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			cfg, err := boot()
			if err != nil {
				return err
			}
			if cfg.TelegramToken == "" {
				slog.WarnContext(ctx, "TELEGRAM_BOT_TOKEN not set; bot idle")
				<-ctx.Done()
				return nil
			}
			if cfg.TelegramMode != "polling" {
				return fmt.Errorf("TELEGRAM_MODE %q not supported yet; use polling", cfg.TelegramMode)
			}

			db, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DBTimeout)
			if err != nil {
				return err
			}
			defer db.Close()
			rc, err := redis.Connect(ctx, cfg.RedisURL, cfg.CacheTimeout)
			if err != nil {
				return err
			}
			defer func() { _ = rc.Close() }()

			reg := telemetry.NewRegistry() // ponytail: not served; the bot has no /metrics endpoint yet
			var impact ports.ImpactExtractor = llm.Disabled{}
			if cfg.OpenAIAPIKey != "" {
				impact = llm.NewOpenAIExtractor(cfg.OpenAIAPIKey, cfg.OpenAIModel, min(cfg.LLMTimeout, botLLMTimeout), reg)
			}
			docs := postgres.NewDocumentRepo(db)
			cache := redis.NewDegrading(rc, reg)
			logs := app.NewLogs(docs, postgres.NewLogRepo(db), impact, cache)
			// codes: raw rc so a Redis outage fails linking loudly; undo: Degrading (a miss is harmless).
			uc := app.NewTelegram(postgres.NewTelegramLinkRepo(db), docs, logs, rc, cache, cfg.AppURL, telemetry.WithTenantID)

			b, err := telegram.New(ctx, cfg.TelegramToken, uc)
			if err != nil {
				return err
			}
			slog.InfoContext(ctx, "bot polling")
			b.Run(ctx)
			slog.InfoContext(ctx, "bot stopped")
			return nil
		},
	}
}
