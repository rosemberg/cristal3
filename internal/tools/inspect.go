package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/format"
)

type inspectArgs struct {
	Target string `json:"target"`
}

// inspectHandler returns a Handler for the "inspect_page" tool.
func inspectHandler(cfg *config.Config, logger *slog.Logger) Handler {
	return func(ctx context.Context, raw json.RawMessage) (CallToolResult, error) {
		var args inspectArgs
		if err := json.Unmarshal(raw, &args); err != nil {
			return errorResult(fmt.Sprintf("**Erro:** argumentos inválidos: %v\n\nForneça `{\"target\": \"<url-ou-path>\"}`.", err)), nil
		}
		if strings.TrimSpace(args.Target) == "" {
			return errorResult("**Erro:** o campo `target` é obrigatório e não pode ser vazio.\n\nExemplo: `{\"target\": \"contabilidade/balancetes\"}`"), nil
		}

		// Defense-in-depth: reject traversal attempts before reaching fsstore.
		if strings.Contains(args.Target, "..") {
			return errorResult(fmt.Sprintf(
				"**Erro:** o target %q contém sequência inválida `..`.\n\nForneça um path relativo válido ou URL completa.", args.Target,
			)), nil
		}

		page, err := app.InspectPage(ctx, logger, cfg, args.Target)
		if err != nil {
			if errors.Is(err, app.ErrPageNotFound) {
				return errorResult(fmt.Sprintf(
					"**Erro:** Página não encontrada no catálogo: %s.\n\nUse `search` para descobrir URLs válidas.", args.Target,
				)), nil
			}
			logger.Error("inspect_page handler error", "target", args.Target, "err", err)
			return errorResult(fmt.Sprintf("**Erro:** falha ao inspecionar página: %v", err)), nil
		}

		logger.Info("inspect_page tool complete", "url", page.URL)

		md := format.RenderPage(page)
		return okResult(md), nil
	}
}
