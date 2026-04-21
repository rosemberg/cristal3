//go:build integration

package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// benchBinPath is set once in TestMain and reused across benchmarks.
var benchBinPath string

// benchFixtureDir holds the pre-created fixture catalog used by benchmarks.
var benchFixtureDir string

func TestMain(m *testing.M) {
	// Build the binary once for all benchmark iterations.
	tmp, err := os.MkdirTemp("", "mcp-bench-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	binPath := filepath.Join(tmp, "site-research-mcp")
	cmd := exec.Command("go", "build", "-o", binPath,
		"github.com/bergmaia/site-research/cmd/site-research-mcp")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("go build failed: " + string(out))
	}
	benchBinPath = binPath

	// Create fixtures once.
	fixtureDir, err := os.MkdirTemp("", "mcp-bench-fixtures-*")
	if err != nil {
		panic("create fixture dir: " + err.Error())
	}
	defer os.RemoveAll(fixtureDir)
	createBenchFixtures(fixtureDir)
	benchFixtureDir = fixtureDir

	os.Exit(m.Run())
}

// createBenchFixtures builds a minimal but valid fixture catalog.
func createBenchFixtures(dir string) {
	const scopePrefix = "https://www.example.com/transparencia"
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		panic(err)
	}

	type row struct{ path, url, title, section, pageType, summary, fullText string }
	rows := []row{
		{"balancetes", scopePrefix + "/balancetes", "Balancetes", "Contabilidade", "landing", "Balancetes contábeis.", "Balancetes mensais."},
		{"diarias", scopePrefix + "/diarias", "Diárias", "RH", "article", "Diárias de servidores.", "Informações sobre diárias."},
	}

	generatedAt := time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC)
	for _, r := range rows {
		pageDir := filepath.Join(dataDir, r.path)
		if err := os.MkdirAll(pageDir, 0o755); err != nil {
			panic(err)
		}
		pageData := map[string]any{
			"schema_version": 2,
			"url":            r.url,
			"canonical_url":  r.url,
			"title":          r.title,
			"section":        r.section,
			"page_type":      r.pageType,
			"mini_summary": map[string]any{
				"text":         r.summary,
				"generated_at": generatedAt.Format(time.RFC3339),
			},
			"metadata": map[string]any{
				"depth": 2, "extracted_at": generatedAt.Format(time.RFC3339),
				"crawler_version": "0.1.0", "discovered_via": "sitemap",
			},
			"links":     map[string]any{"children": []any{}, "internal": []any{}, "external": []any{}},
			"documents": []any{},
			"tags":      []any{},
			"dates":     map[string]any{},
			"content":   map[string]any{"full_text": r.fullText},
		}
		b, _ := json.MarshalIndent(pageData, "", "  ")
		if err := os.WriteFile(filepath.Join(pageDir, "_index.json"), b, 0o644); err != nil {
			panic(err)
		}
	}

	entries := make([]map[string]any, len(rows))
	for i, r := range rows {
		entries[i] = map[string]any{
			"path": r.path, "url": r.url, "title": r.title, "depth": 2,
			"parent": "", "section": r.section, "page_type": r.pageType,
			"has_substantive_content": true, "mini_summary": r.summary,
			"child_count": 0, "has_docs": false,
		}
	}
	catalog := map[string]any{
		"generated_at": generatedAt.Format(time.RFC3339),
		"root_url":     scopePrefix,
		"schema_version": 2,
		"stats": map[string]any{
			"total_pages": len(rows), "by_depth": map[string]int{"2": len(rows)},
			"by_page_type": map[string]int{"article": 1, "landing": 1},
		},
		"entries": entries,
	}
	catalogData, _ := json.MarshalIndent(catalog, "", "  ")
	// Write catalog files into dataDir so the default path {DATA_DIR}/catalog.json works.
	if err := os.WriteFile(filepath.Join(dataDir, "catalog.json"), catalogData, 0o644); err != nil {
		panic(err)
	}

	dbPath := filepath.Join(dataDir, "catalog.sqlite")
	db, _ := sql.Open("sqlite", dbPath)
	defer db.Close()
	db.Exec(`CREATE VIRTUAL TABLE pages_fts USING fts5(
		path UNINDEXED, url UNINDEXED, title, mini_summary, full_text,
		section UNINDEXED, page_type UNINDEXED,
		tokenize = "unicode61 remove_diacritics 2")`)
	for _, r := range rows {
		db.Exec(`INSERT INTO pages_fts(path,url,title,mini_summary,full_text,section,page_type) VALUES(?,?,?,?,?,?,?)`,
			r.path, r.url, r.title, r.summary, r.fullText, r.section, r.pageType)
	}
}

// BenchmarkColdStart measures the time from exec.Command.Start() until the
// first JSON-RPC response arrives on stdout (the initialize response).
// This covers CA-12: cold start < 500ms.
func BenchmarkColdStart(b *testing.B) {
	initReq := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"clientInfo":      map[string]any{"name": "bench", "version": "0.0.1"},
		},
	}
	initLine, _ := json.Marshal(initReq)
	initLine = append(initLine, '\n')

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var stdin bytes.Buffer
		stdin.Write(initLine)

		cmd := exec.Command(benchBinPath)
		cmd.Stdin = &stdin
		cmd.Env = append(os.Environ(),
			"SITE_RESEARCH_DATA_DIR="+filepath.Join(benchFixtureDir, "data"),
			"SITE_RESEARCH_LOG_LEVEL=error",
		)

		stdoutPipe, err := cmd.StdoutPipe()
		if err != nil {
			b.Fatalf("StdoutPipe: %v", err)
		}

		t0 := time.Now()
		if err := cmd.Start(); err != nil {
			b.Fatalf("cmd.Start: %v", err)
		}

		sc := bufio.NewScanner(stdoutPipe)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		responded := false
		if sc.Scan() {
			elapsed := time.Since(t0)
			b.ReportMetric(float64(elapsed.Milliseconds()), "ms/cold-start")
			responded = true
			if elapsed > 500*time.Millisecond {
				b.Errorf("cold start %v exceeds 500ms budget", elapsed)
			}
		}

		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		if !responded {
			b.Error("no response received")
		}
	}
}

// TestColdStartUnder500ms is a non-benchmark test that asserts the cold start
// budget once (single run, no iteration overhead).
func TestColdStartUnder500ms(t *testing.T) {
	initReq := map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-11-25",
			"clientInfo":      map[string]any{"name": "cold-start-test", "version": "0.0.1"},
		},
	}
	initLine, _ := json.Marshal(initReq)
	initLine = append(initLine, '\n')

	var stdin bytes.Buffer
	stdin.Write(initLine)

	cmd := exec.Command(benchBinPath)
	cmd.Stdin = &stdin
	cmd.Env = append(os.Environ(),
		"SITE_RESEARCH_DATA_DIR="+filepath.Join(benchFixtureDir, "data"),
		"SITE_RESEARCH_LOG_LEVEL=error",
	)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("StdoutPipe: %v", err)
	}

	t0 := time.Now()
	if err := cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	sc := bufio.NewScanner(stdoutPipe)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !sc.Scan() {
		t.Fatal("no response from server")
	}
	elapsed := time.Since(t0)

	t.Logf("cold start: %v", elapsed)

	// Verify the response is valid JSON-RPC.
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(sc.Bytes(), &resp); err != nil {
		t.Fatalf("response is not valid JSON: %v\nline: %s", err, sc.Bytes())
	}
	if string(resp["jsonrpc"]) != `"2.0"` {
		t.Errorf("missing jsonrpc:2.0")
	}

	if elapsed > 500*time.Millisecond {
		t.Errorf("cold start %v exceeds 500ms budget (CA-12)", elapsed)
	}
}
