package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/logging"
)

func newCrawlCmd() *cobra.Command {
	var (
		dryRun     bool
		fromFile   string
		maxURLs    int
		purgeStale bool
		confirm    bool
	)
	cmd := &cobra.Command{
		Use:   "crawl",
		Short: "Executa crawl das URLs do sitemap",
		Long:  "Baixa todas as URLs descobertas no escopo do sitemap e armazena as páginas brutas.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			logger := logging.New(logging.Config{Level: logLevel, Format: cfg.Logging.Format})
			report, err := app.Crawl(context.Background(), logger, cfg, app.CrawlOptions{
				FromFile:   fromFile,
				DryRun:     dryRun,
				MaxURLs:    maxURLs,
				PurgeStale: purgeStale,
				Confirm:    confirm,
			})
			_ = report
			return err
		},
	}
	cmd.Flags().StringVar(&fromFile, "from-file", "", "read sitemap from local file instead of configured URL")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "fetch and extract without writing to store")
	cmd.Flags().IntVar(&maxURLs, "max-urls", 0, "maximum number of URLs to process (0 = unlimited)")
	cmd.Flags().BoolVar(&purgeStale, "purge-stale", false, "delete pages marked stale longer than cfg.recrawl.stale_retention_days (requires --confirm)")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "authorize destructive operations such as --purge-stale")
	return cmd
}
