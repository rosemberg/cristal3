# Configuração do Claude Code

Este diretório contém a configuração do Claude Code para o projeto Site Research. O arquivo `settings.json` foi elaborado para permitir fluxo de desenvolvimento ágil sem perder camadas de segurança importantes.

## Filosofia

Três listas de permissões compõem a estratégia:

- **`allow`**: comandos rotineiros e seguros, aprovados automaticamente sem prompt
- **`deny`**: comandos perigosos, bloqueados mesmo que Claude peça
- **`ask`**: comandos sensíveis que sempre pedem confirmação explícita, mesmo que pareçam estar em `allow`

A regra de ouro: **permitir o que é rotineiro e reversível, bloquear o que é destrutivo ou expõe credenciais, perguntar para o que é raro mas legítimo.**

## O Que Está Liberado Automaticamente

Agrupamento lógico dos 126 comandos em `allow`:

**Toolchain Go** — tudo que você espera (`build`, `test`, `run`, `mod`, `vet`, `fmt`, `generate`, `doc`, `list`, `tool`), mais ferramentas auxiliares (`gofmt`, `goimports`, `golangci-lint`, `staticcheck`).

**CLI do próprio projeto** — invocações do binário `site-research` por qualquer caminho de build (`./bin/`, `./`, PATH).

**Leitura de filesystem** — tudo que explora sem modificar: `ls`, `tree`, `cat`, `head`, `tail`, `find`, `grep`, `rg`, `jq`, `yq`, `xmllint`, `sqlite3`, `diff`, entre outros.

**Escrita escopada ao projeto** — `mkdir`, `touch`, `cp`, `mv` apenas para paths relativos iniciados em `./`. Não consegue escrever em `~/` ou `/etc/` sem pedir.

**Limpeza de artefatos gerados** — `rm` específico para `./data/*.sqlite`, `./data/*.db`, `catalog.json`, coverage files, e limpeza total de `./bin/`, `./dist/`, `./tmp/`, `./coverage/`. Qualquer outro `rm -rf` cai em `ask`.

**Git rotineiro** — `status`, `diff`, `log`, `add`, `commit`, `checkout`, `stash`, `fetch`, `pull`, `branch`, `tag`. Operações de reescrita de histórico (`rebase`, `reset --hard`, `push`) caem em `ask`, `push --force` em `deny`.

**Inspeção de rede** — apenas `curl` e `wget` para o domínio do portal TRE-PI, `ping`, `host`, `dig`. Não consegue fazer `curl` para qualquer URL nem uploads.

**Compactação** — `gunzip`, `gzip`, `zcat`, `tar` e `unzip` em modo somente leitura (necessário para o sitemap vir como `.gz`).

**Inspeção de SQLite** — acesso a `./data/*.sqlite`, `./*.sqlite`, `./*.db` para debug do FTS gerado.

**Verificação de ambiente** — `which`, `whereis`, `type`, `command -v`, `printenv` apenas para variáveis inócuas (`PATH`, `HOME`, `GOPATH`, `GOROOT`).

**Hashing e encoding** — `md5sum`, `sha256sum`, `shasum`, `base64`, `xxd`, `hexdump` (usados no schema para `content_hash`).

## O Que Está Bloqueado (deny)

50 entradas que bloqueiam categorias de risco:

**Destruição em massa** — `rm -rf /`, `rm -rf ~`, `rm -rf ..` e variantes com `$HOME`.

**Escalação de privilégios** — `sudo`, `su`, `chmod -R 777`, `chown -R`.

**Operações de sistema** — `dd`, `mkfs`, `mount`, `umount`, `shutdown`, `reboot`, `systemctl`, `launchctl`, `killall`.

**Reescrita irreversível de git** — `push --force`, `reset --hard origin/*`, `clean -fdx`, `filter-branch`, `filter-repo`.

**Acesso a credenciais** — `cat` de `.env`, `~/.aws/credentials`, chaves SSH e PEM. Leitura direta desses arquivos via Read tool também bloqueada.

**Exfiltração de dados** — `scp`, `sftp`, `ssh`, `rsync --delete`.

**Login em serviços** — `gcloud auth`, `aws configure`, `docker login`, `gh auth login` (forçam login via shell separado, não pelo agente).

## O Que Sempre Pergunta (ask)

5 operações que são legítimas mas merecem confirmação:

- Qualquer `rm -rf` não coberto pelas exceções específicas em `allow`
- `git push` (qualquer forma não-force)
- `git rebase`
- `git reset --hard`
- `go get -u` (atualização de dependências — evita surpresas em builds)

## Variáveis de Ambiente

Quatro variáveis pré-configuradas:

- `GO111MODULE=on` — força modules mesmo em instalações antigas
- `CGO_ENABLED=0` — build puramente Go, sem dependências C (importante se o projeto usar `modernc.org/sqlite` em vez de `mattn/go-sqlite3`)
- `SITE_RESEARCH_CONFIG=./config.yaml` — caminho default do arquivo de configuração
- `SITE_RESEARCH_DATA_DIR=./data` — diretório onde o crawler escreve os `_index.json`

**Segredos nunca ficam aqui.** API keys (Gemini, Claude, OpenAI) devem estar em `.env` ou no keychain do sistema. O `settings.json` pode ser commitado no git sem risco.

## O Que Fazer Antes do Primeiro Commit

1. **Verifique o `.gitignore`** — garanta que `.env`, `./data/`, `*.sqlite`, `*.db` estão ignorados.
2. **Adicione suas API keys a `.env`** — seguindo o padrão `GEMINI_API_KEY=...`, `ANTHROPIC_API_KEY=...`. Claude Code não consegue ler esse arquivo (bloqueado em `deny`), mas seu programa Go sim.
3. **Commit do `settings.json`** — este arquivo é seguro para versionar; faz parte da configuração do projeto.

## Ajustando a Configuração

Se durante o desenvolvimento o Claude Code pedir permissão para algo rotineiro que não está em `allow`, adicione ali. Exemplo: se você começar a usar `delve` (debugger Go), adicione `Bash(dlv:*)`.

Se algo em `allow` estiver permitindo mais do que deveria, refine o pattern. Patterns suportam wildcards `*` e a sintaxe `Bash(comando:*)` significa "o comando exato seguido de qualquer argumento".

## Escopo do Arquivo

Este `settings.json` é **local do projeto** (`.claude/settings.json`). Configurações globais ficam em `~/.config/claude-code/settings.json` e são sobrescritas por este arquivo quando você abre o projeto.

Para configurações pessoais que não devem ser commitadas (ex: preferências de UI, atalhos), use `.claude/settings.local.json` e adicione ao `.gitignore`.
