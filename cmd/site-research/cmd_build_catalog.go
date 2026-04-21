package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/logging"
)

func newBuildCatalogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "build-catalog",
		Short: "Consolida árvore em catalog.json + SQLite/FTS",
		Long:  "Lê a árvore de _index.json em cfg.Storage.DataDir, gera cfg.Storage.CatalogPath (JSON) e reconstrói cfg.Storage.SQLitePath (SQLite+FTS) do zero.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			logger := logging.New(logging.Config{Level: logLevel, Format: cfg.Logging.Format})
			return app.BuildCatalog(context.Background(), logger, cfg, app.BuildCatalogOptions{})
		},
	}
}
