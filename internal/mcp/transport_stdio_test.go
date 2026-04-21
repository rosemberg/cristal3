package mcp

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"
)

func TestTransportRoundtrip(t *testing.T) {
	// Write 3 messages to a buffer, then read them back.
	var buf bytes.Buffer
	w := NewMessageWriter(&buf)

	msgs := []map[string]any{
		{"jsonrpc": "2.0", "id": 1, "method": "initialize"},
		{"jsonrpc": "2.0", "id": 2, "method": "tools/list"},
		{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": "search"}},
	}

	for _, m := range msgs {
		if err := w.WriteMessage(m); err != nil {
			t.Fatalf("WriteMessage: %v", err)
		}
	}

	r := NewMessageReader(&buf)
	for i, want := range msgs {
		line, err := r.ReadLine()
		if err != nil {
			t.Fatalf("message %d ReadLine: %v", i, err)
		}
		var got map[string]any
		if err := json.Unmarshal(line, &got); err != nil {
			t.Fatalf("message %d unmarshal: %v", i, err)
		}
		if got["method"] != want["method"] {
			t.Errorf("message %d: method got %q, want %q", i, got["method"], want["method"])
		}
	}
}

func TestTransportConcurrentWrites(t *testing.T) {
	// Concurrent writes must not interleave frames.
	var buf bytes.Buffer
	w := NewMessageWriter(&buf)

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			msg := map[string]any{
				"jsonrpc": "2.0",
				"id":      idx,
				"method":  "ping",
			}
			if err := w.WriteMessage(msg); err != nil {
				t.Errorf("WriteMessage: %v", err)
			}
		}(i)
	}
	wg.Wait()

	// Read all messages back; every line must be valid JSON.
	r := NewMessageReader(&buf)
	count := 0
	for {
		line, err := r.ReadLine()
		if err != nil {
			break
		}
		var obj map[string]any
		if err := json.Unmarshal(line, &obj); err != nil {
			t.Errorf("corrupt frame at message %d: %v\nline: %s", count, err, line)
		}
		count++
	}
	if count != n {
		t.Errorf("expected %d messages, got %d", n, count)
	}
}
