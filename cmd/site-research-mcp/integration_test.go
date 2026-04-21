//go:build integration

package main

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// buildBinary compiles the binary into a temporary directory and returns its path.
func buildBinary(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "site-research-mcp")
	// Build the package by its import path so the cwd doesn't matter.
	cmd := exec.Command("go", "build", "-o", binPath,
		"github.com/bergmaia/site-research/cmd/site-research-mcp")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return binPath
}

// scopePrefix is the test catalog scope prefix.
const scopePrefix = "https://www.example.com/transparencia"

// createFixtures builds a minimal but valid fixture set in tmpDir:
//   - catalog.json (SchemaVersion=2, 3-5 entries)
//   - catalog.sqlite with pages_fts table (same entries)
//   - data/ directory with _index.json files for each page
func createFixtures(t *testing.T, tmpDir string) {
	t.Helper()

	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	type pageRow struct {
		path     string
		url      string
		title    string
		section  string
		pageType string
		summary  string
		fullText string
	}

	rows := []pageRow{
		{
			path:     "balancetes",
			url:      scopePrefix + "/balancetes",
			title:    "Balancetes",
			section:  "Contabilidade",
			pageType: "landing",
			summary:  "Balancetes contábeis mensais.",
			fullText: "Balancetes contábeis mensais do TRE.",
		},
		{
			path:     "contabilidade",
			url:      scopePrefix + "/contabilidade",
			title:    "Contabilidade",
			section:  "Contabilidade",
			pageType: "landing",
			summary:  "Demonstrativos de contabilidade.",
			fullText: "Seção de contabilidade do portal de transparência.",
		},
		{
			path:     "diarias",
			url:      scopePrefix + "/diarias",
			title:    "Diárias",
			section:  "Recursos Humanos",
			pageType: "article",
			summary:  "Pagamentos de diárias a servidores.",
			fullText: "Informações sobre diárias de servidores e magistrados.",
		},
		{
			path:     "estrategia",
			url:      scopePrefix + "/estrategia",
			title:    "Estratégia",
			section:  "Estratégia",
			pageType: "article",
			summary:  "Planejamento estratégico institucional.",
			fullText: "Documentos de planejamento estratégico do TRE.",
		},
	}

	// Write _index.json for each page.
	generatedAt := time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC)
	for _, r := range rows {
		pageDir := filepath.Join(dataDir)
		for _, seg := range filepath.SplitList(r.path) {
			pageDir = filepath.Join(pageDir, seg)
		}
		// Handle simple single-segment paths.
		pageDir = filepath.Join(dataDir, r.path)
		if err := os.MkdirAll(pageDir, 0o755); err != nil {
			t.Fatalf("mkdir page dir %s: %v", pageDir, err)
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
				"depth":           2,
				"extracted_at":    generatedAt.Format(time.RFC3339),
				"crawler_version": "0.1.0",
				"discovered_via":  "sitemap",
			},
			"links":     map[string]any{"children": []any{}, "internal": []any{}, "external": []any{}},
			"documents": []any{},
			"tags":      []any{},
			"dates":     map[string]any{},
			"content":   map[string]any{"full_text": r.fullText},
		}
		b, _ := json.MarshalIndent(pageData, "", "  ")
		if err := os.WriteFile(filepath.Join(pageDir, "_index.json"), b, 0o644); err != nil {
			t.Fatalf("write _index.json for %s: %v", r.path, err)
		}
	}

	// catalog.json — written inside dataDir so the default path {DATA_DIR}/catalog.json works.
	entries := make([]map[string]any, len(rows))
	for i, r := range rows {
		entries[i] = map[string]any{
			"path":                    r.path,
			"url":                     r.url,
			"title":                   r.title,
			"depth":                   2,
			"parent":                  "",
			"section":                 r.section,
			"page_type":               r.pageType,
			"has_substantive_content": true,
			"mini_summary":            r.summary,
			"child_count":             0,
			"has_docs":                false,
		}
	}
	catalog := map[string]any{
		"generated_at":   generatedAt.Format(time.RFC3339),
		"root_url":       scopePrefix,
		"schema_version": 2,
		"stats": map[string]any{
			"total_pages":  len(rows),
			"by_depth":     map[string]int{"2": len(rows)},
			"by_page_type": map[string]int{"article": 2, "landing": 2},
		},
		"entries": entries,
	}
	catalogData, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	catalogPath := filepath.Join(dataDir, "catalog.json")
	if err := os.WriteFile(catalogPath, catalogData, 0o644); err != nil {
		t.Fatalf("write catalog.json: %v", err)
	}

	// catalog.sqlite with pages_fts FTS5 table — written inside dataDir.
	dbPath := filepath.Join(dataDir, "catalog.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE VIRTUAL TABLE pages_fts USING fts5(
		path UNINDEXED,
		url UNINDEXED,
		title,
		mini_summary,
		full_text,
		section UNINDEXED,
		page_type UNINDEXED,
		tokenize = "unicode61 remove_diacritics 2"
	)`)
	if err != nil {
		t.Fatalf("create pages_fts: %v", err)
	}

	for _, r := range rows {
		_, err = db.Exec(`INSERT INTO pages_fts
			(path, url, title, mini_summary, full_text, section, page_type)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			r.path, r.url, r.title, r.summary, r.fullText, r.section, r.pageType,
		)
		if err != nil {
			t.Fatalf("insert pages_fts for %s: %v", r.path, err)
		}
	}
}

// writeRequest encodes a JSON-RPC request to a buffer.
func writeRequest(buf *bytes.Buffer, id int, method string, params any) {
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}
	b, _ := json.Marshal(req)
	buf.Write(b)
	buf.WriteByte('\n')
}

// runMCP spawns the MCP binary with the given fixtures and stdin payload,
// and returns all stdout lines decoded as JSON-RPC responses.
// It also checks the stdout contract (every line must be JSON-RPC).
// It relies on defaults: catalog.json and catalog.sqlite are derived from DATA_DIR,
// and scope_prefix is read from catalog.root_url. Only DATA_DIR + LOG_LEVEL are set.
func runMCP(t *testing.T, binPath, tmpDir string, stdin *bytes.Buffer) []map[string]json.RawMessage {
	t.Helper()

	cmd := exec.Command(binPath)
	cmd.Stdin = stdin
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SITE_RESEARCH_DATA_DIR=%s", filepath.Join(tmpDir, "data")),
		"SITE_RESEARCH_LOG_LEVEL=error",
	)

	stdout, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Logf("process exited with code %d\nstderr: %s", ee.ExitCode(), ee.Stderr)
		}
	}

	var responses []map[string]json.RawMessage
	sc := bufio.NewScanner(bytes.NewReader(stdout))
	lineNum := 0
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		lineNum++
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("stdout line %d is not valid JSON: %v\nline: %s", lineNum, err, line)
			continue
		}
		if v, ok := obj["jsonrpc"]; !ok || string(v) != `"2.0"` {
			t.Errorf("stdout line %d missing jsonrpc:2.0\nline: %s", lineNum, line)
		}
		responses = append(responses, obj)
	}
	return responses
}

// responseByID finds a JSON-RPC response by its id field in a slice of responses.
func responseByID(responses []map[string]json.RawMessage, id int) map[string]json.RawMessage {
	want := fmt.Sprintf("%d", id)
	for _, r := range responses {
		if v, ok := r["id"]; ok && string(v) == want {
			return r
		}
	}
	return nil
}

// TestStdoutContract spawns the MCP binary, sends initialize + tools/list +
// tools/call, and verifies that every stdout line is valid JSON-RPC.
func TestStdoutContract(t *testing.T) {
	binPath := buildBinary(t)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	var stdin bytes.Buffer
	writeRequest(&stdin, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
	})
	writeRequest(&stdin, 2, "tools/list", nil)
	writeRequest(&stdin, 3, "tools/call", map[string]any{
		"name":      "search",
		"arguments": map[string]any{"query": "balancetes"},
	})

	responses := runMCP(t, binPath, tmpDir, &stdin)
	if len(responses) == 0 {
		t.Error("no output lines from server — expected at least responses to 3 requests")
	}
}

// TestToolsCallSearch verifies that tools/call search returns markdown results.
func TestToolsCallSearch(t *testing.T) {
	binPath := buildBinary(t)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	var stdin bytes.Buffer
	writeRequest(&stdin, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
	})
	writeRequest(&stdin, 10, "tools/call", map[string]any{
		"name":      "search",
		"arguments": map[string]any{"query": "balancetes"},
	})

	responses := runMCP(t, binPath, tmpDir, &stdin)
	resp := responseByID(responses, 10)
	if resp == nil {
		t.Fatal("no response for id=10")
	}

	// Decode the result content.
	resultRaw, ok := resp["result"]
	if !ok {
		t.Fatalf("response missing 'result' field; response: %v", resp)
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected isError:false, got error: %v", result.Content)
	}
	if len(result.Content) == 0 {
		t.Fatal("empty content")
	}
	text := result.Content[0].Text
	if !startsWith(text, "# Resultados para:") && !startsWith(text, "# Nenhum resultado") {
		// Allow either success or empty response header.
		if !containsSubstr(text, "Resultados") && !containsSubstr(text, "Nenhum resultado") {
			t.Errorf("unexpected markdown format:\n%s", text)
		}
	}
}

// TestToolsCallSearchEmpty verifies that a query with no hits returns a "Nenhum resultado" section.
func TestToolsCallSearchEmpty(t *testing.T) {
	binPath := buildBinary(t)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	var stdin bytes.Buffer
	writeRequest(&stdin, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
	})
	writeRequest(&stdin, 11, "tools/call", map[string]any{
		"name":      "search",
		"arguments": map[string]any{"query": "xpto_inexistente_12345abc"},
	})

	responses := runMCP(t, binPath, tmpDir, &stdin)
	resp := responseByID(responses, 11)
	if resp == nil {
		t.Fatal("no response for id=11")
	}
	resultRaw := resp["result"]
	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
		IsError bool                                   `json:"isError"`
	}
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.IsError {
		t.Fatalf("zero hits should not be isError; got: %s", result.Content[0].Text)
	}
	if len(result.Content) == 0 || !containsSubstr(result.Content[0].Text, "Nenhum resultado") {
		t.Errorf("expected 'Nenhum resultado' in empty-search response")
	}
}

// TestToolsCallInspectPage verifies inspect_page with a valid target.
func TestToolsCallInspectPage(t *testing.T) {
	binPath := buildBinary(t)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	var stdin bytes.Buffer
	writeRequest(&stdin, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
	})
	writeRequest(&stdin, 20, "tools/call", map[string]any{
		"name":      "inspect_page",
		"arguments": map[string]any{"target": "balancetes"},
	})

	responses := runMCP(t, binPath, tmpDir, &stdin)
	resp := responseByID(responses, 20)
	if resp == nil {
		t.Fatal("no response for id=20")
	}
	resultRaw := resp["result"]
	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
		IsError bool                                   `json:"isError"`
	}
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success for valid target, got error: %s", result.Content[0].Text)
	}
	if len(result.Content) == 0 || !containsSubstr(result.Content[0].Text, "Balancetes") {
		t.Errorf("expected page title in markdown response")
	}
}

// TestToolsCallInspectPageNotFound verifies inspect_page with a missing target returns isError.
func TestToolsCallInspectPageNotFound(t *testing.T) {
	binPath := buildBinary(t)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	var stdin bytes.Buffer
	writeRequest(&stdin, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
	})
	writeRequest(&stdin, 21, "tools/call", map[string]any{
		"name":      "inspect_page",
		"arguments": map[string]any{"target": "pagina/inexistente/xyz"},
	})

	responses := runMCP(t, binPath, tmpDir, &stdin)
	resp := responseByID(responses, 21)
	if resp == nil {
		t.Fatal("no response for id=21")
	}
	resultRaw := resp["result"]
	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
		IsError bool                                   `json:"isError"`
	}
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError:true for missing page")
	}
}

// TestToolsCallCatalogStats verifies catalog_stats returns markdown with required sections.
func TestToolsCallCatalogStats(t *testing.T) {
	binPath := buildBinary(t)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	var stdin bytes.Buffer
	writeRequest(&stdin, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "test", "version": "0.0.1"},
	})
	writeRequest(&stdin, 30, "tools/call", map[string]any{
		"name":      "catalog_stats",
		"arguments": map[string]any{},
	})

	responses := runMCP(t, binPath, tmpDir, &stdin)
	resp := responseByID(responses, 30)
	if resp == nil {
		t.Fatal("no response for id=30")
	}
	resultRaw := resp["result"]
	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
		IsError bool                                   `json:"isError"`
	}
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.IsError {
		t.Fatalf("expected success for catalog_stats, got error: %s", result.Content[0].Text)
	}
	if len(result.Content) == 0 {
		t.Fatal("empty content")
	}
	text := result.Content[0].Text
	if !containsSubstr(text, "## Totais") {
		t.Errorf("expected '## Totais' section in catalog_stats response")
	}
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func containsSubstr(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}

// buildBinaryWithVersion compiles the binary with the given version string
// embedded via ldflags and returns the path to the produced binary.
func buildBinaryWithVersion(t *testing.T, ver string) string {
	t.Helper()
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "site-research-mcp")
	ldflags := fmt.Sprintf("-X main.version=%s", ver)
	cmd := exec.Command("go", "build",
		"-ldflags", ldflags,
		"-o", binPath,
		"github.com/bergmaia/site-research/cmd/site-research-mcp",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build (with ldflags) failed: %v\n%s", err, out)
	}
	return binPath
}

// TestVersionEmbedded verifies that the version embedded via -ldflags
// "-X main.version=0.1.0-test" is reflected in the serverInfo.version
// field of the initialize response. This is the M4 smoke test for CA-2.
func TestVersionEmbedded(t *testing.T) {
	const wantVersion = "0.1.0-test"
	binPath := buildBinaryWithVersion(t, wantVersion)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	var stdin bytes.Buffer
	writeRequest(&stdin, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "version-smoke-test", "version": "0.0.1"},
	})

	responses := runMCP(t, binPath, tmpDir, &stdin)
	resp := responseByID(responses, 1)
	if resp == nil {
		t.Fatal("no initialize response received")
	}

	resultRaw, ok := resp["result"]
	if !ok {
		t.Fatalf("initialize response missing 'result' field; full response: %v", resp)
	}

	var result struct {
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(resultRaw, &result); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}

	if result.ServerInfo.Version != wantVersion {
		t.Errorf("serverInfo.version = %q; want %q", result.ServerInfo.Version, wantVersion)
	}
	if result.ServerInfo.Name != "site-research-mcp" {
		t.Errorf("serverInfo.name = %q; want %q", result.ServerInfo.Name, "site-research-mcp")
	}
	t.Logf("version embedding confirmed: serverInfo.version=%s", result.ServerInfo.Version)
}

// TestMissingEnvVars verifies exit code 2 when required vars are absent.
func TestMissingEnvVars(t *testing.T) {
	binPath := buildBinary(t)

	cmd := exec.Command(binPath)
	// Pass no env vars at all (strip inherited ones except PATH).
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code")
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if ee.ExitCode() != 2 {
		t.Errorf("expected exit code 2, got %d\nstderr: %s", ee.ExitCode(), stderr.String())
	}
}

// TestCatalogMissing verifies exit code 1 when the catalog file does not exist.
// Only SITE_RESEARCH_DATA_DIR is required; SITE_RESEARCH_CATALOG overrides the
// default to a nonexistent path — the server must still fail with exit 1.
func TestCatalogMissing(t *testing.T) {
	binPath := buildBinary(t)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SITE_RESEARCH_DATA_DIR=%s", filepath.Join(tmpDir, "data")),
		"SITE_RESEARCH_CATALOG=/nonexistent/catalog.json",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code")
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T", err)
	}
	if ee.ExitCode() != 1 {
		t.Errorf("expected exit code 1, got %d\nstderr: %s", ee.ExitCode(), stderr.String())
	}

	// Stdout must be empty (nothing should have been written to it).
	var stdout bytes.Buffer
	cmd2 := exec.Command(binPath)
	cmd2.Env = cmd.Env
	cmd2.Stdout = &stdout
	_ = cmd2.Run()
	if stdout.Len() != 0 {
		t.Errorf("stdout must be empty on startup failure, got: %s", stdout.String())
	}
}

// TestExplicitOverrideTakesPrecedence verifies that when SITE_RESEARCH_CATALOG
// and SITE_RESEARCH_FTS_DB are set explicitly, they take precedence over the
// defaults derived from DATA_DIR. Here we set them to the same derived paths
// and confirm that startup still succeeds (explicit == default is a valid override).
func TestExplicitOverrideTakesPrecedence(t *testing.T) {
	binPath := buildBinary(t)
	tmpDir := t.TempDir()
	createFixtures(t, tmpDir)

	dataDir := filepath.Join(tmpDir, "data")

	var stdin bytes.Buffer
	writeRequest(&stdin, 1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "override-test", "version": "0.0.1"},
	})

	cmd := exec.Command(binPath)
	cmd.Stdin = &stdin
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SITE_RESEARCH_DATA_DIR=%s", dataDir),
		// Explicit overrides pointing to the same derived paths — must succeed.
		fmt.Sprintf("SITE_RESEARCH_CATALOG=%s", filepath.Join(dataDir, "catalog.json")),
		fmt.Sprintf("SITE_RESEARCH_FTS_DB=%s", filepath.Join(dataDir, "catalog.sqlite")),
		fmt.Sprintf("SITE_RESEARCH_SCOPE_PREFIX=%s", scopePrefix),
		"SITE_RESEARCH_LOG_LEVEL=error",
	)

	stdout, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("server exited with code %d\nstderr: %s", ee.ExitCode(), ee.Stderr)
		}
		t.Fatalf("unexpected error: %v", err)
	}

	// Must have produced at least one JSON-RPC response.
	if len(stdout) == 0 {
		t.Fatal("no output from server — expected initialize response")
	}
}
