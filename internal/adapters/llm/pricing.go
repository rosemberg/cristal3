package llm

import "strings"

// pricingEntry holds per-million-token USD prices for input and output tokens.
type pricingEntry struct {
	inputPerMTok  float64 // USD per 1M input tokens
	outputPerMTok float64 // USD per 1M output tokens
}

// pricingTable maps provider:model patterns to pricing entries.
// Model matching is done by prefix on the model name (after lower-casing).
var pricingTable = []struct {
	provider string
	prefix   string
	entry    pricingEntry
}{
	// Anthropic Claude Haiku 4.5 — input $1/MTok, output $5/MTok
	{"anthropic", "claude-haiku-4-5", pricingEntry{inputPerMTok: 1.0, outputPerMTok: 5.0}},
	// Anthropic Claude Haiku 3.x — input $0.25/MTok, output $1.25/MTok
	{"anthropic", "claude-haiku", pricingEntry{inputPerMTok: 0.25, outputPerMTok: 1.25}},
	// Anthropic Claude Sonnet 4.x — input $3/MTok, output $15/MTok
	{"anthropic", "claude-sonnet-4", pricingEntry{inputPerMTok: 3.0, outputPerMTok: 15.0}},
	// Anthropic Claude Sonnet 3.x — input $3/MTok, output $15/MTok
	{"anthropic", "claude-sonnet", pricingEntry{inputPerMTok: 3.0, outputPerMTok: 15.0}},
	// Anthropic Claude Opus — input $15/MTok, output $75/MTok
	{"anthropic", "claude-opus", pricingEntry{inputPerMTok: 15.0, outputPerMTok: 75.0}},
	// Google Gemini 2.0 Flash — input $0.10/MTok, output $0.40/MTok
	{"google", "gemini-2.0-flash", pricingEntry{inputPerMTok: 0.10, outputPerMTok: 0.40}},
	{"gemini", "gemini-2.0-flash", pricingEntry{inputPerMTok: 0.10, outputPerMTok: 0.40}},
	// Google Gemini 1.5 Flash
	{"google", "gemini-1.5-flash", pricingEntry{inputPerMTok: 0.075, outputPerMTok: 0.30}},
	{"gemini", "gemini-1.5-flash", pricingEntry{inputPerMTok: 0.075, outputPerMTok: 0.30}},
}

// EstimateCost returns a best-effort USD cost for the given token counts.
// Heuristic pricing table; returns 0 for unknown providers or models.
// For claude-haiku-4-5: input $1/MTok, output $5/MTok.
// For claude-sonnet-4-*: input $3/MTok, output $15/MTok.
// For gemini-2.0-flash: input $0.10/MTok, output $0.40/MTok.
// Unknown: 0.
func EstimateCost(provider, model string, tokensIn, tokensOut int64) float64 {
	provLower := strings.ToLower(provider)
	modelLower := strings.ToLower(model)

	for _, e := range pricingTable {
		if e.provider == provLower && strings.HasPrefix(modelLower, e.prefix) {
			inputCost := float64(tokensIn) / 1_000_000.0 * e.entry.inputPerMTok
			outputCost := float64(tokensOut) / 1_000_000.0 * e.entry.outputPerMTok
			return inputCost + outputCost
		}
	}
	return 0
}
