package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/bergmaia/site-research/internal/adapters/llm"
	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain/ports"
	"github.com/bergmaia/site-research/internal/logging"
)

func newSummarizeCmd() *cobra.Command {
	var force bool
	var maxPages int

	cmd := &cobra.Command{
		Use:   "summarize",
		Short: "Gera mini_summaries para páginas pendentes",
		Long:  "Percorre o catálogo local e, para cada página sem mini_summary (ou com source_hash divergente), chama o provider LLM configurado.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			logger := logging.New(logging.Config{Level: logLevel, Format: cfg.Logging.Format})
			provider, err := buildLLMProvider(cfg)
			if err != nil {
				return fmt.Errorf("build provider: %w", err)
			}
			_, err = app.Summarize(context.Background(), logger, cfg, app.SummarizeOptions{Force: force, MaxPages: maxPages}, provider)
			return err
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "regenerate mini_summary even if source_hash matches")
	cmd.Flags().IntVar(&maxPages, "max-pages", 0, "maximum pages to process (0 = unlimited)")

	return cmd
}

// buildLLMProvider constructs a ports.LLMProvider from configuration.
func buildLLMProvider(cfg *config.Config) (ports.LLMProvider, error) {
	switch cfg.LLM.Provider {
	case "anthropic":
		key := os.Getenv(cfg.LLM.APIKeyEnv)
		if key == "" {
			return nil, fmt.Errorf("env var %s is not set", cfg.LLM.APIKeyEnv)
		}
		return llm.NewAnthropic(llm.AnthropicOptions{
			APIKey:   key,
			Model:    cfg.LLM.Model,
			Endpoint: cfg.LLM.Endpoint,
			Timeout:  time.Duration(cfg.LLM.RequestTimeoutSeconds) * time.Second,
		})
	case "mock":
		return llm.NewMockProvider(llm.MockOptions{
			Name:  "mock",
			Model: "mock-model",
			Responses: []llm.MockResponse{
				{Response: &ports.GenerateResponse{
					Text:         "mini summary mock",
					TokensInput:  10,
					TokensOutput: 15,
					Provider:     "mock",
					Model:        "mock-model",
				}},
			},
			Loop: true,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported llm.provider: %q", cfg.LLM.Provider)
	}
}
