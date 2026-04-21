package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"time"
)

var (
	ErrTimeout          = errors.New("mcp: request timeout")
	ErrConnectionClosed = errors.New("mcp: connection closed")
)

// Client manages a single MCP server process via stdio
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr io.ReadCloser

	reqID      int
	reqIDMutex sync.Mutex

	pending      map[int]chan Response
	pendingMutex sync.RWMutex

	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

// ClientConfig configures a new MCP client
type ClientConfig struct {
	Command    string
	Args       []string
	WorkingDir string
	Env        map[string]string
	Timeout    time.Duration
	Logger     *slog.Logger
}

// NewClient creates and starts an MCP server process
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)
	cmd.Dir = cfg.WorkingDir

	// Set environment variables
	if len(cfg.Env) > 0 {
		cmd.Env = make([]string, 0, len(cfg.Env))
		for k, v := range cfg.Env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("start process: %w", err)
	}

	c := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewScanner(stdout),
		stderr:  stderr,
		reqID:   0,
		pending: make(map[int]chan Response),
		logger:  cfg.Logger,
		ctx:     ctx,
		cancel:  cancel,
	}

	go c.readLoop()
	go c.stderrLoop()

	return c, nil
}

// Initialize performs MCP handshake
func (c *Client) Initialize() error {
	req := InitializeRequest{
		ProtocolVersion: "2025-11-25",
		Capabilities:    map[string]interface{}{},
		ClientInfo: ClientInfo{
			Name:    "cristal-backend",
			Version: "0.1.0",
		},
	}

	var result InitializeResult
	if err := c.call("initialize", req, &result); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	c.logger.Info("MCP initialized",
		"server", result.ServerInfo.Name,
		"version", result.ServerInfo.Version)

	return nil
}

// ListTools returns available tools
func (c *Client) ListTools() ([]Tool, error) {
	var result struct {
		Tools []Tool `json:"tools"`
	}

	if err := c.call("tools/list", nil, &result); err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	return result.Tools, nil
}

// CallTool executes a tool
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*CallToolResult, error) {
	req := CallToolRequest{
		Name:      name,
		Arguments: args,
	}

	var result CallToolResult
	if err := c.callWithContext(ctx, "tools/call", req, &result); err != nil {
		return nil, fmt.Errorf("call tool %s: %w", name, err)
	}

	if result.IsError {
		errMsg := "unknown error"
		if len(result.Content) > 0 {
			errMsg = result.Content[0].Text
		}
		return nil, fmt.Errorf("tool error: %s", errMsg)
	}

	return &result, nil
}

// Close shuts down the client
func (c *Client) Close() error {
	c.cancel()

	if err := c.stdin.Close(); err != nil {
		c.logger.Warn("close stdin", "error", err)
	}

	if err := c.cmd.Wait(); err != nil {
		if c.ctx.Err() == context.Canceled {
			return nil
		}
		return fmt.Errorf("wait process: %w", err)
	}

	return nil
}

// call executes a request with response
func (c *Client) call(method string, params, result interface{}) error {
	return c.callWithContext(c.ctx, method, params, result)
}

// callWithContext executes a request with custom context
func (c *Client) callWithContext(ctx context.Context, method string, params, result interface{}) error {
	id := c.nextID()

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	respChan := make(chan Response, 1)
	c.setPending(id, respChan)
	defer c.deletePending(id)

	if err := c.send(req); err != nil {
		return err
	}

	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		if result != nil {
			if len(resp.Result) == 0 {
				return fmt.Errorf("empty result from MCP server")
			}

			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("unmarshal result: %w", err)
			}
		}

		return nil

	case <-ctx.Done():
		return ctx.Err()

	case <-c.ctx.Done():
		return ErrConnectionClosed
	}
}

// send writes a request to stdin
func (c *Client) send(req Request) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	data = append(data, '\n')

	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("write stdin: %w", err)
	}

	c.logger.Debug("sent request", "method", req.Method, "id", req.ID)
	return nil
}

// readLoop reads responses from stdout
func (c *Client) readLoop() {
	for c.stdout.Scan() {
		line := c.stdout.Bytes()

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			c.logger.Warn("invalid response", "error", err, "line", string(line))
			continue
		}

		c.logger.Debug("received response", "id", resp.ID)

		c.pendingMutex.RLock()
		ch, ok := c.pending[resp.ID]
		c.pendingMutex.RUnlock()

		if ok {
			select {
			case ch <- resp:
			case <-c.ctx.Done():
				return
			}
		}
	}

	if err := c.stdout.Err(); err != nil {
		c.logger.Error("stdout scan error", "error", err)
	}
}

// stderrLoop logs stderr from the process
func (c *Client) stderrLoop() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		c.logger.Debug("mcp stderr", "line", scanner.Text())
	}
}

// Helpers
func (c *Client) nextID() int {
	c.reqIDMutex.Lock()
	defer c.reqIDMutex.Unlock()
	c.reqID++
	return c.reqID
}

func (c *Client) setPending(id int, ch chan Response) {
	c.pendingMutex.Lock()
	defer c.pendingMutex.Unlock()
	c.pending[id] = ch
}

func (c *Client) deletePending(id int) {
	c.pendingMutex.Lock()
	defer c.pendingMutex.Unlock()
	delete(c.pending, id)
}
