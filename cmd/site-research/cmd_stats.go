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

func newStatsCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Imprime métricas do catálogo",
		Long:  "Lê catalog.json e relata distribuição por profundidade, por page_type, páginas sem mini_summary, anexos detectados etc.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			logger := logging.New(logging.Config{Level: logLevel, Format: cfg.Logging.Format})
			return app.Stats(context.Background(), logger, cfg, app.StatsOptions{
				Output: os.Stdout, Format: format,
			})
		},
	}
	cmd.Flags().StringVar(&format, "format", "text", "output format: text|json")
	return cmd
}
