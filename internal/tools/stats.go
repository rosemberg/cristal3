package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/format"
)

// statsHandler returns a Handler for the "catalog_stats" tool.
func statsHandler(cfg *config.Config, logger *slog.Logger) Handler {
	return func(ctx context.Context, raw json.RawMessage) (CallToolResult, error) {
		// catalog_stats takes no arguments, but we validate that no unknown
		// fields were sent (additionalProperties: false in the schema).
		if len(raw) > 0 {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(raw, &obj); err != nil {
				return errorResult(fmt.Sprintf("**Erro:** argumentos inválidos: %v\n\nEsta tool não aceita argumentos.", err)), nil
			}
			// Any key present is unexpected since the schema has no properties.
			// We tolerate empty objects {} to be lenient.
		}

		report, err := app.GetStats(ctx, logger, cfg)
		if err != nil {
			logger.Error("catalog_stats handler error", "err", err)
			return errorResult(fmt.Sprintf("**Erro:** falha ao obter estatísticas do catálogo: %v\n\nVerifique se o catálogo está acessível.", err)), nil
		}

		logger.Info("catalog_stats tool complete", "total_pages", report.TotalPages)

		md := format.RenderStats(report)
		return okResult(md), nil
	}
}
