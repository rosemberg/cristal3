package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain/ports"
	"github.com/bergmaia/site-research/internal/format"
)

type searchArgs struct {
	Query   string  `json:"query"`
	Limit   *int    `json:"limit"`
	Section *string `json:"section"`
}

// searchHandler returns a Handler for the "search" tool.
func searchHandler(cfg *config.Config, logger *slog.Logger) Handler {
	return func(ctx context.Context, raw json.RawMessage) (CallToolResult, error) {
		var args searchArgs
		if err := json.Unmarshal(raw, &args); err != nil {
			return errorResult(fmt.Sprintf("**Erro:** argumentos inválidos: %v\n\nForneça pelo menos `{\"query\": \"<termo>\"}`.", err)), nil
		}
		if strings.TrimSpace(args.Query) == "" {
			return errorResult("**Erro:** o campo `query` é obrigatório e não pode ser vazio.\n\nExemplo: `{\"query\": \"balancetes 2025\"}`"), nil
		}

		limit := 10
		if args.Limit != nil {
			limit = *args.Limit
			if limit < 1 {
				limit = 1
			}
			if limit > 50 {
				limit = 50
			}
		}

		// Fetch more hits than needed when a section filter will be applied,
		// so we have enough candidates after filtering.
		fetchLimit := limit
		if args.Section != nil {
			fetchLimit = 200
		}

		hits, err := app.SearchHits(ctx, logger, cfg, args.Query, fetchLimit)
		if err != nil {
			logger.Error("search handler error", "query", args.Query, "err", err)
			return errorResult(fmt.Sprintf("**Erro:** falha ao executar busca: %v\n\nVerifique se o catálogo FTS está acessível.", err)), nil
		}

		totalFound := len(hits)

		// Apply section filter (case-insensitive exact match).
		if args.Section != nil && *args.Section != "" {
			filterSection := strings.ToLower(*args.Section)
			filtered := hits[:0]
			for _, h := range hits {
				if strings.ToLower(h.Section) == filterSection {
					filtered = append(filtered, h)
				}
			}
			hits = filtered
		}

		// Apply effective limit after filtering.
		sectionFilter := ""
		if args.Section != nil {
			sectionFilter = *args.Section
		}

		// Build the effective hit list respecting limit.
		effective := hits
		if len(effective) > limit {
			effective = effective[:limit]
		}

		logger.Info("search tool complete",
			"query", args.Query,
			"total_found", totalFound,
			"hits_after_filter", len(hits),
			"displayed", len(effective),
		)

		md := format.RenderSearchHits(args.Query, effective, totalFound, limit, sectionFilter)
		return okResult(md), nil
	}
}

// filterBySection filters hits by section field (case-insensitive).
// It is exported for use in tests but is not part of the public API.
func filterBySection(hits []ports.SearchHit, section string) []ports.SearchHit {
	low := strings.ToLower(section)
	out := make([]ports.SearchHit, 0, len(hits))
	for _, h := range hits {
		if strings.ToLower(h.Section) == low {
			out = append(out, h)
		}
	}
	return out
}
