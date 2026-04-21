// Binary site-research-mcp is the MCP server for the site-research project.
// It exposes the crawled catalog via the Model Context Protocol over stdio.
// All configuration is via environment variables (see README).
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	_ "modernc.org/sqlite"

	"github.com/bergmaia/site-research/internal/config"
	"github.com/bergmaia/site-research/internal/domain"
	"github.com/bergmaia/site-research/internal/logging"
	"github.com/bergmaia/site-research/internal/mcp"
	"github.com/bergmaia/site-research/internal/tools"
)

// version is set at build time via -ldflags "-X main.version=<semver>".
var version = "dev"

func main() {
	os.Exit(run())
}

func run() int {
	// --- 1. Read SITE_RESEARCH_DATA_DIR (required) --------------------------

	dataDir := os.Getenv("SITE_RESEARCH_DATA_DIR")
	if dataDir == "" {
		fmt.Fprintf(os.Stderr, "site-research-mcp: missing required environment variables: SITE_RESEARCH_DATA_DIR\n")
		return 2
	}

	logLevel := os.Getenv("SITE_RESEARCH_LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// ANTHROPIC_API_KEY is read here for completeness; it is only used in M2.
	_ = os.Getenv("ANTHROPIC_API_KEY")

	// --- 2. Initialise logger -----------------------------------------------

	logger := logging.New(logging.Config{
		Level:  logLevel,
		Format: "json",
		Output: os.Stderr,
	})

	logger.Info("starting site-research-mcp", "version", version)

	// --- 3. Verify DATA_DIR exists and is a directory -----------------------

	if fi, err := os.Stat(dataDir); err != nil {
		logger.Error("data dir not accessible", "path", dataDir, "err", err)
		return 1
	} else if !fi.IsDir() {
		logger.Error("data dir is not a directory", "path", dataDir)
		return 1
	}

	// --- 4. Compute defaults for catalog + fts paths; override with env -----

	sourceCatalog := "default"
	catalogPath := os.Getenv("SITE_RESEARCH_CATALOG")
	if catalogPath == "" {
		catalogPath = filepath.Join(dataDir, "catalog.json")
	} else {
		sourceCatalog = "env"
	}

	sourceFTS := "default"
	ftsDBPath := os.Getenv("SITE_RESEARCH_FTS_DB")
	if ftsDBPath == "" {
		ftsDBPath = filepath.Join(dataDir, "catalog.sqlite")
	} else {
		sourceFTS = "env"
	}

	// --- 5. Startup validation: file existence + FTS sanity -----------------

	// 5a. catalog.json must exist.
	if _, err := os.Stat(catalogPath); err != nil {
		logger.Error("catalog file not accessible", "path", catalogPath, "err", err)
		return 1
	}

	// 5b. FTS database must exist.
	if _, err := os.Stat(ftsDBPath); err != nil {
		logger.Error("FTS database not accessible", "path", ftsDBPath, "err", err)
		return 1
	}

	// 5c. Open SQLite read-only and verify pages_fts table exists.
	dsn := fmt.Sprintf("file:%s?mode=ro", ftsDBPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		logger.Error("open FTS database", "path", ftsDBPath, "err", err)
		return 1
	}
	var pageCount int
	if err := db.QueryRow("SELECT count(*) FROM pages_fts").Scan(&pageCount); err != nil {
		_ = db.Close()
		logger.Error("FTS sanity check failed", "path", ftsDBPath, "err", err)
		return 1
	}
	_ = db.Close()
	logger.Info("FTS database ok", "path", ftsDBPath, "pages_fts_count", pageCount)

	// 5d. Read and validate catalog.json.
	catalogData, err := os.ReadFile(catalogPath)
	if err != nil {
		logger.Error("read catalog.json", "path", catalogPath, "err", err)
		return 1
	}
	var cat domain.Catalog
	if err := json.Unmarshal(catalogData, &cat); err != nil {
		logger.Error("parse catalog.json", "path", catalogPath, "err", err)
		return 1
	}
	if cat.SchemaVersion != 2 {
		logger.Error("unexpected catalog schema version", "path", catalogPath, "schema_version", cat.SchemaVersion, "expected", 2)
		return 1
	}
	logger.Info("catalog ok", "path", catalogPath, "schema_version", cat.SchemaVersion, "total_pages", cat.Stats.TotalPages)

	// --- 6. Resolve scope prefix (env override or catalog.RootURL) ----------

	sourceScope := "catalog.root_url"
	scopePrefix := os.Getenv("SITE_RESEARCH_SCOPE_PREFIX")
	if scopePrefix != "" {
		sourceScope = "env"
	} else {
		scopePrefix = cat.RootURL
		if scopePrefix == "" {
			logger.Error("scope prefix not set: SITE_RESEARCH_SCOPE_PREFIX is unset and catalog.root_url is empty",
				"path", catalogPath)
			return 1
		}
	}

	// --- 7. Log resolved config ---------------------------------------------

	logger.Info("resolved config",
		"data_dir", dataDir,
		"catalog_path", catalogPath,
		"fts_db_path", ftsDBPath,
		"scope_prefix", scopePrefix,
		"source_catalog", sourceCatalog,
		"source_fts", sourceFTS,
		"source_scope", sourceScope,
	)

	// --- 8. Build Config, registry, server ----------------------------------

	cfg := &config.Config{
		Scope: config.ScopeConfig{
			Prefix: scopePrefix,
		},
		Storage: config.StorageConfig{
			CatalogPath: catalogPath,
			SQLitePath:  ftsDBPath,
			DataDir:     dataDir,
		},
	}

	registry := tools.DefaultRegistry(cfg, logger)
	server := mcp.NewServer(logger, registry, os.Stdin, os.Stdout, version)

	// --- 9. Signal handling -------------------------------------------------

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// --- 10. Run server -----------------------------------------------------

	if err := server.Run(ctx); err != nil {
		logger.Error("server error", "err", err)
		return 1
	}

	logger.Info("site-research-mcp exiting cleanly")
	return 0
}
