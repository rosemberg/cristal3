package mcp

import (
	"encoding/json"
	"testing"
)

func TestRequestMarshalRoundtrip(t *testing.T) {
	req := Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2025-11-25"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Request
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc: got %q, want %q", got.JSONRPC, "2.0")
	}
	if got.Method != req.Method {
		t.Errorf("method: got %q, want %q", got.Method, req.Method)
	}
	if string(got.ID) != string(req.ID) {
		t.Errorf("id: got %s, want %s", got.ID, req.ID)
	}
}

func TestResponseWithResultRoundtrip(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`42`),
		Result:  json.RawMessage(`{"ok":true}`),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify jsonrpc field is present in the output.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if string(raw["jsonrpc"]) != `"2.0"` {
		t.Errorf("jsonrpc field: got %s", raw["jsonrpc"])
	}
	if _, ok := raw["error"]; ok {
		t.Error("error field must be absent when result is set")
	}

	var got Response
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal Response: %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc: got %q", got.JSONRPC)
	}
	if string(got.Result) != `{"ok":true}` {
		t.Errorf("result: got %s", got.Result)
	}
}

func TestResponseWithErrorRoundtrip(t *testing.T) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`"abc"`),
		Error:   &RPCError{Code: CodeMethodNotFound, Message: "method not found"},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Response
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Error == nil {
		t.Fatal("error must not be nil")
	}
	if got.Error.Code != CodeMethodNotFound {
		t.Errorf("error code: got %d, want %d", got.Error.Code, CodeMethodNotFound)
	}
	if got.Result != nil {
		t.Errorf("result must be nil when error is set, got %s", got.Result)
	}
}

func TestNotificationRoundtrip(t *testing.T) {
	n := Notification{
		JSONRPC: "2.0",
		Method:  "notifications/cancelled",
		Params:  json.RawMessage(`{"requestId":1,"reason":"user cancelled"}`),
	}

	data, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Notification
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.JSONRPC != "2.0" {
		t.Errorf("jsonrpc: got %q", got.JSONRPC)
	}
	if got.Method != n.Method {
		t.Errorf("method: got %q, want %q", got.Method, n.Method)
	}

	// Notification must not have an "id" field.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if _, ok := raw["id"]; ok {
		t.Error("notification must not have an id field")
	}
}

func TestErrorCodes(t *testing.T) {
	cases := []struct {
		name string
		code int
	}{
		{"parse", CodeParseError},
		{"invalid_request", CodeInvalidRequest},
		{"method_not_found", CodeMethodNotFound},
		{"invalid_params", CodeInvalidParams},
		{"internal_error", CodeInternalError},
	}
	for _, tc := range cases {
		if tc.code >= -32600 && tc.name == "parse" {
			// parse error must be below -32600
			if tc.code != -32700 {
				t.Errorf("%s: expected -32700, got %d", tc.name, tc.code)
			}
		}
	}
}
