package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeYAML writes content to a file named "config.yaml" in dir and returns its path.
func writeYAML(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeYAML: %v", err)
	}
	return path
}

// TestLoad_FileNotFound verifies that loading a non-existent file returns an error
// whose message contains "read config".
func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read config") {
		t.Fatalf("expected error to contain %q, got: %v", "read config", err)
	}
}

// TestLoad_MalformedYAML verifies that intentionally-broken YAML returns an error
// whose message contains "parse config yaml".
func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, "not a valid: yaml: :::\n")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parse config yaml") {
		t.Fatalf("expected error to contain %q, got: %v", "parse config yaml", err)
	}
}

// TestLoad_MissingSeedURL verifies that a valid YAML without scope.seed_url returns
// an error containing "scope.seed_url is required".
func TestLoad_MissingSeedURL(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, `
sitemap:
  url: "https://example.com/sitemap.xml"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "scope.seed_url is required") {
		t.Fatalf("expected error to contain %q, got: %v", "scope.seed_url is required", err)
	}
}

// TestLoad_MissingPrefix verifies that a YAML with seed_url but no prefix returns
// an error containing "scope.prefix is required".
func TestLoad_MissingPrefix(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, `
scope:
  seed_url: "https://example.com/start"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "scope.prefix is required") {
		t.Fatalf("expected error to contain %q, got: %v", "scope.prefix is required", err)
	}
}

// TestLoad_AppliesDefaults verifies that a minimal valid YAML gets all expected defaults.
func TestLoad_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, `
scope:
  seed_url: "https://example.com/start"
  prefix:   "https://example.com"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	check := func(label, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %q, want %q", label, got, want)
		}
	}
	checkInt := func(label string, got, want int) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %d, want %d", label, got, want)
		}
	}
	checkFloat := func(label string, got, want float64) {
		t.Helper()
		if got != want {
			t.Errorf("%s: got %f, want %f", label, got, want)
		}
	}

	check("Crawler.UserAgent", cfg.Crawler.UserAgent, "TRE-PI-Research-Crawler/0.1 (+contact: cotdi@tre-pi.jus.br)")
	checkFloat("Crawler.RateLimitPerSecond", cfg.Crawler.RateLimitPerSecond, 1.0)
	checkInt("Crawler.JitterMS", cfg.Crawler.JitterMS, 200)
	checkInt("Crawler.RequestTimeoutSeconds", cfg.Crawler.RequestTimeoutSeconds, 30)
	checkInt("Crawler.MaxRetries", cfg.Crawler.MaxRetries, 3)
	checkInt("Crawler.CircuitBreaker.MaxConsecutiveFailures", cfg.Crawler.CircuitBreaker.MaxConsecutiveFailures, 5)
	checkInt("Crawler.CircuitBreaker.PauseMinutes", cfg.Crawler.CircuitBreaker.PauseMinutes, 10)
	checkInt("Crawler.CircuitBreaker.AbortThreshold", cfg.Crawler.CircuitBreaker.AbortThreshold, 3)
	checkInt("Crawler.SuspiciousResponse.MinBodyBytes", cfg.Crawler.SuspiciousResponse.MinBodyBytes, 500)

	if got := len(cfg.Crawler.SuspiciousResponse.BlockTitlePatterns); got != 5 {
		t.Errorf("BlockTitlePatterns: got %d entries, want 5", got)
	}
	foundCloudflare := false
	for _, p := range cfg.Crawler.SuspiciousResponse.BlockTitlePatterns {
		if p == "Cloudflare" {
			foundCloudflare = true
			break
		}
	}
	if !foundCloudflare {
		t.Errorf("BlockTitlePatterns: expected to contain %q", "Cloudflare")
	}

	check("Storage.DataDir", cfg.Storage.DataDir, "./data")
	check("Storage.CatalogPath", cfg.Storage.CatalogPath, "./data/catalog.json")
	check("Storage.SQLitePath", cfg.Storage.SQLitePath, "./data/catalog.sqlite")

	check("LLM.Provider", cfg.LLM.Provider, "anthropic")
	check("LLM.Model", cfg.LLM.Model, "claude-haiku-4-5")
	check("LLM.Endpoint", cfg.LLM.Endpoint, "https://api.anthropic.com")
	check("LLM.APIKeyEnv", cfg.LLM.APIKeyEnv, "ANTHROPIC_API_KEY")
	checkInt("LLM.Concurrency", cfg.LLM.Concurrency, 3)
	checkInt("LLM.RequestTimeoutSeconds", cfg.LLM.RequestTimeoutSeconds, 60)

	checkInt("Recrawl.StaleRetentionDays", cfg.Recrawl.StaleRetentionDays, 30)

	check("Logging.Level", cfg.Logging.Level, "info")
	check("Logging.Format", cfg.Logging.Format, "json")
}

// TestLoad_RespectsExplicitValues verifies that explicitly-set YAML values are preserved
// and not overwritten by defaults, while default-only fields are still populated.
func TestLoad_RespectsExplicitValues(t *testing.T) {
	dir := t.TempDir()
	path := writeYAML(t, dir, `
scope:
  seed_url: "https://example.com/start"
  prefix:   "https://example.com"
crawler:
  rate_limit_per_second: 0.5
llm:
  provider: "gemini"
  model:    "gemini-2.0-flash"
storage:
  data_dir: "/custom/data"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Explicit values must be preserved.
	if cfg.Crawler.RateLimitPerSecond != 0.5 {
		t.Errorf("RateLimitPerSecond: got %f, want 0.5", cfg.Crawler.RateLimitPerSecond)
	}
	if cfg.LLM.Provider != "gemini" {
		t.Errorf("LLM.Provider: got %q, want %q", cfg.LLM.Provider, "gemini")
	}
	if cfg.LLM.Model != "gemini-2.0-flash" {
		t.Errorf("LLM.Model: got %q, want %q", cfg.LLM.Model, "gemini-2.0-flash")
	}
	if cfg.Storage.DataDir != "/custom/data" {
		t.Errorf("Storage.DataDir: got %q, want %q", cfg.Storage.DataDir, "/custom/data")
	}

	// Default-only fields must still be populated.
	if cfg.LLM.Endpoint == "" {
		t.Errorf("LLM.Endpoint: expected default to be applied, got empty string")
	}
	if cfg.Crawler.UserAgent == "" {
		t.Errorf("Crawler.UserAgent: expected default to be applied, got empty string")
	}
}

// TestLoad_ConfigExampleIsValid verifies that the repo's config.yaml.example loads
// without error and contains the expected canonical values.
func TestLoad_ConfigExampleIsValid(t *testing.T) {
	path := "/Users/rosemberg/projetos-gemini/cristal3/config.yaml.example"
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error loading config.yaml.example: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil *Config, got nil")
	}
	wantSeedURL := "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas"
	if cfg.Scope.SeedURL != wantSeedURL {
		t.Errorf("Scope.SeedURL: got %q, want %q", cfg.Scope.SeedURL, wantSeedURL)
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("LLM.Provider: got %q, want %q", cfg.LLM.Provider, "anthropic")
	}
}
