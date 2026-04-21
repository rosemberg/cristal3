package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"sync"

	"github.com/bergmaia/site-research/internal/tools"
)

// Server is the MCP server. It reads JSON-RPC requests from a MessageReader,
// dispatches them to the appropriate handler, and writes responses via a
// MessageWriter.
type Server struct {
	logger   *slog.Logger
	registry *tools.Registry
	reader   *MessageReader
	writer   *MessageWriter
	version  string // serverInfo.version, set at construction

	mu       sync.Mutex
	inflight map[string]context.CancelFunc // requestID string → cancel
	wg       sync.WaitGroup               // tracks in-flight handler goroutines
}

// NewServer creates a Server reading from r and writing to w.
// version is embedded in the initialize response as serverInfo.version.
func NewServer(logger *slog.Logger, registry *tools.Registry, r io.Reader, w io.Writer, version string) *Server {
	return &Server{
		logger:   logger,
		registry: registry,
		reader:   NewMessageReader(r),
		writer:   NewMessageWriter(w),
		version:  version,
		inflight: make(map[string]context.CancelFunc),
	}
}

// Run starts the request/response loop. It returns nil on clean EOF (stdin
// closed by the client) and a non-nil error only on unexpected I/O failures.
// Run waits for all in-flight handler goroutines to finish before returning.
func (s *Server) Run(ctx context.Context) error {
	defer s.wg.Wait() // drain in-flight handlers before returning

	for {
		// Check context cancellation between reads.
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err := s.reader.ReadLine()
		if err == io.EOF {
			s.logger.Info("stdin closed, shutting down")
			return nil
		}
		if err != nil {
			return fmt.Errorf("mcp: read: %w", err)
		}

		s.handleLine(ctx, line)
	}
}

// handleLine parses one line and dispatches it.
// All errors are handled internally; the method never returns an error to the
// loop (so a bad message does not kill the connection).
func (s *Server) handleLine(ctx context.Context, line []byte) {
	// Try to decode as a generic JSON object first to get the ID for error responses.
	var raw struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Method  string          `json:"method"`
	}
	if err := json.Unmarshal(line, &raw); err != nil {
		s.logger.Warn("parse error", "err", err)
		s.sendError(json.RawMessage(`null`), CodeParseError, "Parse error", nil)
		return
	}

	// Notifications have no ID and expect no response.
	isNotification := raw.ID == nil || string(raw.ID) == "null"

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		s.logger.Warn("invalid request", "err", err)
		if !isNotification {
			s.sendError(raw.ID, CodeInvalidRequest, "Invalid Request", nil)
		}
		return
	}

	switch req.Method {
	case "initialize":
		if isNotification {
			return
		}
		result, rpcErr := s.handleInitialize(ctx, req.Params)
		s.sendResult(req.ID, result, rpcErr)

	case "tools/list":
		if isNotification {
			return
		}
		result, rpcErr := s.handleToolsList(ctx)
		s.sendResult(req.ID, result, rpcErr)

	case "tools/call":
		if isNotification {
			return
		}
		// Dispatch asynchronously so multiple tools/call requests can run in
		// parallel. Each invocation gets its own cancellable context registered
		// in the inflight map so notifications/cancelled can reach it.
		s.dispatchToolsCall(ctx, req)

	case "notifications/cancelled":
		s.handleCancelled(ctx, req.Params)
		// No response for notifications.

	case "shutdown":
		if isNotification {
			return
		}
		s.handleShutdown(ctx)
		s.sendResult(req.ID, json.RawMessage(`null`), nil)

	default:
		s.logger.Warn("method not found", "method", req.Method)
		if !isNotification {
			s.sendError(req.ID, CodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
		}
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func (s *Server) handleInitialize(_ context.Context, params json.RawMessage) (any, *RPCError) {
	var p InitializeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid initialize params"}
	}

	if p.ProtocolVersion != ProtocolVersion {
		return nil, &RPCError{
			Code: CodeInvalidParams,
			Message: fmt.Sprintf(
				"unsupported protocol version %q; this server supports %s only — no fallback negotiation",
				p.ProtocolVersion, ProtocolVersion,
			),
		}
	}

	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo: ServerInfo{
			Name:    "site-research-mcp",
			Version: s.version,
		},
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
	}

	s.logger.Info("initialized",
		"client_name", p.ClientInfo.Name,
		"client_version", p.ClientInfo.Version,
		"protocol_version", p.ProtocolVersion,
	)
	return result, nil
}

func (s *Server) handleToolsList(_ context.Context) (any, *RPCError) {
	ts := s.registry.Tools()
	mcpTools := make([]Tool, 0, len(ts))
	for _, t := range ts {
		mcpTools = append(mcpTools, Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return ToolsListResult{Tools: mcpTools}, nil
}

// dispatchToolsCall executes the tools/call handler in a goroutine.
// The goroutine gets a cancellable context registered in s.inflight keyed by
// the request ID string. Panic recovery and context cleanup are handled inside.
func (s *Server) dispatchToolsCall(parentCtx context.Context, req Request) {
	// Validate params before spawning so we can return a synchronous error for
	// obviously bad requests.
	var p CallToolParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.sendError(req.ID, CodeInvalidParams, "invalid tools/call params", nil)
		return
	}
	if p.Name == "" {
		s.sendError(req.ID, CodeInvalidParams, "tools/call: name is required", nil)
		return
	}

	handler, ok := s.registry.Handler(p.Name)
	if !ok {
		s.sendError(req.ID, CodeMethodNotFound, fmt.Sprintf("tool not found: %s", p.Name), nil)
		return
	}

	// Create a per-request cancellable context derived from the server context.
	callCtx, cancel := context.WithCancel(parentCtx)

	// Register in the inflight map using the raw ID string as key.
	idKey := string(req.ID)
	s.mu.Lock()
	s.inflight[idKey] = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// Always remove from inflight and release the cancel on exit.
		defer func() {
			cancel()
			s.mu.Lock()
			delete(s.inflight, idKey)
			s.mu.Unlock()
		}()

		// Recover from panics in the tool handler.
		defer func() {
			if r := recover(); r != nil {
				stack := debug.Stack()
				s.logger.Error("panic in tool handler",
					"tool", p.Name,
					"request_id", idKey,
					"recover", fmt.Sprintf("%v", r),
					"stack", string(stack),
				)
				s.sendResult(req.ID, tools.CallToolResult{
					IsError: true,
					Content: []tools.ContentBlock{{
						Type: "text",
						Text: "**Erro:** erro interno no servidor; consulte logs para detalhes.",
					}},
				}, nil)
			}
		}()

		result, err := handler(callCtx, p.Arguments)
		if err != nil {
			s.logger.Error("tool handler error", "tool", p.Name, "err", err)
			s.sendResult(req.ID, tools.CallToolResult{
				IsError: true,
				Content: []tools.ContentBlock{{Type: "text", Text: fmt.Sprintf("tool %s error: %v", p.Name, err)}},
			}, nil)
			return
		}

		s.sendResult(req.ID, result, nil)
	}()
}

func (s *Server) handleCancelled(_ context.Context, params json.RawMessage) {
	var n CancelledNotification
	if err := json.Unmarshal(params, &n); err != nil {
		s.logger.Warn("invalid cancelled notification", "err", err)
		return
	}

	idKey := string(n.RequestID)
	s.mu.Lock()
	cancel, ok := s.inflight[idKey]
	if ok {
		delete(s.inflight, idKey)
	}
	s.mu.Unlock()

	if !ok {
		s.logger.Warn("received cancellation for unknown request", "request_id", idKey, "reason", n.Reason)
		return
	}

	s.logger.Info("cancelling request", "request_id", idKey, "reason", n.Reason)
	cancel()
}

func (s *Server) handleShutdown(_ context.Context) {
	s.logger.Info("shutdown requested")
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

func (s *Server) sendResult(id json.RawMessage, result any, rpcErr *RPCError) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
	}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		raw, err := json.Marshal(result)
		if err != nil {
			s.logger.Error("marshal result failed", "err", err)
			resp.Error = &RPCError{Code: CodeInternalError, Message: "internal marshal error"}
		} else {
			resp.Result = raw
		}
	}
	if err := s.writer.WriteMessage(resp); err != nil {
		s.logger.Error("write response failed", "err", err)
	}
}

func (s *Server) sendError(id json.RawMessage, code int, message string, data any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: message, Data: data},
	}
	if err := s.writer.WriteMessage(resp); err != nil {
		s.logger.Error("write error response failed", "err", err)
	}
}
