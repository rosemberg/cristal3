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

func newInspectCmd() *cobra.Command {
	var full bool
	cmd := &cobra.Command{
		Use:   "inspect <path|url>",
		Short: "Inspeciona uma entrada do catálogo",
		Long:  "Carrega o _index.json da página indicada (path relativo ao escopo OU URL absoluta) e imprime um resumo ou o JSON completo (--full).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			logger := logging.New(logging.Config{Level: logLevel, Format: cfg.Logging.Format})
			return app.Inspect(context.Background(), logger, cfg, app.InspectOptions{
				Target: args[0], Full: full, Output: os.Stdout,
			})
		},
	}
	cmd.Flags().BoolVar(&full, "full", false, "print full Page JSON instead of a compact summary")
	return cmd
}
