// Package format contains markdown renderers for MCP tool responses.
// All functions are deterministic: they never call time.Now().
package format

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/domain/ports"
)

const (
	maxSummaryRunes = 500
	maxListItems    = 10
)

// RenderSearchHits formats a slice of SearchHit values as LLM-optimised markdown.
// totalFound is the count before any section filter was applied.
// sectionFilter is the filter string (may be empty).
func RenderSearchHits(query string, hits []ports.SearchHit, totalFound int, limit int, sectionFilter string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Resultados para: %q\n\n", query))

	if len(hits) == 0 {
		b.WriteString("## Nenhum resultado\n\n")
		if sectionFilter != "" {
			b.WriteString(fmt.Sprintf("Nenhuma página encontrada para a consulta %q na seção %q.\n\n", query, sectionFilter))
		} else {
			b.WriteString(fmt.Sprintf("Nenhuma página encontrada para a consulta %q no catálogo.\n\n", query))
		}
		b.WriteString("Sugestões:\n")
		b.WriteString("- Tente termos mais gerais ou variações ortográficas.\n")
		b.WriteString("- Remova filtros de seção, se aplicável.\n")
		b.WriteString("- Verifique se o catálogo foi gerado e contém páginas relevantes.\n")
		b.WriteString("\n---\n")
		b.WriteString("_Use `catalog_stats` para verificar quantas páginas estão indexadas._\n")
		return b.String()
	}

	displayed := len(hits)
	if limit > 0 && displayed > limit {
		displayed = limit
	}

	if sectionFilter != "" {
		b.WriteString(fmt.Sprintf(
			"Foram encontradas **%d páginas** no catálogo correspondendo a essa consulta (exibindo top %d, seção: %q).\n\n",
			totalFound, displayed, sectionFilter,
		))
	} else {
		b.WriteString(fmt.Sprintf(
			"Foram encontradas **%d páginas** no catálogo (exibindo top %d).\n\n",
			totalFound, displayed,
		))
	}

	for i, h := range hits[:displayed] {
		b.WriteString(fmt.Sprintf("## %d. %s\n", i+1, h.Title))
		summary := truncateRunes(h.MiniSummary, maxSummaryRunes)
		if summary != "" {
			b.WriteString(summary + "\n")
		}
		if h.Section != "" {
			b.WriteString(fmt.Sprintf("**Seção:** %s\n", h.Section))
		}
		b.WriteString(fmt.Sprintf("**URL:** %s\n", h.URL))
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	b.WriteString("_Use `inspect_page` para ver detalhes completos de uma página._\n")

	return b.String()
}

// RenderPage formats a *domain.Page as LLM-optimised markdown.
func RenderPage(page *domain.Page) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# %s\n\n", page.Title))

	b.WriteString(fmt.Sprintf("**URL:** %s\n", page.URL))
	if page.Section != "" {
		b.WriteString(fmt.Sprintf("**Seção:** %s\n", page.Section))
	}
	b.WriteString(fmt.Sprintf("**Tipo:** %s\n", string(page.PageType)))
	b.WriteString(fmt.Sprintf("**Profundidade:** %d\n", page.Metadata.Depth))
	b.WriteString("\n")

	// Breadcrumb
	breadcrumb := buildBreadcrumb(page)
	if breadcrumb != "" {
		b.WriteString("## Breadcrumb\n")
		b.WriteString(breadcrumb + "\n\n")
	}

	// Mini-resumo
	miniSummary := page.MiniSummary.Text
	if miniSummary == "" && page.MiniSummary.Skipped != nil {
		miniSummary = fmt.Sprintf("(não gerado — motivo: %s)", *page.MiniSummary.Skipped)
	} else if miniSummary == "" {
		miniSummary = "(não disponível)"
	} else {
		miniSummary = truncateRunes(miniSummary, maxSummaryRunes)
	}
	b.WriteString("## Mini-resumo\n")
	b.WriteString(miniSummary + "\n\n")

	// Children
	if len(page.Links.Children) > 0 {
		shown := page.Links.Children
		extra := 0
		if len(shown) > maxListItems {
			extra = len(shown) - maxListItems
			shown = shown[:maxListItems]
		}
		b.WriteString(fmt.Sprintf("## Páginas filhas (%d)\n", len(page.Links.Children)))
		for _, c := range shown {
			b.WriteString(fmt.Sprintf("- **%s** — %s\n", c.Title, c.URL))
		}
		if extra > 0 {
			b.WriteString(fmt.Sprintf("- … (+ %d mais)\n", extra))
		}
		b.WriteString("\n")
	}

	// Documents
	if len(page.Documents) > 0 {
		shown := page.Documents
		extra := 0
		if len(shown) > maxListItems {
			extra = len(shown) - maxListItems
			shown = shown[:maxListItems]
		}
		b.WriteString(fmt.Sprintf("## Documentos anexos (%d)\n", len(page.Documents)))
		for _, d := range shown {
			b.WriteString(fmt.Sprintf("- **%s** (%s) — %s\n", d.Title, d.Type, d.URL))
		}
		if extra > 0 {
			b.WriteString(fmt.Sprintf("- … (+ %d mais)\n", extra))
		}
		b.WriteString("\n")
	}

	// Metadados
	b.WriteString("## Metadados\n")
	b.WriteString(fmt.Sprintf("- **Extraído em:** %s\n", page.Metadata.ExtractedAt.UTC().Format("2006-01-02 15:04:05 UTC")))
	if page.Dates.PageUpdatedAt != nil {
		b.WriteString(fmt.Sprintf("- **Atualizado em:** %s\n", page.Dates.PageUpdatedAt.UTC().Format("2006-01-02")))
	} else if page.Dates.ContentDate != nil {
		b.WriteString(fmt.Sprintf("- **Data do conteúdo:** %s\n", *page.Dates.ContentDate))
	}
	if page.Metadata.DiscoveredVia != "" {
		b.WriteString(fmt.Sprintf("- **Descoberto via:** %s\n", page.Metadata.DiscoveredVia))
	}
	if page.Metadata.CrawlerVersion != "" {
		b.WriteString(fmt.Sprintf("- **Versão do crawler:** %s\n", page.Metadata.CrawlerVersion))
	}

	b.WriteString("\n---\n")
	b.WriteString("_Use `search` para encontrar outras páginas relacionadas._\n")

	return b.String()
}

// RenderStats formats a StatsReport as LLM-optimised markdown.
func RenderStats(r app.StatsReport) string {
	var b strings.Builder

	b.WriteString("# Catálogo site-research — estatísticas\n\n")
	b.WriteString(fmt.Sprintf("**Raiz:** %s\n", r.RootURL))
	b.WriteString(fmt.Sprintf("**Schema:** v%d\n", r.SchemaVersion))
	b.WriteString(fmt.Sprintf("**Gerado em:** %s\n", r.GeneratedAt))
	b.WriteString("\n")

	b.WriteString("## Totais\n")
	b.WriteString(fmt.Sprintf("- **Páginas:** %d\n", r.TotalPages))
	b.WriteString(fmt.Sprintf("- **Sem mini-resumo:** %d\n", r.PagesWithoutMiniSummary))
	b.WriteString(fmt.Sprintf("- **Com documentos anexos:** %d\n", r.PagesWithDocs))
	b.WriteString(fmt.Sprintf("- **Total de documentos listados:** %d\n", r.TotalDocuments))
	b.WriteString(fmt.Sprintf("- **Páginas stale:** %d\n", r.StalePages))
	b.WriteString("\n")

	// By depth table
	if len(r.ByDepth) > 0 {
		b.WriteString("## Por profundidade\n")
		b.WriteString("| Profundidade | Páginas |\n")
		b.WriteString("| ---: | ---: |\n")
		depthKeys := make([]int, 0, len(r.ByDepth))
		for k := range r.ByDepth {
			depthKeys = append(depthKeys, k)
		}
		sort.Ints(depthKeys)
		for _, d := range depthKeys {
			b.WriteString(fmt.Sprintf("| %d | %d |\n", d, r.ByDepth[d]))
		}
		b.WriteString("\n")
	}

	// By page type
	if len(r.ByPageType) > 0 {
		b.WriteString("## Por tipo de página\n")
		pageTypeOrder := []string{"landing", "article", "listing", "redirect", "empty"}
		printed := map[string]bool{}
		for _, pt := range pageTypeOrder {
			if n, ok := r.ByPageType[pt]; ok {
				b.WriteString(fmt.Sprintf("- **%s:** %d\n", pt, n))
				printed[pt] = true
			}
		}
		// Any extra types not in the canonical order.
		extras := make([]string, 0)
		for pt := range r.ByPageType {
			if !printed[pt] {
				extras = append(extras, pt)
			}
		}
		sort.Strings(extras)
		for _, pt := range extras {
			b.WriteString(fmt.Sprintf("- **%s:** %d\n", pt, r.ByPageType[pt]))
		}
		b.WriteString("\n")
	}

	// Top sections
	if len(r.TopSections) > 0 {
		b.WriteString("## Top seções (por contagem)\n")
		for i, s := range r.TopSections {
			b.WriteString(fmt.Sprintf("%d. %s — %d\n", i+1, s.Section, s.Count))
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	b.WriteString("_Use `search` para pesquisar páginas no catálogo ou `inspect_page` para detalhes de uma página específica._\n")

	return b.String()
}

// truncateRunes truncates s to at most maxRunes runes.
// If truncated, appends "… (+ N mais)" where N is the number of remaining runes.
func truncateRunes(s string, maxRunes int) string {
	n := utf8.RuneCountInString(s)
	if n <= maxRunes {
		return s
	}
	runes := []rune(s)
	remaining := n - maxRunes
	return string(runes[:maxRunes]) + fmt.Sprintf("… (+ %d mais)", remaining)
}

// buildBreadcrumb returns a " › " separated string from PathTitles or Breadcrumb.
func buildBreadcrumb(page *domain.Page) string {
	if len(page.PathTitles) > 0 {
		return strings.Join(page.PathTitles, " › ")
	}
	if len(page.Breadcrumb) > 0 {
		titles := make([]string, 0, len(page.Breadcrumb))
		for _, b := range page.Breadcrumb {
			if b.Title != "" {
				titles = append(titles, b.Title)
			}
		}
		if len(titles) > 0 {
			return strings.Join(titles, " › ")
		}
	}
	return ""
}
