// Command site-research is the CLI entrypoint for the Phase 1 crawler + catalog tool.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgPath  string
	logLevel string
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "site-research",
		Short:         "Crawler + catálogo do portal de transparência do TRE-PI (Fase 1)",
		Long:          "site-research descobre, crawleia, consolida e pesquisa o subsite /transparencia-e-prestacao-de-contas do portal TRE-PI.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	defaultCfg := os.Getenv("SITE_RESEARCH_CONFIG")
	if defaultCfg == "" {
		defaultCfg = "./config.yaml"
	}
	root.PersistentFlags().StringVar(&cfgPath, "config", defaultCfg, "path to config file (YAML)")
	root.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level: debug|info|warn|error")

	root.AddCommand(newDiscoverCmd())
	root.AddCommand(newCrawlCmd())
	root.AddCommand(newSummarizeCmd())
	root.AddCommand(newBuildCatalogCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newInspectCmd())
	root.AddCommand(newStatsCmd())
	return root
}
