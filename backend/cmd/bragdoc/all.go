package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// allCmd runs api, bot and worker in one process (PRD-0008). No migrations.
func allCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Run the api, bot and worker together",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := boot()
			if err != nil {
				return err
			}
			return runServices(cmd.Context(),
				service{"api", func(ctx context.Context) error { return runAPI(ctx, cfg) }},
				service{"bot", func(ctx context.Context) error { return runBot(ctx, cfg) }},
				service{"worker", func(ctx context.Context) error { return runWorker(ctx, cfg) }},
			)
		},
	}
}

type service struct {
	name string
	run  func(context.Context) error
}

// runServices runs each service until ctx is done. The first error cancels the
// others, which shut down as they do alone; it is returned prefixed with the
// failing service's name.
func runServices(ctx context.Context, svcs ...service) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, s := range svcs {
		g.Go(func() error {
			if err := s.run(ctx); err != nil {
				return fmt.Errorf("%s: %w", s.name, err)
			}
			return nil
		})
	}
	return g.Wait()
}
