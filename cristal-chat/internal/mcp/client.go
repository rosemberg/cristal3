package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
	"time"
)

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

type Config struct {
	PythonPath string
	ScriptPath string
	WorkingDir string
	Timeout    time.Duration
	Logger     *slog.Logger
}

// NewClient cria e inicia o processo MCP
func NewClient(cfg Config) (*Client, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, cfg.PythonPath, "-m", "src.server")
	cmd.Dir = cfg.WorkingDir

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

	// Goroutine para ler respostas
	go c.readLoop()

	// Goroutine para logar stderr
	go c.stderrLoop()

	return c, nil
}

// Initialize faz handshake MCP
func (c *Client) Initialize() error {
	req := InitializeRequest{
		ProtocolVersion: "2025-11-25",
		Capabilities:    map[string]interface{}{},
		ClientInfo: ClientInfo{
			Name:    "cristal-chat",
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

	// NOTE: notifications/initialized não é necessário para o protocol 2025-11-25
	// O servidor Python MCP não reconhece esse método
	// return c.notify("notifications/initialized", nil)
	return nil
}

// ListTools retorna tools disponíveis
func (c *Client) ListTools() ([]Tool, error) {
	var result struct {
		Tools []Tool `json:"tools"`
	}

	if err := c.call("tools/list", nil, &result); err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	return result.Tools, nil
}

// CallTool executa uma tool
func (c *Client) CallTool(name string, args map[string]interface{}) (*CallToolResult, error) {
	req := CallToolRequest{
		Name:      name,
		Arguments: args,
	}

	var result CallToolResult
	if err := c.call("tools/call", req, &result); err != nil {
		return nil, fmt.Errorf("call tool %s: %w", name, err)
	}

	if result.IsError {
		return nil, fmt.Errorf("tool error: %s", result.Content[0].Text)
	}

	return &result, nil
}

// Close encerra o cliente
func (c *Client) Close() error {
	c.cancel()

	if err := c.stdin.Close(); err != nil {
		c.logger.Warn("close stdin", "error", err)
	}

	if err := c.cmd.Wait(); err != nil {
		// Exit error é esperado quando cancelamos o context
		if c.ctx.Err() == context.Canceled {
			return nil
		}
		return fmt.Errorf("wait process: %w", err)
	}

	return nil
}

// call executa request com response esperado
func (c *Client) call(method string, params, result interface{}) error {
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
			// Debug: log raw result before unmarshal
			c.logger.Debug("unmarshaling result", "raw", string(resp.Result), "len", len(resp.Result))

			if len(resp.Result) == 0 {
				return fmt.Errorf("empty result from MCP server")
			}

			if err := json.Unmarshal(resp.Result, result); err != nil {
				c.logger.Error("unmarshal failed", "error", err, "raw", string(resp.Result))
				return fmt.Errorf("unmarshal result: %w", err)
			}
		}

		return nil

	case <-time.After(120 * time.Second):
		return ErrTimeout

	case <-c.ctx.Done():
		return ErrConnectionClosed
	}
}

// notify envia notification (sem resposta esperada)
func (c *Client) notify(method string, params interface{}) error {
	req := Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return c.send(req)
}

// send envia request via stdin
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

// readLoop lê respostas do stdout
func (c *Client) readLoop() {
	for c.stdout.Scan() {
		line := c.stdout.Bytes()

		var resp Response
		if err := json.Unmarshal(line, &resp); err != nil {
			c.logger.Warn("invalid response", "error", err, "line", string(line))
			continue
		}

		c.logger.Debug("received response", "id", resp.ID, "json", string(line))

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

// stderrLoop loga stderr do processo
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
