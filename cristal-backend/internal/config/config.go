package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Supported LLM providers.
const (
	ProviderAnthropic = "anthropic"
	ProviderVertex    = "vertex"
)

// Config is the top-level configuration structure
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	MCP       MCPConfig       `yaml:"mcp"`
	LLM       LLMConfig       `yaml:"llm"`
	Anthropic AnthropicConfig `yaml:"anthropic"`
	Vertex    VertexConfig    `yaml:"vertex"`
}

// ServerConfig defines HTTP server settings
type ServerConfig struct {
	Port int `yaml:"port"`
}

// MCPConfig holds settings for both MCP servers
type MCPConfig struct {
	DataOrchestrator MCPServerConfig `yaml:"data_orchestrator"`
	SiteResearch     MCPServerConfig `yaml:"site_research"`
}

// MCPServerConfig defines settings for a single MCP server
type MCPServerConfig struct {
	Command    string            `yaml:"command"`
	Args       []string          `yaml:"args"`
	WorkingDir string            `yaml:"working_dir"`
	Env        map[string]string `yaml:"env"`
	Timeout    time.Duration     `yaml:"timeout"`
}

// LLMConfig holds provider-neutral generation settings plus provider selection.
type LLMConfig struct {
	Provider    string        `yaml:"provider"` // "anthropic" | "vertex"
	MaxTokens   int           `yaml:"max_tokens"`
	Temperature float64       `yaml:"temperature"`
	Timeout     time.Duration `yaml:"timeout"`
}

// AnthropicConfig holds Anthropic Messages API settings.
type AnthropicConfig struct {
	Model    string `yaml:"model"`
	Endpoint string `yaml:"endpoint"` // optional override
}

// VertexConfig holds Vertex AI / Gemini settings.
type VertexConfig struct {
	ProjectID       string `yaml:"project_id"`
	Location        string `yaml:"location"` // e.g. "global", "us-central1"
	Model           string `yaml:"model"`    // e.g. "gemini-3-flash-preview"
	CredentialsFile string `yaml:"credentials_file"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}

	// Backwards compatibility: if llm.provider is empty but the anthropic
	// section is populated, default to anthropic.
	if cfg.LLM.Provider == "" {
		if cfg.Anthropic.Model != "" {
			cfg.LLM.Provider = ProviderAnthropic
		} else if cfg.Vertex.Model != "" {
			cfg.LLM.Provider = ProviderVertex
		}
	}
	cfg.LLM.Provider = strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))

	if cfg.LLM.MaxTokens == 0 {
		cfg.LLM.MaxTokens = 4096
	}
	if cfg.LLM.Temperature == 0 {
		cfg.LLM.Temperature = 0.7
	}
	if cfg.LLM.Timeout == 0 {
		cfg.LLM.Timeout = 60 * time.Second
	}

	if cfg.Vertex.Location == "" {
		cfg.Vertex.Location = "global"
	}

	if cfg.MCP.DataOrchestrator.Timeout == 0 {
		cfg.MCP.DataOrchestrator.Timeout = 120 * time.Second
	}
	if cfg.MCP.SiteResearch.Timeout == 0 {
		cfg.MCP.SiteResearch.Timeout = 30 * time.Second
	}
}

func validate(cfg *Config) error {
	switch cfg.LLM.Provider {
	case ProviderAnthropic:
		if cfg.Anthropic.Model == "" {
			return fmt.Errorf("anthropic.model is required when llm.provider=anthropic")
		}
	case ProviderVertex:
		if cfg.Vertex.ProjectID == "" {
			return fmt.Errorf("vertex.project_id is required when llm.provider=vertex")
		}
		if cfg.Vertex.Model == "" {
			return fmt.Errorf("vertex.model is required when llm.provider=vertex")
		}
	case "":
		return fmt.Errorf("llm.provider is required (anthropic|vertex)")
	default:
		return fmt.Errorf("unknown llm.provider %q (expected anthropic|vertex)", cfg.LLM.Provider)
	}

	if cfg.MCP.DataOrchestrator.Command == "" {
		return fmt.Errorf("mcp.data_orchestrator.command is required")
	}
	if cfg.MCP.SiteResearch.Command == "" {
		return fmt.Errorf("mcp.site_research.command is required")
	}
	return nil
}
