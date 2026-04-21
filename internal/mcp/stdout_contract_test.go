//go:build integration

package mcp

// stdout_contract_test.go provides CA-10: verifies that every byte written to
// stdout by the MCP server is part of a valid JSON-RPC line, with zero
// spurious bytes, even under a stress workload of 100 interleaved requests
// (50 valid + 50 malformed/invalid).
//
// This file is compiled only with -tags integration because it builds and
// spawns a real subprocess binary, which makes it unsuitable for fast unit
// runs but necessary for the full contract audit.

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/bergmaia/site-research/internal/tools"
)

const contractScopePrefix = "https://www.example.com/transparency"

// buildBinaryContract compiles the MCP binary into a temp directory once.
func buildBinaryContract(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "site-research-mcp")
	cmd := exec.Command("go", "build", "-o", binPath,
		"github.com/bergmaia/site-research/cmd/site-research-mcp")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return binPath
}

// createContractFixtures builds a minimal catalog in tmpDir.
func createContractFixtures(t *testing.T, tmpDir string) {
	t.Helper()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	type row struct {
		path, url, title, section, pageType, summary, fullText string
	}
	rows := []row{
		{"balancetes", contractScopePrefix + "/balancetes", "Balancetes", "Contabilidade", "landing", "Balancetes contábeis.", "Balancetes mensais do TRE."},
		{"diarias", contractScopePrefix + "/diarias", "Diárias", "RH", "article", "Pagamentos de diárias.", "Informações sobre diárias de servidores."},
		{"contratos", contractScopePrefix + "/contratos", "Contratos", "Licitações", "article", "Contratos vigentes.", "Contratos e licitações do TRE."},
	}

	generatedAt := time.Date(2026, 4, 20, 18, 0, 0, 0, time.UTC)
	for _, r := range rows {
		pageDir := filepath.Join(dataDir, r.path)
		if err := os.MkdirAll(pageDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", pageDir, err)
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
			t.Fatalf("write _index.json %s: %v", r.path, err)
		}
	}

	entries := make([]map[string]any, len(rows))
	for i, r := range rows {
		entries[i] = map[string]any{
			"path": r.path, "url": r.url, "title": r.title,
			"depth": 2, "parent": "", "section": r.section,
			"page_type": r.pageType, "has_substantive_content": true,
			"mini_summary": r.summary, "child_count": 0, "has_docs": false,
		}
	}
	catalog := map[string]any{
		"generated_at":   generatedAt.Format(time.RFC3339),
		"root_url":       contractScopePrefix,
		"schema_version": 2,
		"stats": map[string]any{
			"total_pages":  len(rows),
			"by_depth":     map[string]int{"2": len(rows)},
			"by_page_type": map[string]int{"article": 2, "landing": 1},
		},
		"entries": entries,
	}
	catalogData, _ := json.MarshalIndent(catalog, "", "  ")
	// Write catalog files into dataDir so the default path {DATA_DIR}/catalog.json works.
	if err := os.WriteFile(filepath.Join(dataDir, "catalog.json"), catalogData, 0o644); err != nil {
		t.Fatalf("write catalog.json: %v", err)
	}

	dbPath := filepath.Join(dataDir, "catalog.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE VIRTUAL TABLE pages_fts USING fts5(
		path UNINDEXED, url UNINDEXED, title, mini_summary, full_text,
		section UNINDEXED, page_type UNINDEXED,
		tokenize = "unicode61 remove_diacritics 2")`)
	if err != nil {
		t.Fatalf("create pages_fts: %v", err)
	}
	for _, r := range rows {
		_, err = db.Exec(`INSERT INTO pages_fts (path,url,title,mini_summary,full_text,section,page_type) VALUES (?,?,?,?,?,?,?)`,
			r.path, r.url, r.title, r.summary, r.fullText, r.section, r.pageType)
		if err != nil {
			t.Fatalf("insert fts %s: %v", r.path, err)
		}
	}
}

// encodeContractRequest encodes a JSON-RPC request line.
func encodeContractRequest(id int, method string, params any) []byte {
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	b, _ := json.Marshal(req)
	return append(b, '\n')
}

// TestStdoutContractStress is CA-10: 100 interleaved requests (50 valid, 50
// malformed/invalid) — every stdout line must be a parseable JSON-RPC object.
func TestStdoutContractStress(t *testing.T) {
	binPath := buildBinaryContract(t)
	tmpDir := t.TempDir()
	createContractFixtures(t, tmpDir)

	var stdin bytes.Buffer

	// Initialize first.
	stdin.Write(encodeContractRequest(1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"clientInfo":      map[string]any{"name": "stress-test", "version": "0.0.1"},
	}))

	queries := []string{
		"balancetes", "diarias", "contratos", "licitacao", "servidores",
		"pagamento", "receita", "despesa", "transparencia", "orcamento",
	}

	// 50 valid tools/call requests with varying queries.
	for i := 2; i <= 51; i++ {
		query := queries[(i-2)%len(queries)]
		stdin.Write(encodeContractRequest(i, "tools/call", map[string]any{
			"name":      "search",
			"arguments": map[string]any{"query": query},
		}))
	}

	// 50 malformed or invalid requests interleaved.
	for i := 52; i <= 101; i++ {
		switch i % 5 {
		case 0:
			// Completely invalid JSON.
			stdin.Write([]byte("not valid json at all\n"))
		case 1:
			// tools/call with missing query (validation error → isError result).
			stdin.Write(encodeContractRequest(i, "tools/call", map[string]any{
				"name":      "search",
				"arguments": map[string]any{},
			}))
		case 2:
			// Unknown method.
			stdin.Write(encodeContractRequest(i, "unknown/method", nil))
		case 3:
			// tools/call with unknown tool name.
			stdin.Write(encodeContractRequest(i, "tools/call", map[string]any{
				"name":      "nonexistent_tool",
				"arguments": map[string]any{},
			}))
		case 4:
			// tools/call with invalid JSON in arguments (string where object expected).
			line := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"search","arguments":"not-an-object"}}`, i)
			stdin.Write(append([]byte(line), '\n'))
		}
	}

	cmd := exec.Command(binPath)
	cmd.Stdin = &stdin
	cmd.Env = append(os.Environ(),
		"SITE_RESEARCH_DATA_DIR="+filepath.Join(tmpDir, "data"),
		"SITE_RESEARCH_LOG_LEVEL=error",
	)

	stdout, _ := cmd.Output() // ignore process exit; audit stdout content

	// Every stdout line must be a parseable JSON-RPC object with jsonrpc:2.0.
	lineNum := 0
	badLines := 0
	sc := bufio.NewScanner(bytes.NewReader(stdout))
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		lineNum++
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("stdout line %d is not valid JSON: %v\nline: %.200s", lineNum, err, line)
			badLines++
			continue
		}
		if v, ok := obj["jsonrpc"]; !ok || string(v) != `"2.0"` {
			t.Errorf("stdout line %d missing jsonrpc:\"2.0\"\nline: %.200s", lineNum, line)
			badLines++
		}
	}
	if sc.Err() != nil {
		t.Fatalf("scanner error: %v", sc.Err())
	}
	if badLines > 0 {
		t.Fatalf("%d/%d stdout lines failed the JSON-RPC contract", badLines, lineNum)
	}
	if lineNum == 0 {
		t.Fatal("no stdout lines produced — server may have crashed")
	}
	t.Logf("stdout contract: %d lines all valid JSON-RPC", lineNum)
}

// TestStdoutContractConcurrent runs many parallel tool calls via an in-process
// server and confirms zero spurious bytes in the output.
func TestStdoutContractConcurrent(t *testing.T) {
	const workers = 20
	const callsPerWorker = 5

	reg := tools.NewRegistry()
	schema, _ := json.Marshal(map[string]any{"type": "object", "properties": map[string]any{}})
	reg.RegisterTool(tools.Tool{Name: "noop", Description: "noop", InputSchema: schema},
		func(_ context.Context, _ json.RawMessage) (tools.CallToolResult, error) {
			return tools.CallToolResult{
				Content: []tools.ContentBlock{{Type: "text", Text: "ok"}},
			}, nil
		},
	)

	var sb safeBuffer
	pr, pw := io.Pipe()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(logger, reg, pr, &sb, "test")

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		_ = srv.Run(context.Background())
	}()

	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		w := w
		go func() {
			defer wg.Done()
			for c := 0; c < callsPerWorker; c++ {
				id := w*callsPerWorker + c + 2
				req := map[string]any{
					"jsonrpc": "2.0",
					"id":      id,
					"method":  "tools/call",
					"params":  map[string]any{"name": "noop", "arguments": map[string]any{}},
				}
				b, _ := json.Marshal(req)
				b = append(b, '\n')
				pw.Write(b)
			}
		}()
	}
	wg.Wait()
	pw.Close()
	<-serverDone

	data := sb.Bytes()
	lineNum := 0
	bad := 0
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		lineNum++
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d not valid JSON: %v\n%.200s", lineNum, err, line)
			bad++
		}
	}
	if bad > 0 {
		t.Errorf("%d/%d stdout lines invalid", bad, lineNum)
	}
	expected := workers * callsPerWorker
	if lineNum != expected {
		t.Errorf("expected %d responses, got %d", expected, lineNum)
	}
}
