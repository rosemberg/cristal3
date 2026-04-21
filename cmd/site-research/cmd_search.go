package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/logging"
)

func newSearchCmd() *cobra.Command {
	var limit int
	var format string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Busca textual no catálogo via FTS",
		Long:  "Executa consulta FTS5 sobre title + mini_summary + full_text e imprime top-N hits.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			logger := logging.New(logging.Config{Level: logLevel, Format: cfg.Logging.Format})
			return app.Search(context.Background(), logger, cfg, app.SearchOptions{
				Query:  args[0],
				Limit:  limit,
				Format: format,
				Output: os.Stdout,
			})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 10, "maximum number of hits to return")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text|json")
	return cmd
}
