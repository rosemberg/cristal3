package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/bergmaia/site-research/internal/app"
	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/logging"
)

const (
	fixtureFile  = "/Users/rosemberg/projetos-gemini/cristal3/fixtures/sitemap.xml.gz"
	scopePrefix  = "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas"
	wantInScope  = 573
)

func minCfg() *config.Config {
	cfg := &config.Config{}
	cfg.Scope.SeedURL = scopePrefix
	cfg.Scope.Prefix = scopePrefix
	cfg.Crawler.UserAgent = "test-ua"
	cfg.Crawler.RequestTimeoutSeconds = 10
	cfg.Logging.Format = "json"
	return cfg
}

func TestDiscover_FromFile_ReturnsExpectedInScope(t *testing.T) {
	cfg := minCfg()
	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})

	var buf bytes.Buffer
	err := app.Discover(context.Background(), logger, cfg, app.DiscoverOptions{
		FromFile: fixtureFile,
		Format:   "text",
		Output:   &buf,
	})
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}

	raw := strings.TrimRight(buf.String(), "\n")
	lines := strings.Split(raw, "\n")
	if got := len(lines); got != wantInScope {
		t.Fatalf("got %d in-scope URLs, want %d", got, wantInScope)
	}
	for _, u := range lines[:5] {
		if !strings.HasPrefix(u, scopePrefix) {
			t.Errorf("URL %q does not start with scope prefix", u)
		}
	}
}

func TestDiscover_FromFile_JSONFormat(t *testing.T) {
	cfg := minCfg()
	logger := logging.New(logging.Config{Level: "info", Format: "json", Output: io.Discard})

	var buf bytes.Buffer
	err := app.Discover(context.Background(), logger, cfg, app.DiscoverOptions{
		FromFile: fixtureFile,
		Format:   "json",
		Output:   &buf,
	})
	if err != nil {
		t.Fatalf("Discover error: %v", err)
	}

	var result app.DiscoverResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON unmarshal error: %v", err)
	}
	if result.InScope != wantInScope {
		t.Fatalf("got in_scope=%d, want %d", result.InScope, wantInScope)
	}
	if len(result.URLs) != wantInScope {
		t.Fatalf("got %d URLs in json.urls, want %d", len(result.URLs), wantInScope)
	}
}

func TestDiscover_BadFormat_Errors(t *testing.T) {
	cfg := &config.Config{}
	cfg.Scope.SeedURL = "https://example.com/x"
	cfg.Scope.Prefix = "https://example.com/x"
	logger := logging.New(logging.Config{Level: "error", Format: "json", Output: io.Discard})
	err := app.Discover(context.Background(), logger, cfg, app.DiscoverOptions{Format: "xml"})
	if err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}
