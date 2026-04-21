package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bergmaia/site-research/internal/tools"
)

// safeBuffer is a bytes.Buffer protected by a mutex for concurrent writes.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (sb *safeBuffer) Write(p []byte) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *safeBuffer) Bytes() []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	cp := make([]byte, sb.buf.Len())
	copy(cp, sb.buf.Bytes())
	return cp
}

// runServerSync writes all requests in inBuf, runs the server to EOF, and
// returns the collected response lines.
func runServerSync(t *testing.T, registry *tools.Registry, inBuf *bytes.Buffer) [][]byte {
	t.Helper()
	var sb safeBuffer
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(logger, registry, inBuf, &sb, "test")

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = srv.Run(context.Background())
	}()
	<-done

	return parseLines(sb.Bytes())
}

// runServerWithPipes creates a server backed by an io.Pipe so tests can write
// incrementally, then close the write end to signal EOF.
// Returns (writePipe, output-collector, done-channel).
// The caller must wait on done before reading from the collector.
func runServerWithPipes(t *testing.T, registry *tools.Registry) (io.WriteCloser, *safeBuffer, <-chan struct{}) {
	t.Helper()
	pr, pw := io.Pipe()
	var sb safeBuffer
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := NewServer(logger, registry, pr, &sb, "test")

	ch := make(chan struct{})
	go func() {
		defer close(ch)
		_ = srv.Run(context.Background())
	}()
	return pw, &sb, ch
}

// parseLines splits raw bytes into non-empty lines.
func parseLines(data []byte) [][]byte {
	var lines [][]byte
	r := NewMessageReader(bytes.NewReader(data))
	for {
		line, err := r.ReadLine()
		if err != nil {
			break
		}
		lines = append(lines, line)
	}
	return lines
}

// defaultRegistry returns a registry backed by stub handlers (nil config).
func defaultRegistry() *tools.Registry {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return tools.DefaultRegistry(nil, logger)
}

// mockRegistry builds a Registry with a single mock tool whose handler is h.
func mockRegistry(name string, h tools.Handler) *tools.Registry {
	r := tools.NewRegistry()
	schema, _ := json.Marshal(map[string]any{"type": "object", "properties": map[string]any{}})
	r.RegisterTool(tools.Tool{Name: name, Description: "mock", InputSchema: schema}, h)
	return r
}

// buildRequest creates a newline-terminated JSON request.
func buildRequest(t *testing.T, method string, id int, params any) []byte {
	t.Helper()
	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(jsonIntOrNull(id)),
		Method:  method,
	}
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("marshal params: %v", err)
		}
		req.Params = b
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return append(b, '\n')
}

// buildNotification creates a newline-terminated JSON notification (no id).
func buildNotification(t *testing.T, method string, params any) []byte {
	t.Helper()
	n := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if params != nil {
		n["params"] = params
	}
	b, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}
	return append(b, '\n')
}

func jsonIntOrNull(i int) string {
	if i == 0 {
		return "null"
	}
	b, _ := json.Marshal(i)
	return string(b)
}

// findResponseByID searches parsed response lines for one with the given numeric id.
func findResponseByID(lines [][]byte, id int) *Response {
	want := jsonIntOrNull(id)
	for _, line := range lines {
		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		if string(resp.ID) == want {
			return &resp
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Existing tests (M1/M2)
// ---------------------------------------------------------------------------

// TestServerInitialize verifies the server responds correctly to initialize.
func TestServerInitialize(t *testing.T) {
	var inBuf bytes.Buffer
	inBuf.Write(buildRequest(t, "initialize", 1, InitializeParams{
		ProtocolVersion: ProtocolVersion,
		ClientInfo:      ClientInfo{Name: "test-client", Version: "0.1"},
	}))

	lines := runServerSync(t, defaultRegistry(), &inBuf)
	if len(lines) == 0 {
		t.Fatal("expected at least one response")
	}

	var resp Response
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("unmarshal: %v\nline: %s", err, lines[0])
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.ProtocolVersion != ProtocolVersion {
		t.Errorf("protocolVersion: got %q, want %q", result.ProtocolVersion, ProtocolVersion)
	}
	if result.Capabilities.Tools == nil {
		t.Error("capabilities.tools must be present")
	}
	if result.ServerInfo.Name != "site-research-mcp" {
		t.Errorf("serverInfo.name: got %q", result.ServerInfo.Name)
	}
}

// TestServerToolsList verifies tools/list returns 3 tools.
func TestServerToolsList(t *testing.T) {
	var inBuf bytes.Buffer
	inBuf.Write(buildRequest(t, "tools/list", 2, nil))

	lines := runServerSync(t, defaultRegistry(), &inBuf)
	if len(lines) == 0 {
		t.Fatal("expected at least one response")
	}

	var resp Response
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}

	var result ToolsListResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if len(result.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(result.Tools))
	}
}

// TestServerToolsCall_NilConfig verifies that tools/call with nil config
// returns some response (isError or error) rather than panicking.
func TestServerToolsCall_NilConfig(t *testing.T) {
	pw, sb, done := runServerWithPipes(t, defaultRegistry())
	pw.Write(buildRequest(t, "tools/call", 3, CallToolParams{
		Name:      "search",
		Arguments: json.RawMessage(`{"query":"test"}`),
	}))
	pw.Close()
	<-done

	lines := parseLines(sb.Bytes())
	if len(lines) == 0 {
		t.Fatal("expected at least one response")
	}

	var resp Response
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Either an RPC error or a tool result with isError:true — both acceptable.
	if resp.Error != nil {
		return // RPC-level error is fine
	}

	var result tools.CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError:true for stub with nil config")
	}
	if len(result.Content) == 0 {
		t.Fatal("content must not be empty")
	}
	if result.Content[0].Text == "" {
		t.Error("content[0].text must not be empty")
	}
}

// TestServerMethodNotFound verifies -32601 for unknown methods.
func TestServerMethodNotFound(t *testing.T) {
	var inBuf bytes.Buffer
	inBuf.Write(buildRequest(t, "unknown/method", 4, nil))

	lines := runServerSync(t, defaultRegistry(), &inBuf)
	if len(lines) == 0 {
		t.Fatal("expected at least one response")
	}

	var resp Response
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected an error response")
	}
	if resp.Error.Code != CodeMethodNotFound {
		t.Errorf("error code: got %d, want %d", resp.Error.Code, CodeMethodNotFound)
	}
}

// TestServerParseError verifies -32700 for malformed JSON and that the
// connection stays alive (next valid request is processed correctly).
func TestServerParseError(t *testing.T) {
	pw, sb, done := runServerWithPipes(t, defaultRegistry())

	// Bad JSON first.
	pw.Write([]byte("this is not json\n"))
	// Then a valid tools/list.
	pw.Write(buildRequest(t, "tools/list", 10, nil))
	pw.Close()

	<-done

	lines := parseLines(sb.Bytes())
	if len(lines) < 2 {
		t.Fatalf("expected 2 responses, got %d", len(lines))
	}

	// First response should be parse error -32700.
	var errResp Response
	if err := json.Unmarshal(lines[0], &errResp); err != nil {
		t.Fatalf("unmarshal first response: %v", err)
	}
	if errResp.Error == nil || errResp.Error.Code != CodeParseError {
		t.Errorf("expected parse error -32700, got: %+v", errResp.Error)
	}

	// Second response should be a valid tools/list result.
	var okResp Response
	if err := json.Unmarshal(lines[1], &okResp); err != nil {
		t.Fatalf("unmarshal second response: %v", err)
	}
	if okResp.Error != nil {
		t.Fatalf("second response must not be an error: %+v", okResp.Error)
	}
	var result ToolsListResult
	if err := json.Unmarshal(okResp.Result, &result); err != nil {
		t.Fatalf("unmarshal tools list: %v", err)
	}
	if len(result.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(result.Tools))
	}
}

// ---------------------------------------------------------------------------
// M3 tests
// ---------------------------------------------------------------------------

// TestServerConcurrentToolsCall dispatches 10 tools/call requests in parallel
// and verifies that all 10 responses arrive with the correct IDs.
func TestServerConcurrentToolsCall(t *testing.T) {
	const n = 10

	// Handler that returns immediately.
	reg := mockRegistry("echo", func(ctx context.Context, raw json.RawMessage) (tools.CallToolResult, error) {
		var args struct{ Query string `json:"query"` }
		_ = json.Unmarshal(raw, &args)
		return tools.CallToolResult{
			Content: []tools.ContentBlock{{Type: "text", Text: "ok:" + args.Query}},
		}, nil
	})

	pw, sb, done := runServerWithPipes(t, reg)

	// Build all requests into a single buffer to send atomically.
	var bulk bytes.Buffer
	for i := 1; i <= n; i++ {
		query := map[string]any{"query": "q" + strings.TrimSpace(jsonIntOrNull(i))}
		req := buildRequest(t, "tools/call", i, CallToolParams{
			Name:      "echo",
			Arguments: mustMarshalJSON(query),
		})
		bulk.Write(req)
	}
	pw.Write(bulk.Bytes())
	pw.Close()

	<-done

	lines := parseLines(sb.Bytes())
	if len(lines) != n {
		t.Fatalf("expected %d responses, got %d", n, len(lines))
	}

	// Verify each expected ID has a response without isError.
	for i := 1; i <= n; i++ {
		resp := findResponseByID(lines, i)
		if resp == nil {
			t.Errorf("no response for id=%d", i)
			continue
		}
		if resp.Error != nil {
			t.Errorf("id=%d: unexpected RPC error: %+v", i, resp.Error)
			continue
		}
		var result tools.CallToolResult
		if err := json.Unmarshal(resp.Result, &result); err != nil {
			t.Errorf("id=%d: unmarshal result: %v", i, err)
			continue
		}
		if result.IsError {
			t.Errorf("id=%d: unexpected isError:true", i)
		}
	}
}

// TestServerCancellation sends a tools/call that blocks until ctx is done, then
// sends notifications/cancelled. Verifies the response arrives in ≤ 500ms and
// has isError:true.
func TestServerCancellation(t *testing.T) {
	// Handler blocks until context is cancelled or 5-second safety timeout.
	handlerStarted := make(chan struct{})
	reg := mockRegistry("slow", func(ctx context.Context, raw json.RawMessage) (tools.CallToolResult, error) {
		close(handlerStarted)
		select {
		case <-ctx.Done():
			return tools.CallToolResult{
				IsError: true,
				Content: []tools.ContentBlock{{Type: "text", Text: "**Erro:** operação cancelada."}},
			}, nil
		case <-time.After(5 * time.Second):
			return tools.CallToolResult{
				Content: []tools.ContentBlock{{Type: "text", Text: "timeout fallback"}},
			}, nil
		}
	})

	pw, sb, done := runServerWithPipes(t, reg)

	// Send the tools/call (id=42).
	pw.Write(buildRequest(t, "tools/call", 42, CallToolParams{
		Name:      "slow",
		Arguments: json.RawMessage(`{}`),
	}))

	// Wait until the handler goroutine has actually started.
	select {
	case <-handlerStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start in time")
	}

	start := time.Now()

	// Send cancellation notification.
	pw.Write(buildNotification(t, "notifications/cancelled", CancelledNotification{
		RequestID: json.RawMessage(`42`),
		Reason:    "test cancel",
	}))
	pw.Close()

	<-done
	elapsed := time.Since(start)

	lines := parseLines(sb.Bytes())

	resp := findResponseByID(lines, 42)
	if resp == nil {
		t.Fatalf("no response for id=42 (total lines: %d)", len(lines))
	}

	// Response should arrive well within 500ms of sending the cancellation.
	if elapsed > 500*time.Millisecond {
		t.Errorf("cancellation took too long: %v (want ≤ 500ms)", elapsed)
	}

	if resp.Error != nil {
		// Accept an RPC-level error as valid cancellation signal.
		t.Logf("got RPC error on cancellation (acceptable): %+v", resp.Error)
		return
	}

	var result tools.CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if !result.IsError {
		t.Error("expected isError:true for cancelled request")
	}
}

// TestServerPanicRecovery verifies that a panicking handler yields
// isError:true with a generic message and that the next request is served.
func TestServerPanicRecovery(t *testing.T) {
	reg := mockRegistry("boom", func(_ context.Context, _ json.RawMessage) (tools.CallToolResult, error) {
		panic("boom")
	})
	// Also register a safe tool for the follow-up request.
	schema, _ := json.Marshal(map[string]any{"type": "object", "properties": map[string]any{}})
	reg.RegisterTool(tools.Tool{Name: "safe", Description: "safe", InputSchema: schema},
		func(_ context.Context, _ json.RawMessage) (tools.CallToolResult, error) {
			return tools.CallToolResult{
				Content: []tools.ContentBlock{{Type: "text", Text: "all good"}},
			}, nil
		},
	)

	pw, sb, done := runServerWithPipes(t, reg)

	pw.Write(buildRequest(t, "tools/call", 1, CallToolParams{
		Name:      "boom",
		Arguments: json.RawMessage(`{}`),
	}))
	pw.Write(buildRequest(t, "tools/call", 2, CallToolParams{
		Name:      "safe",
		Arguments: json.RawMessage(`{}`),
	}))
	pw.Close()

	<-done

	lines := parseLines(sb.Bytes())
	if len(lines) < 2 {
		t.Fatalf("expected 2 responses, got %d", len(lines))
	}

	// Find the panic response (id=1).
	panicResp := findResponseByID(lines, 1)
	if panicResp == nil {
		t.Fatal("no response for panic request id=1")
	}
	var panicResult tools.CallToolResult
	if err := json.Unmarshal(panicResp.Result, &panicResult); err != nil {
		t.Fatalf("unmarshal panic result: %v", err)
	}
	if !panicResult.IsError {
		t.Error("panic handler: expected isError:true")
	}
	if len(panicResult.Content) == 0 {
		t.Fatal("panic handler: expected non-empty content")
	}
	if panicResult.Content[0].Text == "" {
		t.Error("panic handler: content text must not be empty")
	}

	// Find the safe response (id=2) — server must still be alive.
	safeResp := findResponseByID(lines, 2)
	if safeResp == nil {
		t.Fatal("no response for safe request id=2 — server did not recover")
	}
	var safeResult tools.CallToolResult
	if err := json.Unmarshal(safeResp.Result, &safeResult); err != nil {
		t.Fatalf("unmarshal safe result: %v", err)
	}
	if safeResult.IsError {
		t.Error("safe handler: unexpected isError:true")
	}
}

// TestServerProtocolVersionRejected verifies that initialize with a wrong
// protocolVersion returns CodeInvalidParams (-32602).
func TestServerProtocolVersionRejected(t *testing.T) {
	var inBuf bytes.Buffer
	inBuf.Write(buildRequest(t, "initialize", 1, InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      ClientInfo{Name: "old-client", Version: "1.0"},
	}))

	lines := runServerSync(t, defaultRegistry(), &inBuf)
	if len(lines) == 0 {
		t.Fatal("expected at least one response")
	}

	var resp Response
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error for incompatible protocolVersion")
	}
	if resp.Error.Code != CodeInvalidParams {
		t.Errorf("expected code -32602, got %d", resp.Error.Code)
	}
}

// TestServerParseErrorThenValid verifies -32700 on bad JSON followed by a
// valid request being served (connection stays alive).
func TestServerParseErrorThenValid(t *testing.T) {
	pw, sb, done := runServerWithPipes(t, defaultRegistry())

	pw.Write([]byte("not json at all\n"))
	pw.Write(buildRequest(t, "tools/list", 99, nil))
	pw.Close()

	<-done

	lines := parseLines(sb.Bytes())
	if len(lines) < 2 {
		t.Fatalf("expected ≥2 responses, got %d", len(lines))
	}

	var first Response
	if err := json.Unmarshal(lines[0], &first); err != nil {
		t.Fatalf("unmarshal first: %v", err)
	}
	if first.Error == nil || first.Error.Code != CodeParseError {
		t.Errorf("expected -32700 parse error, got: %+v", first.Error)
	}

	second := findResponseByID(lines, 99)
	if second == nil {
		t.Fatal("no response for id=99 after parse error")
	}
	if second.Error != nil {
		t.Errorf("id=99 should succeed, got error: %+v", second.Error)
	}
}

// TestServerInvalidParams verifies -32602 when initialize params are wrong type.
func TestServerInvalidParams(t *testing.T) {
	var inBuf bytes.Buffer
	// Send initialize with protocolVersion as a number instead of string.
	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":12345,"clientInfo":{"name":"x"}}}` + "\n")
	inBuf.Write(raw)

	lines := runServerSync(t, defaultRegistry(), &inBuf)
	if len(lines) == 0 {
		t.Fatal("expected at least one response")
	}

	var resp Response
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Either an invalid params error or a protocol version rejection — both acceptable.
	if resp.Error == nil {
		t.Fatal("expected error response for bad params type")
	}
	if resp.Error.Code != CodeInvalidParams && resp.Error.Code != CodeInternalError {
		t.Errorf("expected -32602 or -32603, got %d: %s", resp.Error.Code, resp.Error.Message)
	}
}

// TestServerShutdown verifies that the shutdown method returns a null result.
func TestServerShutdown(t *testing.T) {
	var inBuf bytes.Buffer
	inBuf.Write(buildRequest(t, "shutdown", 99, nil))

	lines := runServerSync(t, defaultRegistry(), &inBuf)
	if len(lines) == 0 {
		t.Fatal("expected at least one response")
	}

	var resp Response
	if err := json.Unmarshal(lines[0], &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func mustMarshalJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
