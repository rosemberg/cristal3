# site-research-mcp

Servidor MCP (Model Context Protocol) sobre stdio para o projeto `site-research`.
Expõe o catálogo gerado pela Fase 1 (crawler) como três ferramentas read-only —
`search`, `inspect_page` e `catalog_stats` — para uso direto em clientes como
Claude Desktop, Claude Code e Cowork.

---

## Visao Geral

`site-research-mcp` e uma fachada fina sobre o catálogo produzido pela Fase 1
do projeto `site-research`. Ele não implementa lógica de busca própria: delega
ao banco FTS5 SQLite e ao fsstore de `_index.json` gerados pelo crawler.

- Transporte: stdio com JSON-RPC 2.0 delimitado por newline.
- Protocolo MCP: revisão `2025-11-25` (única; sem fallback).
- Capacidades: apenas `tools`. Resources, prompts, sampling e roots nao sao
  anunciados.
- Binário estático, sem dependência de biblioteca dinâmica (driver SQLite
  CGO-free via `modernc.org/sqlite`).
- Compatível com Claude Desktop, Claude Code e qualquer cliente MCP que implemente
  a revisão `2025-11-25`.

---

## Pre-requisitos

1. **Catalogo gerado pela Fase 1.** Os arquivos abaixo devem existir antes de
   iniciar o servidor:
   - `catalog.json` — índice consolidado.
   - `catalog.sqlite` — banco FTS5 para buscas.
   - Árvore de `_index.json` — metadados por página.

   Caso o catálogo nao exista, execute o pipeline completo da Fase 1:

   ```
   site-research discover
   site-research crawl
   site-research summarize
   site-research build-catalog
   ```

2. **Go 1.25+** — somente para build local; para usar o binário pré-compilado
   nao e necessário.

3. **ANTHROPIC_API_KEY** — necessária somente se alguma tool invocar a engine
   da Fase 2 em modo que exija LLM em runtime. Na versao atual (v0.x), a engine
   nao usa LLM em consultas; a variável e lida mas ignorada.

---

## Instalacao

### Binário pré-compilado (GitHub Release)

```bash
# Substitua VERSION e PLATAFORMA conforme necessário
# Plataformas disponíveis: darwin_amd64, darwin_arm64, linux_amd64,
#                          linux_arm64, windows_amd64

VERSION=0.1.0
PLATFORM=darwin_arm64

curl -L \
  "https://github.com/bergmaia/site-research/releases/download/v${VERSION}/site-research-mcp_${VERSION}_${PLATFORM}.tar.gz" \
  | tar xz site-research-mcp

chmod +x site-research-mcp
sudo mv site-research-mcp /usr/local/bin/
```

**macOS — binário nao assinado:**

Se o macOS exibir "can't be opened because Apple cannot check it for malicious
software", remova o atributo de quarentena:

```bash
xattr -d com.apple.quarantine /usr/local/bin/site-research-mcp
```

### Build local

Requer o repositório e Go 1.25+:

```bash
go build -o /usr/local/bin/site-research-mcp ./cmd/site-research-mcp
```

Para embutir a versão no binário (recomendado):

```bash
go build \
  -trimpath \
  -ldflags "-s -w -X main.version=0.1.0" \
  -o /usr/local/bin/site-research-mcp \
  ./cmd/site-research-mcp
```

---

## Variaveis de Ambiente

O servidor le exclusivamente variáveis de ambiente no startup. Nao há arquivo
de configuração.

| Variável | Obrigatória | Padrão | Descricao |
|---|---|---|---|
| `SITE_RESEARCH_DATA_DIR` | **sim** | — | Caminho absoluto para o diretório raiz do fsstore (árvore de `_index.json`). **Única variável obrigatória.** |
| `SITE_RESEARCH_CATALOG` | nao | `{DATA_DIR}/catalog.json` | Caminho para `catalog.json`. Definir somente se o arquivo estiver fora de `DATA_DIR`. |
| `SITE_RESEARCH_FTS_DB` | nao | `{DATA_DIR}/catalog.sqlite` | Caminho para `catalog.sqlite`. Definir somente se o arquivo estiver fora de `DATA_DIR`. |
| `SITE_RESEARCH_SCOPE_PREFIX` | nao | lido de `catalog.root_url` | Prefixo de escopo do catálogo (ex: `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas`). Usado por `inspect_page` para resolver paths relativos. Se não definido, o valor é lido do campo `root_url` dentro de `catalog.json`. |
| `SITE_RESEARCH_LOG_LEVEL` | nao | `info` | Nível de log: `debug`, `info`, `warn` ou `error`. |
| `ANTHROPIC_API_KEY` | condicional | — | Obrigatória apenas se a engine da Fase 2 exigir LLM em runtime. Na v0.x, ignorada. |

Ausência de `SITE_RESEARCH_DATA_DIR` resulta em **exit code 2** com mensagem
em stderr. Falhas de validação (arquivo inexistente, schema inválido, FTS
corrompido, `root_url` vazio sem `SCOPE_PREFIX` definido) resultam em
**exit code 1**.

---

## Tools Expostas

### `search`

Busca páginas no catálogo usando FTS5. Aceita consulta em linguagem natural,
número máximo de resultados (`limit`, padrão 10, máximo 50) e filtro de seção
opcional.

Detalhes do schema e exemplos de resposta: `MCP_BRIEF.md` §RF-02.

### `inspect_page`

Retorna metadados completos de uma única página: título, seção, breadcrumb,
mini-resumo, tipo, documentos anexos e páginas filhas. Aceita URL completa ou
path relativo ao escopo.

Detalhes: `MCP_BRIEF.md` §RF-03.

### `catalog_stats`

Retorna estatísticas agregadas do catálogo: total de páginas, distribuição por
profundidade e tipo, top seções e contagem de documentos. Sem argumentos.

Detalhes: `MCP_BRIEF.md` §RF-04.

---

## Configuracao em Clientes MCP

### Claude Desktop (macOS)

Arquivo de configuracao: `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "site-research": {
      "command": "/usr/local/bin/site-research-mcp",
      "env": {
        "SITE_RESEARCH_DATA_DIR": "/Users/SEU_USUARIO/site-research/data",
        "SITE_RESEARCH_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Claude Desktop (Linux)

Arquivo de configuracao: `~/.config/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "site-research": {
      "command": "/usr/local/bin/site-research-mcp",
      "env": {
        "SITE_RESEARCH_DATA_DIR": "/home/SEU_USUARIO/site-research/data",
        "SITE_RESEARCH_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Claude Desktop (Windows)

Arquivo de configuracao:
`%APPDATA%\Claude\claude_desktop_config.json`
(ex: `C:\Users\SEU_USUARIO\AppData\Roaming\Claude\claude_desktop_config.json`)

```json
{
  "mcpServers": {
    "site-research": {
      "command": "C:\\Users\\SEU_USUARIO\\bin\\site-research-mcp.exe",
      "env": {
        "SITE_RESEARCH_DATA_DIR": "C:\\Users\\SEU_USUARIO\\site-research\\data",
        "SITE_RESEARCH_LOG_LEVEL": "info"
      }
    }
  }
}
```

Após editar o arquivo, **reinicie o Claude Desktop** para que o servidor seja
carregado.

### Claude Code (CLI)

```bash
claude mcp add site-research \
  --env SITE_RESEARCH_DATA_DIR=/Users/SEU_USUARIO/site-research/data \
  -- /usr/local/bin/site-research-mcp
```

Para verificar se foi registrado: `claude mcp list`

Para remover: `claude mcp remove site-research`

### Configuracao com overrides explícitos

Caso os arquivos `catalog.json` e `catalog.sqlite` estejam em local diferente
do `DATA_DIR`, ou o `root_url` do catálogo precise ser substituído, use as
variáveis opcionais como overrides:

```json
{
  "mcpServers": {
    "site-research": {
      "command": "/usr/local/bin/site-research-mcp",
      "env": {
        "SITE_RESEARCH_DATA_DIR": "/dados/site-research/data",
        "SITE_RESEARCH_CATALOG": "/dados/site-research/catalog.json",
        "SITE_RESEARCH_FTS_DB": "/dados/site-research/catalog.sqlite",
        "SITE_RESEARCH_SCOPE_PREFIX": "https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas",
        "SITE_RESEARCH_LOG_LEVEL": "info"
      }
    }
  }
}
```

### Cowork

O Cowork segue o mesmo modelo de configuracao: comando absoluto para o binário
mais bloco de variáveis de ambiente. A forma canônica exata do arquivo de
configuracao do Cowork será documentada quando a integração for testada
end-to-end. Por ora, use a estrutura JSON de Claude Desktop como referência —
a semântica e identica (campo `command` + campo `env`).

---

## Protocolo MCP

- **Revisao suportada**: `2025-11-25` (única; sem fallback de versao).
- **Transporte**: stdio com JSON-RPC 2.0 delimitado por newline (`\n`).
- **Capacidades anunciadas**: apenas `tools: {}`. Resources, prompts, sampling
  e roots nao sao anunciados nem suportados.
- **Cancelamento**: via notificacao `notifications/cancelled` com `requestId`.
  O servidor cancela o contexto da tool em andamento; a resposta retorna
  `isError: true` com mensagem informando o cancelamento.
- **Clientes com revisao diferente** recebem erro explícito no `initialize` e
  devem atualizar para suportar `2025-11-25`.

---

## Desempenho

- **Cold start**: ~7 ms em Apple M4 Pro com binário pré-compilado (medido por
  `BenchmarkColdStart` com catálogo de 2 páginas de fixture em M3 do ciclo de
  desenvolvimento; catálogo real de 573 páginas teve cold start dentro do
  orçamento de 500 ms definido pelo CA-12).
- **Orçamento de cold start (CA-12)**: < 500 ms em macOS/Linux com catálogo de
  573 páginas.
- **Tamanho dos binários** (build com `-s -w -trimpath -X main.version=0.1.0`):

| Plataforma | Tamanho |
|---|---|
| darwin/arm64 | ver tabela em §Build e Release |
| darwin/amd64 | ver tabela em §Build e Release |
| linux/amd64 | ver tabela em §Build e Release |
| linux/arm64 | ver tabela em §Build e Release |
| windows/amd64 | ver tabela em §Build e Release |

Todos os binários devem ser < 25 MB comprimidos (RNF-04 / CA-1). Os valores
exatos sao preenchidos apos o primeiro `goreleaser build --snapshot --clean`
ou build manual descrito em §Build e Release.

---

## Logs

- Todos os logs vao para **stderr**, formato JSON (`log/slog`).
- **Stdout e reservado exclusivamente a JSON-RPC.** Qualquer byte fora de
  JSON-RPC válido em stdout indica bug grave e corrompe o protocolo.
- Campos mínimos por entrada de log: `time` (RFC 3339), `level`, `msg`.
- Campos adicionais conforme contexto: `tool`, `query`, `duration_ms`, `hits`,
  `request_id`, `error`.
- **API keys nao sao logadas** em nenhum nível.

---

## Troubleshooting

### "Catálogo nao encontrado" no startup

O servidor encerra com **exit code 1** e mensagem em stderr indicando o caminho
e a variável correspondente.

Por padrão, `catalog.json` e `catalog.sqlite` são procurados dentro do
`SITE_RESEARCH_DATA_DIR`. Verifique que o pipeline da Fase 1 foi executado na
ordem correta e que os arquivos existem no diretório configurado:

```bash
site-research discover
site-research crawl
site-research summarize
site-research build-catalog
```

Se os arquivos estiverem em outro local, use as variáveis `SITE_RESEARCH_CATALOG`
e `SITE_RESEARCH_FTS_DB` para apontá-los explicitamente.

### Variável de ambiente obrigatória ausente

Somente `SITE_RESEARCH_DATA_DIR` é obrigatória. O servidor encerra com
**exit code 2** se ela nao estiver definida:

```
site-research-mcp: missing required environment variables: SITE_RESEARCH_DATA_DIR
```

Adicione a variável ao bloco `env` do cliente MCP e reinicie.

### "unsupported protocol version ..." no initialize

O cliente MCP está usando uma revisao do protocolo diferente de `2025-11-25`.
Atualize o cliente para uma versao que suporte essa revisao, ou aguarde uma
versao futura do binário que suporte múltiplas revisoes.

### macOS: "can't be opened because Apple cannot check it for malicious software"

O binário baixado nao está assinado com certificado Apple. Execute:

```bash
xattr -d com.apple.quarantine /usr/local/bin/site-research-mcp
```

Se preferir, compile localmente via `go build ./cmd/site-research-mcp` — o
binário produzido localmente nao recebe o atributo de quarentena.

### Resposta de uma tool veio vazia ou com erro inesperado

1. Verifique que `SITE_RESEARCH_DATA_DIR` aponta para o diretório correto e
   que os arquivos `_index.json` existem dentro dele.
2. Verifique que `SITE_RESEARCH_FTS_DB` aponta para o arquivo SQLite correto e
   que ele nao está corrompido.
3. Para reconstruir o banco FTS a partir do catálogo existente:
   ```bash
   site-research build-catalog
   ```
4. Ative logs de depuracao (`SITE_RESEARCH_LOG_LEVEL=debug`) e inspecione
   stderr para identificar o erro específico.

### Cold start > 500 ms

Provavelmente o catálogo está em um filesystem remoto ou sincronizado (iCloud
Drive, Dropbox, NFS). Mova os arquivos de catálogo para uma pasta local (ex:
`~/site-research/data/`) e atualize as variáveis de ambiente.

### Tool `search` retorna zero resultados para consulta esperada

1. Verifique que o catálogo foi gerado com as páginas desejadas
   (`catalog_stats` mostra o total de páginas indexadas).
2. Tente reformular a consulta com termos mais específicos presentes no título
   ou no texto das páginas.
3. Se usar o filtro `section`, verifique que o nome da seção corresponde
   exatamente ao valor armazenado no catálogo (sensível a maiúsculas).

---

## Smoke Test Manual (checklist)

Execute estes passos após instalar o binário e antes de declarar a versao
pronta para uso em producao:

1. **Gere o catálogo** via Fase 1 e confirme que `catalog.json`,
   `catalog.sqlite` e a árvore de `_index.json` existem no `DATA_DIR`.

2. **Configure o cliente MCP** (ex: Claude Desktop) com o bloco JSON descrito
   em §Configuracao em Clientes MCP, substituindo os paths pelos reais.

3. **Reinicie o cliente** (Claude Desktop: feche e reabra; Claude Code: nao
   requer reinicialização).

4. **Liste as tools disponíveis**: envie ao Claude a mensagem
   "liste as tools disponíveis" ou "quais ferramentas você tem?". Verifique
   que as três tools aparecem: `search`, `inspect_page`, `catalog_stats`.

5. **Busca**: peça "busque sobre balancetes" ou "encontre páginas sobre
   diárias". Verifique que o Claude chama `search` e retorna URLs reais do
   catálogo.

6. **Inspecao de página**: peça "inspecione a página de contabilidade" ou
   forneça uma URL retornada pelo `search`. Verifique que o Claude chama
   `inspect_page` e retorna metadados estruturados.

7. **Estatísticas**: peça "quantas páginas tem o catálogo?" ou "mostre
   estatísticas do catálogo". Verifique que o Claude chama `catalog_stats` e
   retorna totais, distribuicao por tipo e top secoes.

8. **Erro esperado**: peça "inspecione a página xyz/nao/existe". Verifique que
   o Claude reporta erro claro indicando que a página nao foi encontrada no
   catálogo.

---

## Build e Release

### Build local (todos os targets)

Se o GoReleaser estiver instalado:

```bash
goreleaser build --snapshot --clean
```

Os binários serao gerados em `dist/` com nome
`site-research-mcp_<versao>_<os>_<arch>`.

**Fallback — build manual com `go build` (valida cross-compilacao):**

```bash
VERSION=0.1.0-dev
LDFLAGS="-s -w -X main.version=${VERSION}"

# darwin/arm64
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "${LDFLAGS}" \
  -o /tmp/site-research-mcp-darwin-arm64 ./cmd/site-research-mcp

# darwin/amd64
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS}" \
  -o /tmp/site-research-mcp-darwin-amd64 ./cmd/site-research-mcp

# linux/amd64
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS}" \
  -o /tmp/site-research-mcp-linux-amd64 ./cmd/site-research-mcp

# linux/arm64
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "${LDFLAGS}" \
  -o /tmp/site-research-mcp-linux-arm64 ./cmd/site-research-mcp

# windows/amd64
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "${LDFLAGS}" \
  -o /tmp/site-research-mcp-windows-amd64.exe ./cmd/site-research-mcp
```

Verifique os tamanhos dos binários produzidos:

```bash
ls -lh /tmp/site-research-mcp-*
```

Todos devem ser < 25 MB (orçamento CA-1 do `MCP_BRIEF.md` §RNF-04).

### Release com GoReleaser

Pre-requisito: o projeto precisa de um repositório git com ao menos um commit
e uma tag semântica.

```bash
git init
git add .
git commit -m "feat: Fase 3 — site-research-mcp v0.1.0"
git tag v0.1.0

# Release local (sem publicacao remota):
goreleaser build --snapshot --clean

# Release completa (requer GITHUB_TOKEN e remote configurado):
goreleaser release --clean
```

Enquanto o repositório git nao existir, use o build manual descrito acima.

---

## Contato e Licenca

Este binário faz parte do projeto `site-research` desenvolvido para o TRE-PI.
Consulte o `README.md` e o `MCP_BRIEF.md` na raiz do repositório para detalhes
sobre arquitetura, decisoes de design e fases do projeto.
