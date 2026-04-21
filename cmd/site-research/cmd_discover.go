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

func newDiscoverCmd() *cobra.Command {
	var fromFile, format string
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Descobre URLs do escopo via sitemap",
		Long:  "Baixa o sitemap global, filtra URLs pelo prefixo do escopo configurado e imprime a lista.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			logger := logging.New(logging.Config{Level: logLevel, Format: cfg.Logging.Format})
			return app.Discover(context.Background(), logger, cfg, app.DiscoverOptions{
				FromFile: fromFile,
				Format:   format,
				Output:   os.Stdout,
			})
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "caminho local do sitemap (.xml ou .xml.gz) — substitui sitemap.url da config")
	cmd.Flags().StringVar(&format, "format", "text", "formato de saída: text|json")
	return cmd
}
