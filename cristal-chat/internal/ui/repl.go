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
