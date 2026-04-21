// Package config loads and validates the YAML configuration file.
package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure.
type Config struct {
	Scope   ScopeConfig   `yaml:"scope"`
	Sitemap SitemapConfig `yaml:"sitemap"`
	Crawler CrawlerConfig `yaml:"crawler"`
	Storage StorageConfig `yaml:"storage"`
	LLM     LLMConfig     `yaml:"llm"`
	Recrawl RecrawlConfig `yaml:"recrawl"`
	Logging LoggingConfig `yaml:"logging"`
}

// ScopeConfig defines the crawl scope.
type ScopeConfig struct {
	SeedURL string `yaml:"seed_url"`
	Prefix  string `yaml:"prefix"`
}

// SitemapConfig holds the sitemap URL.
type SitemapConfig struct {
	URL string `yaml:"url"`
}

// CrawlerConfig controls HTTP crawl behavior.
type CrawlerConfig struct {
	UserAgent             string                   `yaml:"user_agent"`
	RateLimitPerSecond    float64                  `yaml:"rate_limit_per_second"`
	JitterMS              int                      `yaml:"jitter_ms"`
	RequestTimeoutSeconds int                      `yaml:"request_timeout_seconds"`
	MaxRetries            int                      `yaml:"max_retries"`
	RespectRobotsTxt      bool                     `yaml:"respect_robots_txt"`
	HonorRetryAfter       bool                     `yaml:"honor_retry_after"`
	CircuitBreaker        CircuitBreakerConfig     `yaml:"circuit_breaker"`
	SuspiciousResponse    SuspiciousResponseConfig `yaml:"suspicious_response"`
}

// CircuitBreakerConfig configures the circuit breaker.
type CircuitBreakerConfig struct {
	MaxConsecutiveFailures int `yaml:"max_consecutive_failures"`
	PauseMinutes           int `yaml:"pause_minutes"`
	AbortThreshold         int `yaml:"abort_threshold"`
}

// SuspiciousResponseConfig defines criteria for detecting blocked responses.
type SuspiciousResponseConfig struct {
	MinBodyBytes       int      `yaml:"min_body_bytes"`
	BlockTitlePatterns []string `yaml:"block_title_patterns"`
}

// StorageConfig specifies filesystem paths for stored data.
type StorageConfig struct {
	DataDir     string `yaml:"data_dir"`
	CatalogPath string `yaml:"catalog_path"`
	SQLitePath  string `yaml:"sqlite_path"`
}

// LLMConfig holds LLM provider settings.
type LLMConfig struct {
	Provider              string `yaml:"provider"`
	Model                 string `yaml:"model"`
	Endpoint              string `yaml:"endpoint"`
	APIKeyEnv             string `yaml:"api_key_env"`
	Concurrency           int    `yaml:"concurrency"`
	RequestTimeoutSeconds int    `yaml:"request_timeout_seconds"`
}

// RecrawlConfig controls recrawl behavior.
type RecrawlConfig struct {
	StaleRetentionDays int  `yaml:"stale_retention_days"`
	ForceResummarize   bool `yaml:"force_resummarize"`
}

// LoggingConfig configures the logger.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load reads the YAML file at path, applies defaults, validates and returns a Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Crawler.UserAgent == "" {
		c.Crawler.UserAgent = "TRE-PI-Research-Crawler/0.1 (+contact: cotdi@tre-pi.jus.br)"
	}
	if c.Crawler.RateLimitPerSecond == 0 {
		c.Crawler.RateLimitPerSecond = 1.0
	}
	if c.Crawler.JitterMS == 0 {
		c.Crawler.JitterMS = 200
	}
	if c.Crawler.RequestTimeoutSeconds == 0 {
		c.Crawler.RequestTimeoutSeconds = 30
	}
	if c.Crawler.MaxRetries == 0 {
		c.Crawler.MaxRetries = 3
	}
	if c.Crawler.CircuitBreaker.MaxConsecutiveFailures == 0 {
		c.Crawler.CircuitBreaker.MaxConsecutiveFailures = 5
	}
	if c.Crawler.CircuitBreaker.PauseMinutes == 0 {
		c.Crawler.CircuitBreaker.PauseMinutes = 10
	}
	if c.Crawler.CircuitBreaker.AbortThreshold == 0 {
		c.Crawler.CircuitBreaker.AbortThreshold = 3
	}
	if c.Crawler.SuspiciousResponse.MinBodyBytes == 0 {
		c.Crawler.SuspiciousResponse.MinBodyBytes = 500
	}
	if c.Crawler.SuspiciousResponse.BlockTitlePatterns == nil {
		c.Crawler.SuspiciousResponse.BlockTitlePatterns = []string{
			"Access Denied",
			"Forbidden",
			"Captcha",
			"Cloudflare",
			"Just a moment",
		}
	}
	if c.Storage.DataDir == "" {
		c.Storage.DataDir = "./data"
	}
	if c.Storage.CatalogPath == "" {
		c.Storage.CatalogPath = "./data/catalog.json"
	}
	if c.Storage.SQLitePath == "" {
		c.Storage.SQLitePath = "./data/catalog.sqlite"
	}
	if c.LLM.Provider == "" {
		c.LLM.Provider = "anthropic"
	}
	if c.LLM.Model == "" {
		c.LLM.Model = "claude-haiku-4-5"
	}
	if c.LLM.Endpoint == "" {
		c.LLM.Endpoint = "https://api.anthropic.com"
	}
	if c.LLM.APIKeyEnv == "" {
		c.LLM.APIKeyEnv = "ANTHROPIC_API_KEY"
	}
	if c.LLM.Concurrency == 0 {
		c.LLM.Concurrency = 3
	}
	if c.LLM.RequestTimeoutSeconds == 0 {
		c.LLM.RequestTimeoutSeconds = 60
	}
	if c.Recrawl.StaleRetentionDays == 0 {
		c.Recrawl.StaleRetentionDays = 30
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	// TODO(M3): RespectRobotsTxt and HonorRetryAfter default to false (Go zero value).
	// BRIEF RF-07 requires them to default to true. Proper pointer-or-sentinel handling
	// is needed so that explicit false in YAML is honoured while the zero value defaults to true.
}

func (c *Config) validate() error {
	if c.Scope.SeedURL == "" {
		return errors.New("config: scope.seed_url is required")
	}
	if c.Scope.Prefix == "" {
		return errors.New("config: scope.prefix is required")
	}
	return nil
}
