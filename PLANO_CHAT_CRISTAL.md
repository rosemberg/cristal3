# Cristal Chat - Plano de Implementação

**Projeto**: Cristal Chat CLI  
**Versão**: 0.1.0 (MVP)  
**Linguagem**: Go 1.22+  
**Data**: 2026-04-21  
**Autor**: Rosemberg Maia Gomes

---

## 1. Visão Geral

### 1.1 Objetivo

Implementar um chat de linha de comando em Go que se integra ao `data-orchestrator-mcp` (Python) via protocolo MCP sobre stdio, permitindo consultas interativas ao portal de transparência do TRE-PI com respostas bem formatadas e contexto conversacional.

### 1.2 Premissas

- O `data-orchestrator-mcp` já está implementado e funcional
- O `site-research-mcp` já está implementado e funcional
- O catálogo de ~573 páginas já está gerado e atualizado
- Desenvolvimento incremental com MVPs testáveis a cada sprint
- Foco em simplicidade e robustez no MVP

### 1.3 Não-Objetivos (Fora do Escopo do MVP)

- ❌ Agentes LLM complexos (planning, analysis, aggregation)
- ❌ Semantic caching ou embeddings
- ❌ TUI avançado com Bubble Tea
- ❌ Multi-turn conversation com contexto semântico
- ❌ Geração de relatórios/notebooks
- ❌ Interface web ou API REST

### 1.4 Arquitetura em Camadas

```
┌────────────────────────────────────┐
│     Cristal Chat (Go)              │  ← NOVA CAMADA (MVP)
│  - REPL interativo                 │
│  - MCP Client (stdio)              │
│  - Formatação de output            │
│  - Histórico de sessão             │
└────────────┬───────────────────────┘
             │ MCP Protocol (stdio/JSON-RPC)
             ↓
┌────────────────────────────────────┐
│  data-orchestrator-mcp (Python)    │  ← CAMADA 3 (Existente)
│  - research, get_cached            │
│  - get_document, metrics           │
│  - Extração de PDF/CSV/Excel       │
│  - Cache inteligente               │
└────────────┬───────────────────────┘
             │ MCP Protocol (stdio)
             ↓
┌────────────────────────────────────┐
│  site-research-mcp (Go)            │  ← CAMADA 2 (Existente)
│  - search, inspect_page            │
│  - catalog_stats                   │
│  - FTS5 SQLite                     │
└────────────┬───────────────────────┘
             │ Filesystem
             ↓
┌────────────────────────────────────┐
│  Data Layer                        │  ← CAMADA 1 (Existente)
│  - catalog.json                    │
│  - catalog.sqlite (FTS5)           │
│  - _index.json (hierárquico)       │
└────────────────────────────────────┘
```

---

## 2. Estrutura do Projeto

```
cristal-chat/
├── cmd/
│   └── cristal/
│       └── main.go                 # Entry point
├── internal/
│   ├── mcp/
│   │   ├── client.go              # Cliente MCP genérico (stdio)
│   │   ├── client_test.go         # Testes do client
│   │   ├── types.go               # Types MCP (Request, Response, Tool)
│   │   └── orchestrator.go        # Wrapper data-orchestrator específico
│   ├── chat/
│   │   ├── session.go             # Gerencia sessão do chat
│   │   ├── session_test.go
│   │   ├── history.go             # Histórico de mensagens
│   │   └── history_test.go
│   ├── ui/
│   │   ├── repl.go                # Loop REPL
│   │   ├── repl_test.go
│   │   ├── prompt.go              # Input handler
│   │   ├── formatter.go           # Output formatter (cores, tabelas)
│   │   └── formatter_test.go
│   └── config/
│       ├── config.go              # Configuração (paths, etc)
│       └── config_test.go
├── testdata/
│   ├── mcp_responses/             # Mock responses para testes
│   └── config.example.yaml
├── go.mod
├── go.sum
├── config.yaml                     # Config do usuário (gitignored)
├── config.example.yaml             # Template de config
├── README.md
├── PLANO_CHAT_CRISTAL.md          # Este arquivo
└── .gitignore
```

---

## 3. Roadmap de Implementação

### Sprint 1: Foundation (2-3 dias)
- **M1.1**: MCP Client básico
- **M1.2**: REPL mínimo

**Entregável**: Chat conecta, lista tools, faz queries básicas

### Sprint 2: Integration & Formatting (2-3 dias)
- **M2.1**: Orchestrator wrapper
- **M2.2**: Output formatter

**Entregável**: Chat com respostas bem formatadas

### Sprint 3: Context & History (2-3 dias)
- **M3.1**: História de conversação
- **M3.2**: Session manager

**Entregável**: Chat com memória de conversação

### Sprint 4: Polish & QoL (1-2 dias)
- **M4.1**: Configuração via YAML
- **M4.2**: Commands enhancement
- **M4.3**: Error handling & recovery

**Entregável**: MVP polido e robusto

---

## 4. Milestone 1.1: MCP Client Básico

### 4.1 Objetivos

- Conectar com data-orchestrator-mcp via stdio
- Implementar protocolo JSON-RPC 2.0 sobre newline-delimited JSON
- Fazer handshake MCP (initialize)
- Listar tools disponíveis
- Chamar tools com argumentos

### 4.2 Tipos de Dados

**Arquivo**: `internal/mcp/types.go`

```go
package mcp

import (
	"encoding/json"
	"time"
)

// JSON-RPC 2.0 Request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSON-RPC 2.0 Response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// JSON-RPC 2.0 Error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Initialize Request
type InitializeRequest struct {
	ProtocolVersion string      `json:"protocolVersion"`
	Capabilities    interface{} `json:"capabilities"`
	ClientInfo      ClientInfo  `json:"clientInfo"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// MCP Initialize Response
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	Capabilities    interface{}    `json:"capabilities"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// MCP Tool Definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCP CallTool Request
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// MCP CallTool Result
type CallToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Errors comuns
var (
	ErrConnectionClosed = &RPCError{Code: -32000, Message: "connection closed"}
	ErrTimeout          = &RPCError{Code: -32001, Message: "request timeout"}
	ErrInvalidResponse  = &RPCError{Code: -32002, Message: "invalid response format"}
)
```

### 4.3 Cliente MCP

**Arquivo**: `internal/mcp/client.go`

```go
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
		ProtocolVersion: "2024-11-05",
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
	
	// Enviar initialized notification
	return c.notify("notifications/initialized", nil)
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
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("unmarshal result: %w", err)
			}
		}
		
		return nil
		
	case <-time.After(30 * time.Second):
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
```

### 4.4 Testes

**Arquivo**: `internal/mcp/client_test.go`

```go
package mcp

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClientInitialize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	// Assumindo que data-orchestrator-mcp está no parent dir
	orchDir := filepath.Join("..", "..", "data-orchestrator-mcp")
	if _, err := os.Stat(orchDir); os.IsNotExist(err) {
		t.Skip("data-orchestrator-mcp not found")
	}
	
	cfg := Config{
		PythonPath: "python3",
		ScriptPath: "src.server",
		WorkingDir: orchDir,
		Timeout:    10 * time.Second,
	}
	
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()
	
	if err := client.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
}

func TestListTools(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	client := setupTestClient(t)
	defer client.Close()
	
	tools, err := client.ListTools()
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	
	expectedTools := []string{"research", "get_cached", "get_document", "metrics"}
	if len(tools) != len(expectedTools) {
		t.Errorf("expected %d tools, got %d", len(expectedTools), len(tools))
	}
	
	toolMap := make(map[string]bool)
	for _, tool := range tools {
		toolMap[tool.Name] = true
	}
	
	for _, name := range expectedTools {
		if !toolMap[name] {
			t.Errorf("expected tool %s not found", name)
		}
	}
}

func TestCallToolMetrics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	
	client := setupTestClient(t)
	defer client.Close()
	
	result, err := client.CallTool("metrics", nil)
	if err != nil {
		t.Fatalf("CallTool metrics: %v", err)
	}
	
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	
	if result.Content[0].Type != "text" {
		t.Errorf("expected text content, got %s", result.Content[0].Type)
	}
}

func setupTestClient(t *testing.T) *Client {
	orchDir := filepath.Join("..", "..", "data-orchestrator-mcp")
	
	cfg := Config{
		PythonPath: "python3",
		ScriptPath: "src.server",
		WorkingDir: orchDir,
		Timeout:    10 * time.Second,
	}
	
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	
	if err := client.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	
	return client
}
```

### 4.5 Critérios de Aceite

- ✅ CA-1.1: Cliente conecta ao data-orchestrator-mcp via stdio sem erros
- ✅ CA-1.2: Handshake MCP (initialize) completa com sucesso
- ✅ CA-1.3: ListTools retorna exatamente 4 tools: research, get_cached, get_document, metrics
- ✅ CA-1.4: CallTool("metrics", nil) retorna dados válidos
- ✅ CA-1.5: Close() encerra processo sem deixar processos órfãos
- ✅ CA-1.6: Testes passam com `go test ./internal/mcp/...`

---

## 5. Milestone 1.2: REPL Mínimo

### 5.1 Objetivos

- Loop interativo de input/output
- Comandos especiais: `/help`, `/quit`, `/tools`
- Query passthrough: qualquer entrada não-comando vai para `research`
- Prompt visual simples

### 5.2 REPL Core

**Arquivo**: `internal/ui/repl.go`

```go
package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	
	"github.com/bergmaia/cristal-chat/internal/mcp"
)

type REPL struct {
	client    *mcp.Client
	formatter *Formatter
	reader    *bufio.Reader
	logger    *slog.Logger
	running   bool
}

func NewREPL(client *mcp.Client, logger *slog.Logger) *REPL {
	return &REPL{
		client:    client,
		formatter: NewFormatter(true), // color enabled
		reader:    bufio.NewReader(os.Stdin),
		logger:    logger,
		running:   false,
	}
}

// Run inicia o loop REPL
func (r *REPL) Run(ctx context.Context) error {
	r.running = true
	defer func() { r.running = false }()
	
	r.printWelcome()
	
	for r.running {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		input, err := r.readInput()
		if err != nil {
			if err == io.EOF {
				break
			}
			r.logger.Error("read input", "error", err)
			continue
		}
		
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		
		if err := r.handleInput(input); err != nil {
			r.formatter.PrintError(err)
		}
	}
	
	r.printGoodbye()
	return nil
}

func (r *REPL) readInput() (string, error) {
	fmt.Print(r.formatter.Prompt())
	return r.reader.ReadString('\n')
}

func (r *REPL) handleInput(input string) error {
	// Comandos começam com /
	if strings.HasPrefix(input, "/") {
		return r.handleCommand(input)
	}
	
	// Caso contrário, é uma query
	return r.handleQuery(input)
}

func (r *REPL) handleCommand(input string) error {
	parts := strings.Fields(input)
	cmd := parts[0]
	args := parts[1:]
	
	switch cmd {
	case "/help", "/h":
		return r.cmdHelp(args)
	case "/quit", "/exit", "/q":
		return r.cmdQuit(args)
	case "/tools", "/t":
		return r.cmdTools(args)
	default:
		return fmt.Errorf("comando desconhecido: %s (use /help)", cmd)
	}
}

func (r *REPL) handleQuery(query string) error {
	r.formatter.PrintSearching()
	
	args := map[string]interface{}{
		"query":       query,
		"force_fetch": false,
	}
	
	result, err := r.client.CallTool("research", args)
	if err != nil {
		return fmt.Errorf("research: %w", err)
	}
	
	// Por enquanto, só imprime o texto bruto
	// M2.2 vai formatar isso bonitinho
	for _, content := range result.Content {
		if content.Type == "text" {
			fmt.Println(content.Text)
		}
	}
	
	return nil
}

// Comandos

func (r *REPL) cmdHelp(args []string) error {
	help := `
Cristal Chat - Comandos Disponíveis

COMANDOS:
  /help, /h              Mostra esta ajuda
  /quit, /exit, /q       Sai do chat
  /tools, /t             Lista tools do MCP disponíveis

CONSULTAS:
  Digite qualquer pergunta para buscar no portal de transparência.
  
  Exemplos:
    quanto foi gasto com diárias em 2026
    contratos de licitação
    balancetes de março

ATALHOS:
  Ctrl+C                 Sai do chat
  Ctrl+D                 Sai do chat (EOF)
`
	fmt.Println(help)
	return nil
}

func (r *REPL) cmdQuit(args []string) error {
	r.running = false
	return nil
}

func (r *REPL) cmdTools(args []string) error {
	tools, err := r.client.ListTools()
	if err != nil {
		return err
	}
	
	fmt.Println("\nTools Disponíveis:")
	for _, tool := range tools {
		fmt.Printf("  • %s\n", tool.Name)
		if tool.Description != "" {
			fmt.Printf("    %s\n", tool.Description)
		}
	}
	fmt.Println()
	
	return nil
}

// UI helpers

func (r *REPL) printWelcome() {
	fmt.Println(r.formatter.Logo())
	fmt.Println()
	fmt.Println("Cristal Chat v0.1.0")
	fmt.Println("Digite /help para ajuda ou faça sua pergunta.")
	fmt.Println()
}

func (r *REPL) printGoodbye() {
	fmt.Println()
	fmt.Println("Até logo! 👋")
}
```

### 5.3 Formatter Básico

**Arquivo**: `internal/ui/formatter.go`

```go
package ui

import (
	"fmt"
	"github.com/fatih/color"
)

type Formatter struct {
	colorEnabled bool
	
	// Cores
	primary   *color.Color
	secondary *color.Color
	error_    *color.Color
	success   *color.Color
	muted     *color.Color
}

func NewFormatter(colorEnabled bool) *Formatter {
	return &Formatter{
		colorEnabled: colorEnabled,
		primary:      color.New(color.FgCyan, color.Bold),
		secondary:    color.New(color.FgYellow),
		error_:       color.New(color.FgRed, color.Bold),
		success:      color.New(color.FgGreen),
		muted:        color.New(color.FgHiBlack),
	}
}

func (f *Formatter) Logo() string {
	if f.colorEnabled {
		return f.primary.Sprint("🔮 Cristal Chat")
	}
	return "Cristal Chat"
}

func (f *Formatter) Prompt() string {
	if f.colorEnabled {
		return f.primary.Sprint("🔮 > ")
	}
	return "> "
}

func (f *Formatter) PrintError(err error) {
	if f.colorEnabled {
		fmt.Println(f.error_.Sprintf("❌ Erro: %v", err))
	} else {
		fmt.Printf("Erro: %v\n", err)
	}
}

func (f *Formatter) PrintSearching() {
	if f.colorEnabled {
		fmt.Println(f.muted.Sprint("🔍 Pesquisando..."))
	} else {
		fmt.Println("Pesquisando...")
	}
}

// Será expandido em M2.2
```

### 5.4 Main Entry Point

**Arquivo**: `cmd/cristal/main.go`

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	
	"github.com/bergmaia/cristal-chat/internal/mcp"
	"github.com/bergmaia/cristal-chat/internal/ui"
)

var version = "0.1.0-dev"

func main() {
	os.Exit(run())
}

func run() int {
	// Flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	versionFlag := flag.Bool("version", false, "print version")
	debugFlag := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()
	
	if *versionFlag {
		fmt.Printf("cristal v%s\n", version)
		return 0
	}
	
	// Logger
	logLevel := slog.LevelInfo
	if *debugFlag {
		logLevel = slog.LevelDebug
	}
	
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	
	// Por enquanto, config hardcoded (M4.1 vai ler YAML)
	_ = configPath
	
	// MCP Client config
	mcpCfg := mcp.Config{
		PythonPath: "python3",
		ScriptPath: "src.server",
		WorkingDir: "../data-orchestrator-mcp", // TODO: ler do config
		Logger:     logger,
	}
	
	// Conectar ao MCP
	logger.Info("conectando ao data-orchestrator-mcp")
	client, err := mcp.NewClient(mcpCfg)
	if err != nil {
		logger.Error("falha ao criar cliente MCP", "error", err)
		return 1
	}
	defer client.Close()
	
	if err := client.Initialize(); err != nil {
		logger.Error("falha ao inicializar MCP", "error", err)
		return 1
	}
	
	logger.Info("MCP inicializado com sucesso")
	
	// REPL
	repl := ui.NewREPL(client, logger)
	
	// Context com signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), 
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	
	if err := repl.Run(ctx); err != nil {
		if err != context.Canceled {
			logger.Error("erro no REPL", "error", err)
			return 1
		}
	}
	
	return 0
}
```

### 5.5 Critérios de Aceite

- ✅ CA-1.7: Executar `cristal` mostra welcome e prompt
- ✅ CA-1.8: `/help` mostra lista de comandos
- ✅ CA-1.9: `/tools` mostra 4 tools do data-orchestrator
- ✅ CA-1.10: `/quit` encerra o programa gracefully
- ✅ CA-1.11: Query "teste" chama `research` e exibe resultado
- ✅ CA-1.12: Ctrl+C encerra o programa gracefully
- ✅ CA-1.13: Processo MCP é terminado ao sair

---

## 6. Milestone 2.1: Orchestrator Wrapper

### 6.1 Objetivos

- Abstrair os 4 tools do data-orchestrator
- Tipos estruturados para requests/responses
- Parse de JSON responses para structs Go

### 6.2 Tipos do Orchestrator

**Arquivo**: `internal/mcp/orchestrator.go`

```go
package mcp

import (
	"encoding/json"
	"fmt"
	"time"
)

type Orchestrator struct {
	client *Client
}

func NewOrchestrator(client *Client) *Orchestrator {
	return &Orchestrator{client: client}
}

// Research Types

type ResearchRequest struct {
	Query      string `json:"query"`
	ForceFetch bool   `json:"force_fetch"`
}

type ResearchResponse struct {
	Summary    string       `json:"summary"`
	Pages      []Page       `json:"pages"`
	Documents  []Document   `json:"documents"`
	TotalValue *MoneyValue  `json:"total_value,omitempty"`
	FromCache  bool         `json:"from_cache"`
	Timestamp  time.Time    `json:"timestamp"`
}

type Page struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Section     string `json:"section"`
	MiniSummary string `json:"mini_summary"`
	PageType    string `json:"page_type"`
}

type Document struct {
	URL         string    `json:"url"`
	Type        string    `json:"type"` // pdf, csv, xlsx
	SizeBytes   int64     `json:"size_bytes"`
	Pages       int       `json:"pages,omitempty"`
	ExtractedAt time.Time `json:"extracted_at,omitempty"`
}

type MoneyValue struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Formatted string `json:"formatted"`
}

// Research executa busca completa
func (o *Orchestrator) Research(query string, forceFetch bool) (*ResearchResponse, error) {
	args := map[string]interface{}{
		"query":       query,
		"force_fetch": forceFetch,
	}
	
	result, err := o.client.CallTool("research", args)
	if err != nil {
		return nil, err
	}
	
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	
	var resp ResearchResponse
	if err := json.Unmarshal([]byte(result.Content[0].Text), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	
	return &resp, nil
}

// GetCached retorna dados do cache
func (o *Orchestrator) GetCached(query string) (*ResearchResponse, error) {
	args := map[string]interface{}{
		"query": query,
	}
	
	result, err := o.client.CallTool("get_cached", args)
	if err != nil {
		return nil, err
	}
	
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("not in cache")
	}
	
	var resp ResearchResponse
	if err := json.Unmarshal([]byte(result.Content[0].Text), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	
	return &resp, nil
}

// GetDocument baixa e extrai documento
func (o *Orchestrator) GetDocument(url string) (*DocumentData, error) {
	args := map[string]interface{}{
		"url": url,
	}
	
	result, err := o.client.CallTool("get_document", args)
	if err != nil {
		return nil, err
	}
	
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	
	var data DocumentData
	if err := json.Unmarshal([]byte(result.Content[0].Text), &data); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	
	return &data, nil
}

type DocumentData struct {
	URL        string            `json:"url"`
	Type       string            `json:"type"`
	Content    string            `json:"content,omitempty"`
	Values     []float64         `json:"values,omitempty"`
	Metadata   map[string]string `json:"metadata"`
}

// GetMetrics retorna métricas do sistema
func (o *Orchestrator) GetMetrics() (*Metrics, error) {
	result, err := o.client.CallTool("metrics", nil)
	if err != nil {
		return nil, err
	}
	
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response")
	}
	
	var metrics Metrics
	if err := json.Unmarshal([]byte(result.Content[0].Text), &metrics); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	
	return &metrics, nil
}

type Metrics struct {
	Uptime           string  `json:"uptime"`
	CacheHits        int64   `json:"cache_hits"`
	CacheMisses      int64   `json:"cache_misses"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	QueriesProcessed int64   `json:"queries_processed"`
	DocsExtracted    int64   `json:"docs_extracted"`
	DocsErrors       int64   `json:"docs_errors"`
	BytesTransferred int64   `json:"bytes_transferred"`
	TotalErrors      int64   `json:"total_errors"`
}
```

### 6.3 Integração no REPL

**Atualizar**: `internal/ui/repl.go`

```go
// Adicionar campo
type REPL struct {
	client       *mcp.Client
	orchestrator *mcp.Orchestrator  // NOVO
	formatter    *Formatter
	// ...
}

// Atualizar construtor
func NewREPL(client *mcp.Client, logger *slog.Logger) *REPL {
	return &REPL{
		client:       client,
		orchestrator: mcp.NewOrchestrator(client),  // NOVO
		// ...
	}
}

// Atualizar handleQuery
func (r *REPL) handleQuery(query string) error {
	r.formatter.PrintSearching()
	
	resp, err := r.orchestrator.Research(query, false)
	if err != nil {
		return fmt.Errorf("research: %w", err)
	}
	
	// M2.2 vai formatar isso
	fmt.Printf("\nSummary: %s\n", resp.Summary)
	fmt.Printf("Pages: %d\n", len(resp.Pages))
	fmt.Printf("Documents: %d\n", len(resp.Documents))
	if resp.TotalValue != nil {
		fmt.Printf("Total: %s\n", resp.TotalValue.Formatted)
	}
	
	return nil
}
```

### 6.4 Novos Comandos

Adicionar em `handleCommand`:

```go
case "/cache":
	return r.cmdCache(args)
case "/metrics", "/m":
	return r.cmdMetrics(args)
```

```go
func (r *REPL) cmdCache(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: /cache <query>")
	}
	
	query := strings.Join(args, " ")
	resp, err := r.orchestrator.GetCached(query)
	if err != nil {
		return fmt.Errorf("não encontrado no cache")
	}
	
	fmt.Printf("Cache hit! From: %s\n", resp.Timestamp.Format(time.RFC822))
	return nil
}

func (r *REPL) cmdMetrics(args []string) error {
	metrics, err := r.orchestrator.GetMetrics()
	if err != nil {
		return err
	}
	
	// M2.2 vai formatar isso
	fmt.Printf("\nMétricas:\n")
	fmt.Printf("  Uptime: %s\n", metrics.Uptime)
	fmt.Printf("  Cache hit rate: %.1f%%\n", metrics.CacheHitRate*100)
	fmt.Printf("  Queries: %d\n", metrics.QueriesProcessed)
	fmt.Printf("  Docs extraídos: %d\n", metrics.DocsExtracted)
	
	return nil
}
```

### 6.5 Critérios de Aceite

- ✅ CA-2.1: `Research()` retorna struct tipado com páginas e documentos
- ✅ CA-2.2: `GetCached()` retorna dados se em cache, erro caso contrário
- ✅ CA-2.3: `GetMetrics()` retorna métricas estruturadas
- ✅ CA-2.4: `/metrics` exibe uptime, cache hit rate, contadores
- ✅ CA-2.5: `/cache <query>` verifica se query está em cache

---

## 7. Milestone 2.2: Output Formatter

### 7.1 Objetivos

- Respostas coloridas e estruturadas
- Tabelas ASCII para listas
- Ícones e emojis para visual appeal
- Formatação de valores monetários

### 7.2 Formatter Completo

**Atualizar**: `internal/ui/formatter.go`

```go
package ui

import (
	"fmt"
	"strings"
	
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/bergmaia/cristal-chat/internal/mcp"
)

type Formatter struct {
	colorEnabled bool
	
	// Cores
	primary   *color.Color
	secondary *color.Color
	error_    *color.Color
	success   *color.Color
	muted     *color.Color
	money     *color.Color
}

func NewFormatter(colorEnabled bool) *Formatter {
	return &Formatter{
		colorEnabled: colorEnabled,
		primary:      color.New(color.FgCyan, color.Bold),
		secondary:    color.New(color.FgYellow),
		error_:       color.New(color.FgRed, color.Bold),
		success:      color.New(color.FgGreen),
		muted:        color.New(color.FgHiBlack),
		money:        color.New(color.FgGreen, color.Bold),
	}
}

// FormatResearch formata resposta completa de research
func (f *Formatter) FormatResearch(resp *mcp.ResearchResponse) string {
	var b strings.Builder
	
	b.WriteString(f.divider())
	b.WriteString(f.header("📊 Resultados da Pesquisa"))
	b.WriteString(f.divider())
	b.WriteString("\n")
	
	// Summary
	if resp.Summary != "" {
		b.WriteString(f.summary(resp.Summary))
		b.WriteString("\n")
	}
	
	// Total value
	if resp.TotalValue != nil {
		b.WriteString(f.totalValue(resp.TotalValue))
		b.WriteString("\n")
	}
	
	// Pages
	if len(resp.Pages) > 0 {
		b.WriteString(f.pages(resp.Pages))
		b.WriteString("\n")
	}
	
	// Documents
	if len(resp.Documents) > 0 {
		b.WriteString(f.documents(resp.Documents))
		b.WriteString("\n")
	}
	
	// Cache info
	if resp.FromCache {
		b.WriteString(f.cacheInfo())
		b.WriteString("\n")
	}
	
	b.WriteString(f.divider())
	
	return b.String()
}

// FormatMetrics formata métricas do sistema
func (f *Formatter) FormatMetrics(m *mcp.Metrics) string {
	var b strings.Builder
	
	b.WriteString(f.divider())
	b.WriteString(f.header("📈 Métricas do Sistema"))
	b.WriteString(f.divider())
	b.WriteString("\n")
	
	table := [][]string{
		{"Uptime", m.Uptime},
		{"Cache hit rate", fmt.Sprintf("%.1f%%", m.CacheHitRate*100)},
		{"Queries processadas", fmt.Sprintf("%d", m.QueriesProcessed)},
		{"Documentos extraídos", fmt.Sprintf("%d", m.DocsExtracted)},
		{"Erros de extração", fmt.Sprintf("%d", m.DocsErrors)},
		{"Bytes transferidos", f.formatBytes(m.BytesTransferred)},
		{"Total de erros", fmt.Sprintf("%d", m.TotalErrors)},
	}
	
	for _, row := range table {
		b.WriteString(fmt.Sprintf("  %s: %s\n", 
			f.muted.Sprint(row[0]), 
			f.success.Sprint(row[1])))
	}
	
	b.WriteString("\n")
	b.WriteString(f.divider())
	
	return b.String()
}

// Helpers internos

func (f *Formatter) divider() string {
	return f.muted.Sprintln(strings.Repeat("━", 60))
}

func (f *Formatter) header(text string) string {
	return f.primary.Sprintln(text)
}

func (f *Formatter) summary(text string) string {
	return fmt.Sprintf("📝 %s\n", text)
}

func (f *Formatter) totalValue(mv *mcp.MoneyValue) string {
	return fmt.Sprintf("💰 Total encontrado: %s\n", 
		f.money.Sprint(mv.Formatted))
}

func (f *Formatter) pages(pages []mcp.Page) string {
	var b strings.Builder
	
	b.WriteString(f.secondary.Sprintf("📄 Páginas encontradas (%d):\n", len(pages)))
	
	for i, page := range pages {
		b.WriteString(fmt.Sprintf("  %d. %s\n", i+1, f.primary.Sprint(page.Title)))
		b.WriteString(fmt.Sprintf("     %s\n", f.muted.Sprint(page.URL)))
		if page.MiniSummary != "" {
			b.WriteString(fmt.Sprintf("     » %s\n", page.MiniSummary))
		}
		b.WriteString("\n")
	}
	
	return b.String()
}

func (f *Formatter) documents(docs []mcp.Document) string {
	var b strings.Builder
	
	b.WriteString(f.secondary.Sprintf("📎 Documentos analisados (%d):\n", len(docs)))
	
	for _, doc := range docs {
		b.WriteString(fmt.Sprintf("  • %s (%s)\n", 
			f.getFilename(doc.URL),
			f.formatBytes(doc.SizeBytes)))
		if doc.Pages > 0 {
			b.WriteString(fmt.Sprintf("    %d páginas\n", doc.Pages))
		}
	}
	
	b.WriteString("\n")
	return b.String()
}

func (f *Formatter) cacheInfo() string {
	return f.muted.Sprint("⚡ Resultado do cache\n")
}

func (f *Formatter) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (f *Formatter) getFilename(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return url
}
```

### 7.3 Integração no REPL

**Atualizar**: `internal/ui/repl.go`

```go
func (r *REPL) handleQuery(query string) error {
	r.formatter.PrintSearching()
	
	resp, err := r.orchestrator.Research(query, false)
	if err != nil {
		return fmt.Errorf("research: %w", err)
	}
	
	// Formatar e exibir
	fmt.Println(r.formatter.FormatResearch(resp))
	
	return nil
}

func (r *REPL) cmdMetrics(args []string) error {
	metrics, err := r.orchestrator.GetMetrics()
	if err != nil {
		return err
	}
	
	fmt.Println(r.formatter.FormatMetrics(metrics))
	return nil
}
```

### 7.4 Critérios de Aceite

- ✅ CA-2.6: Respostas exibem cores e emojis quando terminal suporta
- ✅ CA-2.7: Páginas mostram título, URL e mini-summary formatados
- ✅ CA-2.8: Documentos mostram nome e tamanho legível (KB/MB)
- ✅ CA-2.9: Valores monetários aparecem em verde e formatados (R$ 1.234,56)
- ✅ CA-2.10: Métricas exibem tabela limpa com labels e valores

---

## 8. Milestone 3.1: História de Conversação

### 8.1 Objetivos

- Armazenar mensagens user/assistant
- Comandos para ver histórico
- Salvar/carregar histórico em JSON
- Limite de mensagens (rolling window)

### 8.2 História

**Arquivo**: `internal/chat/history.go`

```go
package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Message struct {
	Role      string                 `json:"role"` // "user", "assistant"
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type History struct {
	messages []Message
	maxSize  int
}

func NewHistory(maxSize int) *History {
	if maxSize <= 0 {
		maxSize = 50
	}
	return &History{
		messages: make([]Message, 0, maxSize),
		maxSize:  maxSize,
	}
}

// Add adiciona mensagem ao histórico
func (h *History) Add(role, content string, meta map[string]interface{}) {
	msg := Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		Metadata:  meta,
	}
	
	h.messages = append(h.messages, msg)
	
	// Rolling window: remove antigas se exceder maxSize
	if len(h.messages) > h.maxSize {
		h.messages = h.messages[len(h.messages)-h.maxSize:]
	}
}

// GetLast retorna últimas n mensagens
func (h *History) GetLast(n int) []Message {
	if n <= 0 || n > len(h.messages) {
		n = len(h.messages)
	}
	
	start := len(h.messages) - n
	result := make([]Message, n)
	copy(result, h.messages[start:])
	
	return result
}

// GetAll retorna todas as mensagens
func (h *History) GetAll() []Message {
	result := make([]Message, len(h.messages))
	copy(result, h.messages)
	return result
}

// Clear limpa o histórico
func (h *History) Clear() {
	h.messages = h.messages[:0]
}

// Count retorna número de mensagens
func (h *History) Count() int {
	return len(h.messages)
}

// Save salva histórico em arquivo JSON
func (h *History) Save(path string) error {
	data, err := json.MarshalIndent(h.messages, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	
	return nil
}

// Load carrega histórico de arquivo JSON
func (h *History) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	
	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	
	// Respeita maxSize
	if len(messages) > h.maxSize {
		messages = messages[len(messages)-h.maxSize:]
	}
	
	h.messages = messages
	return nil
}
```

### 8.3 Testes

**Arquivo**: `internal/chat/history_test.go`

```go
package chat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryAdd(t *testing.T) {
	h := NewHistory(5)
	
	h.Add("user", "hello", nil)
	h.Add("assistant", "hi there", nil)
	
	if h.Count() != 2 {
		t.Errorf("expected 2 messages, got %d", h.Count())
	}
}

func TestHistoryRollingWindow(t *testing.T) {
	h := NewHistory(3)
	
	for i := 0; i < 5; i++ {
		h.Add("user", string(rune('a'+i)), nil)
	}
	
	if h.Count() != 3 {
		t.Errorf("expected 3 messages, got %d", h.Count())
	}
	
	// Deve manter últimas 3: c, d, e
	msgs := h.GetAll()
	if msgs[0].Content != "c" {
		t.Errorf("expected 'c', got %s", msgs[0].Content)
	}
}

func TestHistoryGetLast(t *testing.T) {
	h := NewHistory(10)
	
	for i := 0; i < 5; i++ {
		h.Add("user", string(rune('a'+i)), nil)
	}
	
	last2 := h.GetLast(2)
	if len(last2) != 2 {
		t.Errorf("expected 2 messages, got %d", len(last2))
	}
	
	if last2[0].Content != "d" || last2[1].Content != "e" {
		t.Errorf("expected [d, e], got [%s, %s]", last2[0].Content, last2[1].Content)
	}
}

func TestHistorySaveLoad(t *testing.T) {
	h := NewHistory(10)
	h.Add("user", "test1", nil)
	h.Add("assistant", "test2", nil)
	
	tmpfile := filepath.Join(t.TempDir(), "history.json")
	
	if err := h.Save(tmpfile); err != nil {
		t.Fatalf("Save: %v", err)
	}
	
	h2 := NewHistory(10)
	if err := h2.Load(tmpfile); err != nil {
		t.Fatalf("Load: %v", err)
	}
	
	if h2.Count() != 2 {
		t.Errorf("expected 2 messages after load, got %d", h2.Count())
	}
}

func TestHistoryClear(t *testing.T) {
	h := NewHistory(10)
	h.Add("user", "test", nil)
	
	h.Clear()
	
	if h.Count() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", h.Count())
	}
}
```

### 8.4 Integração no REPL

**Atualizar**: `internal/ui/repl.go`

```go
import "github.com/bergmaia/cristal-chat/internal/chat"

type REPL struct {
	client       *mcp.Client
	orchestrator *mcp.Orchestrator
	formatter    *Formatter
	history      *chat.History  // NOVO
	reader       *bufio.Reader
	logger       *slog.Logger
	running      bool
}

func NewREPL(client *mcp.Client, historySize int, logger *slog.Logger) *REPL {
	return &REPL{
		client:       client,
		orchestrator: mcp.NewOrchestrator(client),
		formatter:    NewFormatter(true),
		history:      chat.NewHistory(historySize),  // NOVO
		reader:       bufio.NewReader(os.Stdin),
		logger:       logger,
		running:      false,
	}
}

func (r *REPL) handleQuery(query string) error {
	// Adiciona query do usuário ao histórico
	r.history.Add("user", query, nil)
	
	r.formatter.PrintSearching()
	
	resp, err := r.orchestrator.Research(query, false)
	if err != nil {
		return fmt.Errorf("research: %w", err)
	}
	
	output := r.formatter.FormatResearch(resp)
	fmt.Println(output)
	
	// Adiciona resposta ao histórico
	r.history.Add("assistant", resp.Summary, map[string]interface{}{
		"pages":     len(resp.Pages),
		"documents": len(resp.Documents),
	})
	
	return nil
}

// Novos comandos

case "/history", "/hist":
	return r.cmdHistory(args)
case "/save":
	return r.cmdSave(args)
case "/load":
	return r.cmdLoad(args)
case "/clear":
	return r.cmdClear(args)

func (r *REPL) cmdHistory(args []string) error {
	n := 10
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &n)
	}
	
	msgs := r.history.GetLast(n)
	
	fmt.Printf("\n📜 Histórico (últimas %d mensagens):\n\n", len(msgs))
	
	for _, msg := range msgs {
		timestamp := msg.Timestamp.Format("15:04")
		role := msg.Role
		
		fmt.Printf("  [%s] %s: %s\n", timestamp, role, msg.Content)
	}
	
	fmt.Println()
	return nil
}

func (r *REPL) cmdSave(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: /save <arquivo>")
	}
	
	path := args[0]
	if err := r.history.Save(path); err != nil {
		return fmt.Errorf("salvar histórico: %w", err)
	}
	
	fmt.Printf("✅ Histórico salvo em %s\n", path)
	return nil
}

func (r *REPL) cmdLoad(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("uso: /load <arquivo>")
	}
	
	path := args[0]
	if err := r.history.Load(path); err != nil {
		return fmt.Errorf("carregar histórico: %w", err)
	}
	
	fmt.Printf("✅ Histórico carregado de %s (%d mensagens)\n", 
		path, r.history.Count())
	return nil
}

func (r *REPL) cmdClear(args []string) error {
	r.history.Clear()
	fmt.Println("✅ Histórico limpo")
	return nil
}
```

### 8.5 Critérios de Aceite

- ✅ CA-3.1: Cada query e resposta é adicionada ao histórico
- ✅ CA-3.2: `/history` mostra últimas 10 mensagens por padrão
- ✅ CA-3.3: `/history 20` mostra últimas 20 mensagens
- ✅ CA-3.4: `/save hist.json` salva histórico em JSON
- ✅ CA-3.5: `/load hist.json` carrega histórico e continua sessão
- ✅ CA-3.6: `/clear` limpa histórico mas mantém sessão ativa
- ✅ CA-3.7: Histórico respeita limite (maxSize), remove antigas

---

## 9. Milestone 3.2: Session Manager

### 9.1 Objetivos

- Gerenciar estado da sessão
- Contexto livre (key-value store)
- Summary da sessão
- Auto-save ao sair

### 9.2 Session

**Arquivo**: `internal/chat/session.go`

```go
package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
	
	"github.com/google/uuid"
)

type Session struct {
	ID           string                 `json:"id"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time,omitempty"`
	History      *History               `json:"-"` // não serializar
	Context      map[string]interface{} `json:"context"`
	LastQuery    string                 `json:"last_query"`
	QueryCount   int                    `json:"query_count"`
	HistoryCount int                    `json:"history_count"`
}

func NewSession(historySize int) *Session {
	return &Session{
		ID:        uuid.New().String(),
		StartTime: time.Now(),
		History:   NewHistory(historySize),
		Context:   make(map[string]interface{}),
	}
}

// AddUserMessage adiciona mensagem do usuário
func (s *Session) AddUserMessage(content string) {
	s.History.Add("user", content, nil)
	s.LastQuery = content
	s.QueryCount++
	s.HistoryCount = s.History.Count()
}

// AddAssistantMessage adiciona mensagem do assistente
func (s *Session) AddAssistantMessage(content string, meta map[string]interface{}) {
	s.History.Add("assistant", content, meta)
	s.HistoryCount = s.History.Count()
}

// GetContext retorna valor do contexto
func (s *Session) GetContext(key string) (interface{}, bool) {
	val, ok := s.Context[key]
	return val, ok
}

// SetContext define valor no contexto
func (s *Session) SetContext(key string, value interface{}) {
	s.Context[key] = value
}

// Duration retorna duração da sessão
func (s *Session) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// End marca fim da sessão
func (s *Session) End() {
	s.EndTime = time.Now()
}

// Summary retorna resumo da sessão
func (s *Session) Summary() string {
	duration := s.Duration()
	
	return fmt.Sprintf(`
Sessão: %s
Duração: %s
Queries: %d
Mensagens: %d
`, s.ID[:8], duration.Round(time.Second), s.QueryCount, s.HistoryCount)
}

// Save salva sessão + histórico
func (s *Session) Save(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	
	// Salvar metadata da sessão
	sessionPath := filepath.Join(dir, fmt.Sprintf("session_%s.json", s.ID))
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := os.WriteFile(sessionPath, data, 0644); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	
	// Salvar histórico
	historyPath := filepath.Join(dir, fmt.Sprintf("history_%s.json", s.ID))
	if err := s.History.Save(historyPath); err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	
	return nil
}

// Load carrega sessão + histórico
func Load(dir, sessionID string) (*Session, error) {
	sessionPath := filepath.Join(dir, fmt.Sprintf("session_%s.json", sessionID))
	
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}
	
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	
	// Carregar histórico
	historyPath := filepath.Join(dir, fmt.Sprintf("history_%s.json", sessionID))
	s.History = NewHistory(50) // default size
	if err := s.History.Load(historyPath); err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}
	
	return &s, nil
}
```

### 9.3 Integração no REPL

**Atualizar**: `internal/ui/repl.go`

```go
type REPL struct {
	client       *mcp.Client
	orchestrator *mcp.Orchestrator
	formatter    *Formatter
	session      *chat.Session  // NOVO (substitui history standalone)
	reader       *bufio.Reader
	logger       *slog.Logger
	running      bool
	sessionDir   string         // NOVO
}

func NewREPL(client *mcp.Client, historySize int, sessionDir string, logger *slog.Logger) *REPL {
	return &REPL{
		client:       client,
		orchestrator: mcp.NewOrchestrator(client),
		formatter:    NewFormatter(true),
		session:      chat.NewSession(historySize),
		reader:       bufio.NewReader(os.Stdin),
		logger:       logger,
		running:      false,
		sessionDir:   sessionDir,
	}
}

func (r *REPL) handleQuery(query string) error {
	r.session.AddUserMessage(query)
	
	r.formatter.PrintSearching()
	
	resp, err := r.orchestrator.Research(query, false)
	if err != nil {
		return fmt.Errorf("research: %w", err)
	}
	
	output := r.formatter.FormatResearch(resp)
	fmt.Println(output)
	
	r.session.AddAssistantMessage(resp.Summary, map[string]interface{}{
		"pages":     len(resp.Pages),
		"documents": len(resp.Documents),
	})
	
	return nil
}

// Atualizar printGoodbye
func (r *REPL) printGoodbye() {
	r.session.End()
	
	fmt.Println()
	fmt.Println("👋 Encerrando sessão...")
	fmt.Println(r.session.Summary())
	
	// Auto-save
	if r.sessionDir != "" {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		saveDir := filepath.Join(r.sessionDir, timestamp)
		
		if err := r.session.Save(saveDir); err != nil {
			r.logger.Warn("falha ao salvar sessão", "error", err)
		} else {
			fmt.Printf("Sessão salva em %s\n", saveDir)
		}
	}
	
	fmt.Println("\nAté logo!")
}

// Novo comando
case "/session", "/s":
	return r.cmdSession(args)

func (r *REPL) cmdSession(args []string) error {
	fmt.Println(r.session.Summary())
	return nil
}

// Atualizar /history para usar session
func (r *REPL) cmdHistory(args []string) error {
	n := 10
	if len(args) > 0 {
		fmt.Sscanf(args[0], "%d", &n)
	}
	
	msgs := r.session.History.GetLast(n)
	
	fmt.Printf("\n📜 Histórico (últimas %d mensagens):\n\n", len(msgs))
	
	for _, msg := range msgs {
		timestamp := msg.Timestamp.Format("15:04")
		role := msg.Role
		
		// Truncar conteúdo longo
		content := msg.Content
		if len(content) > 80 {
			content = content[:77] + "..."
		}
		
		fmt.Printf("  [%s] %s: %s\n", timestamp, role, content)
	}
	
	fmt.Println()
	return nil
}
```

### 9.4 Critérios de Aceite

- ✅ CA-3.8: Cada sessão tem ID único (UUID)
- ✅ CA-3.9: `/session` mostra resumo: duração, queries, mensagens
- ✅ CA-3.10: Context permite armazenar valores arbitrários
- ✅ CA-3.11: Ao sair, sessão é auto-salva em `~/.cristal/sessions/<timestamp>/`
- ✅ CA-3.12: Sessão salva inclui metadata + histórico completo

---

## 10. Milestone 4.1: Configuração

### 10.1 Objetivos

- Config via arquivo YAML
- Overrides via env vars e flags
- Validação de config
- Template de config

### 10.2 Config Types

**Arquivo**: `internal/config/config.go`

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	
	"gopkg.in/yaml.v3"
)

type Config struct {
	MCP  MCPConfig  `yaml:"mcp"`
	Chat ChatConfig `yaml:"chat"`
	UI   UIConfig   `yaml:"ui"`
}

type MCPConfig struct {
	DataOrchestrator DataOrchestratorConfig `yaml:"data_orchestrator"`
}

type DataOrchestratorConfig struct {
	PythonPath string `yaml:"python_path"`
	ScriptPath string `yaml:"script_path"`
	WorkingDir string `yaml:"working_dir"`
	Timeout    int    `yaml:"timeout"` // segundos
}

type ChatConfig struct {
	HistorySize int    `yaml:"history_size"`
	SaveHistory bool   `yaml:"save_history"`
	SessionDir  string `yaml:"session_dir"`
}

type UIConfig struct {
	Color          bool   `yaml:"color"`
	ShowTimestamps bool   `yaml:"show_timestamps"`
	Prompt         string `yaml:"prompt"`
}

// Load carrega config de arquivo
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	
	// Expand ~ em paths
	cfg.MCP.DataOrchestrator.WorkingDir = expandHome(cfg.MCP.DataOrchestrator.WorkingDir)
	cfg.Chat.SessionDir = expandHome(cfg.Chat.SessionDir)
	
	// Aplicar defaults
	cfg.applyDefaults()
	
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	
	return &cfg, nil
}

// applyDefaults aplica valores padrão
func (c *Config) applyDefaults() {
	if c.MCP.DataOrchestrator.PythonPath == "" {
		c.MCP.DataOrchestrator.PythonPath = "python3"
	}
	if c.MCP.DataOrchestrator.Timeout == 0 {
		c.MCP.DataOrchestrator.Timeout = 30
	}
	if c.Chat.HistorySize == 0 {
		c.Chat.HistorySize = 50
	}
	if c.Chat.SessionDir == "" {
		home, _ := os.UserHomeDir()
		c.Chat.SessionDir = filepath.Join(home, ".cristal", "sessions")
	}
	if c.UI.Prompt == "" {
		c.UI.Prompt = "🔮 > "
	}
}

// Validate valida configuração
func (c *Config) Validate() error {
	// Verificar WorkingDir existe
	if _, err := os.Stat(c.MCP.DataOrchestrator.WorkingDir); err != nil {
		return fmt.Errorf("data_orchestrator.working_dir: %w", err)
	}
	
	// Verificar HistorySize razoável
	if c.Chat.HistorySize < 10 || c.Chat.HistorySize > 1000 {
		return fmt.Errorf("chat.history_size deve estar entre 10 e 1000")
	}
	
	return nil
}

func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
```

### 10.3 Config Template

**Arquivo**: `config.example.yaml`

```yaml
mcp:
  data_orchestrator:
    python_path: "python3"
    script_path: "src.server"
    working_dir: "../data-orchestrator-mcp"
    timeout: 30

chat:
  history_size: 50
  save_history: true
  session_dir: "~/.cristal/sessions"

ui:
  color: true
  show_timestamps: false
  prompt: "🔮 > "
```

### 10.4 Atualizar Main

**Arquivo**: `cmd/cristal/main.go`

```go
func run() int {
	// Flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	versionFlag := flag.Bool("version", false, "print version")
	debugFlag := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()
	
	if *versionFlag {
		fmt.Printf("cristal v%s\n", version)
		return 0
	}
	
	// Logger
	logLevel := slog.LevelInfo
	if *debugFlag {
		logLevel = slog.LevelDebug
	}
	
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
	
	// Carregar config
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("falha ao carregar config", "path", *configPath, "error", err)
		return 1
	}
	
	logger.Info("config carregada", "path", *configPath)
	
	// MCP Client
	mcpCfg := mcp.Config{
		PythonPath: cfg.MCP.DataOrchestrator.PythonPath,
		ScriptPath: cfg.MCP.DataOrchestrator.ScriptPath,
		WorkingDir: cfg.MCP.DataOrchestrator.WorkingDir,
		Timeout:    time.Duration(cfg.MCP.DataOrchestrator.Timeout) * time.Second,
		Logger:     logger,
	}
	
	logger.Info("conectando ao data-orchestrator-mcp")
	client, err := mcp.NewClient(mcpCfg)
	if err != nil {
		logger.Error("falha ao criar cliente MCP", "error", err)
		return 1
	}
	defer client.Close()
	
	if err := client.Initialize(); err != nil {
		logger.Error("falha ao inicializar MCP", "error", err)
		return 1
	}
	
	logger.Info("MCP inicializado com sucesso")
	
	// REPL
	repl := ui.NewREPL(
		client, 
		cfg.Chat.HistorySize,
		cfg.Chat.SessionDir,
		logger,
	)
	
	// Context com signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), 
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	
	if err := repl.Run(ctx); err != nil {
		if err != context.Canceled {
			logger.Error("erro no REPL", "error", err)
			return 1
		}
	}
	
	return 0
}
```

### 10.5 Critérios de Aceite

- ✅ CA-4.1: `config.yaml` é carregado ao iniciar
- ✅ CA-4.2: Paths com `~` são expandidos corretamente
- ✅ CA-4.3: Defaults são aplicados para campos omitidos
- ✅ CA-4.4: Validação detecta `working_dir` inexistente
- ✅ CA-4.5: Flag `--config` permite especificar config alternativo

---

## 11. Milestone 4.2: Commands Enhancement

### 11.1 Novos Comandos

```go
// internal/ui/repl.go

case "/status":
	return r.cmdStatus(args)
case "/config":
	return r.cmdConfig(args)
case "/version", "/v":
	return r.cmdVersion(args)

func (r *REPL) cmdStatus(args []string) error {
	fmt.Println("\n📡 Status da Conexão MCP\n")
	
	// Testar conexão listando tools
	tools, err := r.client.ListTools()
	if err != nil {
		fmt.Printf("  Status: %s\n", r.formatter.error_.Sprint("❌ Desconectado"))
		return err
	}
	
	fmt.Printf("  Status: %s\n", r.formatter.success.Sprint("✅ Conectado"))
	fmt.Printf("  Tools disponíveis: %d\n", len(tools))
	fmt.Printf("  Sessão: %s\n", r.session.ID[:8])
	fmt.Printf("  Queries: %d\n", r.session.QueryCount)
	fmt.Println()
	
	return nil
}

func (r *REPL) cmdConfig(args []string) error {
	// TODO: mostrar config atual carregada
	fmt.Println("\n⚙️  Configuração Atual\n")
	fmt.Println("  [Implementar exibição de config]")
	fmt.Println()
	return nil
}

func (r *REPL) cmdVersion(args []string) error {
	fmt.Printf("cristal v%s\n", version)
	return nil
}
```

### 11.2 Help Melhorado

```go
func (r *REPL) cmdHelp(args []string) error {
	// Se tem argumento, mostra help de comando específico
	if len(args) > 0 {
		return r.cmdHelpSpecific(args[0])
	}
	
	help := `
Cristal Chat - Comandos Disponíveis

CONSULTAS:
  Digite qualquer pergunta para buscar no portal de transparência.
  
  Exemplos:
    quanto foi gasto com diárias em 2026
    contratos de licitação de 2025
    balancetes de março

COMANDOS GERAIS:
  /help, /h              Mostra esta ajuda
  /help <cmd>            Ajuda de comando específico
  /quit, /exit, /q       Sai do chat
  /status                Status da conexão MCP
  /version, /v           Versão do cristal

FERRAMENTAS MCP:
  /tools, /t             Lista tools do MCP disponíveis
  /metrics, /m           Métricas do sistema
  /cache <query>         Verifica se query está em cache

HISTÓRICO E SESSÃO:
  /history [n]           Mostra últimas n mensagens (padrão: 10)
  /session, /s           Resumo da sessão atual
  /clear                 Limpa histórico (mantém sessão)
  /save <arquivo>        Salva histórico em arquivo
  /load <arquivo>        Carrega histórico de arquivo

ATALHOS:
  Ctrl+C                 Sai do chat
  Ctrl+D                 Sai do chat (EOF)
  
Digite /help <comando> para detalhes.
`
	fmt.Println(help)
	return nil
}

func (r *REPL) cmdHelpSpecific(cmd string) error {
	helps := map[string]string{
		"history": `
/history [n]

Mostra as últimas n mensagens do histórico.
Se n não for especificado, mostra últimas 10.

Exemplos:
  /history       → últimas 10
  /history 20    → últimas 20
  /history 5     → últimas 5
`,
		"cache": `
/cache <query>

Verifica se uma query está em cache.
Útil para saber se uma pesquisa será instantânea.

Exemplo:
  /cache diárias 2026
`,
		// ... adicionar mais
	}
	
	help, ok := helps[strings.TrimPrefix(cmd, "/")]
	if !ok {
		return fmt.Errorf("comando '%s' não encontrado (use /help)", cmd)
	}
	
	fmt.Println(help)
	return nil
}
```

### 11.3 Critérios de Aceite

- ✅ CA-4.6: `/status` mostra conexão, tools, sessão, queries
- ✅ CA-4.7: `/version` mostra versão do binário
- ✅ CA-4.8: `/help <cmd>` mostra ajuda detalhada do comando
- ✅ CA-4.9: `/help` sem argumentos mostra resumo de todos os comandos

---

## 12. Milestone 4.3: Error Handling & Recovery

### 12.1 Retry Logic

**Atualizar**: `internal/mcp/client.go`

```go
// CallToolWithRetry executa tool com retry automático
func (c *Client) CallToolWithRetry(name string, args map[string]interface{}, maxRetries int) (*CallToolResult, error) {
	var lastErr error
	
	for attempt := 0; attempt < maxRetries; attempt++ {
		result, err := c.CallTool(name, args)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		
		// Se erro de conexão, tenta reconectar
		if isConnectionError(err) {
			c.logger.Warn("connection error, reconnecting", 
				"attempt", attempt+1, "error", err)
			
			if err := c.reconnect(); err != nil {
				c.logger.Error("reconnect failed", "error", err)
				continue
			}
		}
		
		// Backoff exponencial
		if attempt < maxRetries-1 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			c.logger.Info("retrying", "attempt", attempt+1, "backoff", backoff)
			time.Sleep(backoff)
		}
	}
	
	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}
	
	str := err.Error()
	return strings.Contains(str, "connection") ||
		strings.Contains(str, "EOF") ||
		strings.Contains(str, "broken pipe")
}

func (c *Client) reconnect() error {
	// Fechar conexão antiga
	c.cancel()
	if err := c.cmd.Wait(); err != nil {
		// Ignora erro de wait, esperado
	}
	
	// Criar nova conexão
	// TODO: implementar recriação completa
	return fmt.Errorf("reconnect not implemented yet")
}
```

### 12.2 Error Messages User-Friendly

```go
// internal/ui/formatter.go

func (f *Formatter) FormatError(err error) string {
	// Mapear erros técnicos para mensagens amigáveis
	msg := err.Error()
	
	switch {
	case strings.Contains(msg, "connection closed"):
		return "❌ Conexão com data-orchestrator perdida. Tente reiniciar o cristal."
	
	case strings.Contains(msg, "timeout"):
		return "⏱️  Timeout: a consulta demorou muito. Tente uma busca mais específica."
	
	case strings.Contains(msg, "not in cache"):
		return "💭 Não encontrado no cache. Use a busca normal."
	
	case strings.Contains(msg, "tool error"):
		return fmt.Sprintf("🔧 Erro na ferramenta: %s", msg)
	
	default:
		return fmt.Sprintf("❌ Erro: %s", msg)
	}
}
```

### 12.3 Graceful Shutdown

**Atualizar**: `internal/ui/repl.go`

```go
func (r *REPL) Run(ctx context.Context) error {
	r.running = true
	defer func() { 
		r.running = false
		r.cleanup()
	}()
	
	r.printWelcome()
	
	// Goroutine para detectar Ctrl+C
	go func() {
		<-ctx.Done()
		fmt.Println("\n\n⚠️  Interrompido. Salvando sessão...")
		r.running = false
	}()
	
	for r.running {
		input, err := r.readInput()
		if err != nil {
			if err == io.EOF {
				break
			}
			r.logger.Error("read input", "error", err)
			continue
		}
		
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		
		if err := r.handleInput(input); err != nil {
			fmt.Println(r.formatter.FormatError(err))
		}
	}
	
	r.printGoodbye()
	return nil
}

func (r *REPL) cleanup() {
	// Garantir que sessão seja salva
	if r.session != nil && r.sessionDir != "" {
		r.session.End()
		
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		saveDir := filepath.Join(r.sessionDir, timestamp)
		
		if err := r.session.Save(saveDir); err != nil {
			r.logger.Warn("falha ao salvar sessão na limpeza", "error", err)
		}
	}
}
```

### 12.4 Critérios de Aceite

- ✅ CA-4.10: Erros de conexão tentam reconectar automaticamente
- ✅ CA-4.11: Retry com backoff exponencial (1s, 2s, 4s)
- ✅ CA-4.12: Mensagens de erro são user-friendly
- ✅ CA-4.13: Ctrl+C salva sessão antes de sair
- ✅ CA-4.14: Timeout em queries longas não trava o chat

---

## 13. Build e Deploy

### 13.1 Go Modules

```bash
go mod init github.com/bergmaia/cristal-chat
go mod tidy
```

### 13.2 Makefile

```makefile
.PHONY: build test run clean install

VERSION ?= 0.1.0-dev
LDFLAGS = -s -w -X main.version=$(VERSION)

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/cristal ./cmd/cristal

test:
	go test -v ./...

test-integration:
	go test -v ./... -tags=integration

run: build
	./bin/cristal --config config.yaml

clean:
	rm -rf bin/
	go clean

install: build
	cp bin/cristal /usr/local/bin/

fmt:
	go fmt ./...

lint:
	golangci-lint run
```

### 13.3 .gitignore

```
# Binaries
bin/
cristal

# Config
config.yaml

# Sessions
.cristal/

# Go
*.o
*.a
*.so

# Test
*.test
*.out
coverage.*

# IDE
.vscode/
.idea/
*.swp
```

---

## 14. Testes

### 14.1 Testes Unitários

```bash
# Rodar todos os testes (sem integração)
go test ./... -short

# Com coverage
go test -cover ./internal/...
```

### 14.2 Testes de Integração

Requerem data-orchestrator-mcp rodando:

```bash
# Tag integration
go test ./... -tags=integration -v

# Ou explícito
go test ./internal/mcp/... -run TestClientInitialize
```

### 14.3 Smoke Tests

Checklist manual após cada milestone:

**Sprint 1:**
- [ ] Conecta ao data-orchestrator sem erros
- [ ] `/tools` lista 4 tools
- [ ] Query simples "teste" retorna resposta
- [ ] `/quit` encerra gracefully

**Sprint 2:**
- [ ] Query retorna páginas e documentos formatados
- [ ] Valores monetários aparecem em verde
- [ ] `/metrics` exibe tabela estruturada

**Sprint 3:**
- [ ] `/history` mostra mensagens anteriores
- [ ] `/save` e `/load` funcionam
- [ ] Sessão é auto-salva ao sair

**Sprint 4:**
- [ ] `config.yaml` é lido corretamente
- [ ] Erro de conexão mostra mensagem amigável
- [ ] Ctrl+C salva sessão antes de sair

---

## 15. Cronograma Estimado

| Sprint | Duração | Entregável |
|--------|---------|------------|
| **Sprint 1** | 2-3 dias | MCP Client + REPL básico |
| **Sprint 2** | 2-3 dias | Orchestrator + Formatação |
| **Sprint 3** | 2-3 dias | História + Sessão |
| **Sprint 4** | 1-2 dias | Config + Polish |
| **Total** | **7-11 dias** | **MVP completo** |

---

## 16. Dependências

```go
// go.mod
module github.com/bergmaia/cristal-chat

go 1.22

require (
	github.com/fatih/color v1.16.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/google/uuid v1.6.0
	gopkg.in/yaml.v3 v3.0.1
)
```

---

## 17. Próximos Passos Pós-MVP

### Fase 2 (Opcional):
- TUI com Bubble Tea (interface mais rica)
- Agentes LLM (planning, analysis)
- Semantic caching
- Multi-turn com contexto

### Fase 3 (Opcional):
- Export de relatórios (Markdown, PDF)
- Gráficos ASCII
- Atalhos e aliases customizáveis
- Suporte a múltiplos MCPs simultâneos

---

## 18. Anexos

### 18.1 Exemplo de Fluxo Completo

```bash
$ cristal --config config.yaml

🔮 Cristal Chat v0.1.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Conectando ao data-orchestrator-mcp...
✓ Conectado! 4 tools disponíveis.

Digite /help para ajuda ou faça sua pergunta.

🔮 > quanto foi gasto com diárias em 2026

🔍 Pesquisando...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Resultados da Pesquisa
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📝 Encontrados dados de diárias no primeiro trimestre de 2026

💰 Total encontrado: R$ 152.342,87

📄 Páginas encontradas (3):
  1. Relatório de Despesas - Q1/2026
     https://tre-pi.jus.br/.../relatorio-q1
     » Relatório consolidado de despesas do primeiro trimestre
     
  2. Balancete - Março 2026
     https://tre-pi.jus.br/.../balancete-mar
     » Balancete mensal com detalhamento por rubrica
     
  3. Prestação de Contas - 2026
     https://tre-pi.jus.br/.../prestacao-contas
     » Prestação de contas anual com todos os gastos

📎 Documentos analisados (2):
  • relatorio-despesas-q1-2026.pdf (845 KB)
    12 páginas
  • balancete-2026-03.pdf (1.2 MB)
    8 páginas

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔮 > /history

📜 Histórico (últimas 2 mensagens):

  [14:23] user: quanto foi gasto com diárias em 2026
  [14:23] assistant: Encontrados dados de diárias no primeiro trimestre de 2026

🔮 > /metrics

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📈 Métricas do Sistema
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Uptime: 2h 15m
  Cache hit rate: 73.5%
  Queries processadas: 42
  Documentos extraídos: 18
  Erros de extração: 2
  Bytes transferidos: 45.2 MB
  Total de erros: 3

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔮 > /quit

👋 Encerrando sessão...

Sessão: a3f7b2c8
Duração: 3m 45s
Queries: 1
Mensagens: 2

Sessão salva em ~/.cristal/sessions/2026-04-21_14-27

Até logo!
```

---

## 19. Referências

- MCP Protocol: https://modelcontextprotocol.io/
- Data Orchestrator MCP: `../data-orchestrator-mcp/README.md`
- Site Research MCP: `../cmd/site-research-mcp/README.md`
- Go Best Practices: https://go.dev/doc/effective_go

---

**Fim do Plano de Implementação**

Versão: 1.0  
Data: 2026-04-21  
Status: Pronto para desenvolvimento
