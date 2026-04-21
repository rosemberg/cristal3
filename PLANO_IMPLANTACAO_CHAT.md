# Plano de Implantação - Cristal Chat em cristal-data

**Data**: 2026-04-21  
**Objetivo**: Implantar o cristal-chat em `/Users/rosemberg/projetos-gemini/cristal-data` para uso integrado com os MCPs

---

## 1. Análise do Ambiente Atual

### 1.1 Estrutura de cristal-data

```
cristal-data/
├── .mcp.json                    # Config MCP para Claude Code
├── README.md                    # Documentação do ambiente
├── test-data/                   # Dados de teste (CSVs, TXT)
├── cache/                       # Cache compartilhado
├── docs/                        # Documentação
└── [múltiplos .md]             # Guias de uso
```

**Característica**: Ambiente de teste isolado que aponta para os MCPs em `cristal3/`

### 1.2 MCPs Disponíveis

**1. data-orchestrator-mcp** (Python)
- Localização: `/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp`
- Tools: research, get_cached, get_document, metrics
- Config em: `.mcp.json`
- Venv: `venv/bin/python`

**2. site-research-mcp** (Go)
- Localização: `/Users/rosemberg/projetos-gemini/cristal3/cmd/site-research-mcp`
- Tools: search, inspect_page, catalog_stats
- Binário: Precisa compilar ou usar existente
- Acesso: Via data-orchestrator (stdio chain)

### 1.3 Cristal Chat Atual

- Localização: `/Users/rosemberg/projetos-gemini/cristal3/cristal-chat`
- Status: MVP funcional (Sprint 1 completo)
- Binário: `bin/cristal`
- Hardcoded path: Aponta direto para data-orchestrator

---

## 2. Estratégia de Implantação

### Opção A: Link Simbólico (Recomendada) ✅

**Vantagens**:
- Mantém código fonte em cristal3/
- Fácil atualização
- Binário acessível de cristal-data/

**Estrutura**:
```
cristal-data/
├── cristal -> /Users/rosemberg/projetos-gemini/cristal3/cristal-chat/bin/cristal
├── cristal-config.yaml
├── start-chat.sh
└── [resto do ambiente]
```

### Opção B: Cópia Completa

**Vantagens**:
- Isolamento total
- Personalização específica

**Desvantagens**:
- Manutenção duplicada
- Sincronização manual

**Recomendação**: **Opção A** (link simbólico)

---

## 3. Plano de Implantação Detalhado

### Fase 1: Preparação do Binário

#### 3.1.1 Compilar cristal-chat (se necessário)
```bash
cd /Users/rosemberg/projetos-gemini/cristal3/cristal-chat
go build -o bin/cristal ./cmd/cristal
```

#### 3.1.2 Verificar binário
```bash
./bin/cristal --version
# Esperado: cristal v0.1.0-dev
```

### Fase 2: Configuração em cristal-data

#### 3.2.1 Criar link simbólico
```bash
cd /Users/rosemberg/projetos-gemini/cristal-data
ln -s /Users/rosemberg/projetos-gemini/cristal3/cristal-chat/bin/cristal ./cristal
```

#### 3.2.2 Criar config.yaml
**Arquivo**: `cristal-data/cristal-config.yaml`

```yaml
mcp:
  data_orchestrator:
    python_path: "/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp/venv/bin/python"
    script_path: "src.server"
    working_dir: "/Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp"
    timeout: 120

chat:
  history_size: 50
  save_history: true
  session_dir: "/Users/rosemberg/projetos-gemini/cristal-data/.sessions"

ui:
  color: true
  show_timestamps: false
  prompt: "🔮 > "

# Notas:
# - data_orchestrator acessa automaticamente o site-research-mcp
# - Cache é compartilhado em /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp/cache
# - Sessões salvas em cristal-data/.sessions (isolado)
```

#### 3.2.3 Criar script de inicialização
**Arquivo**: `cristal-data/start-chat.sh`

```bash
#!/bin/bash
# Start Cristal Chat em cristal-data
# Usa config local e conecta aos MCPs

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "🔮 Iniciando Cristal Chat..."
echo ""
echo "Ambiente: cristal-data"
echo "Config: cristal-config.yaml"
echo "MCPs: data-orchestrator + site-research"
echo ""

# Verificar se binário existe
if [ ! -f "./cristal" ]; then
    echo "❌ Erro: binário cristal não encontrado"
    echo "Execute: ln -s ../cristal3/cristal-chat/bin/cristal ./cristal"
    exit 1
fi

# Verificar se config existe
if [ ! -f "./cristal-config.yaml" ]; then
    echo "❌ Erro: cristal-config.yaml não encontrado"
    exit 1
fi

# Criar diretório de sessões se não existir
mkdir -p .sessions

# Executar cristal
./cristal --config cristal-config.yaml "$@"
```

```bash
chmod +x cristal-data/start-chat.sh
```

#### 3.2.4 Criar script de debug
**Arquivo**: `cristal-data/start-chat-debug.sh`

```bash
#!/bin/bash
# Start Cristal Chat em modo debug

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

echo "🔍 Iniciando Cristal Chat (DEBUG)..."
echo ""

./cristal --config cristal-config.yaml --debug "$@"
```

```bash
chmod +x cristal-data/start-chat-debug.sh
```

### Fase 3: Documentação

#### 3.3.1 Criar CHAT_README.md
**Arquivo**: `cristal-data/CHAT_README.md`

```markdown
# Cristal Chat - Guia de Uso

Chat CLI para consultar o portal de transparência TRE-PI via MCPs.

## 🚀 Iniciar

```bash
./start-chat.sh
```

Ou em modo debug:
```bash
./start-chat-debug.sh
```

## 💬 Comandos Disponíveis

### Comandos Especiais
```
/help          - Mostra ajuda
/quit, /q      - Sai do chat
/tools         - Lista tools do MCP
```

### Consultas

Digite perguntas naturais:

```
quanto foi gasto com diárias em 2026
contratos de licitação
balancetes de março
remuneração de servidores
```

## 📊 Como Funciona

```
Você ──> Cristal Chat ──> data-orchestrator-mcp ──> site-research-mcp ──> Catálogo (656 páginas)
                                      ↓
                               Extrai PDFs/CSVs
                                      ↓
                               Formata resposta
                                      ↓
                            Você recebe resultado
```

## 🎯 Exemplos de Uso

### Exemplo 1: Busca Simples
```
🔮 > diárias 2026

🔍 Pesquisando...
[Resultado formatado com páginas encontradas]
```

### Exemplo 2: Dados Locais
```
🔮 > extrair test-data/diarias-janeiro-2026.csv

[Dados extraídos e agregados]
Total: R$ 5.789,67
```

### Exemplo 3: Comandos
```
🔮 > /tools

research
get_cached
get_document
metrics
```

## ⚙️ Configuração

**Arquivo**: `cristal-config.yaml`

- MCP: Aponta para data-orchestrator em cristal3/
- Chat: Histórico de 50 mensagens
- Sessões: Salvas em `.sessions/`

## 🔧 Troubleshooting

### Erro: binário não encontrado
```bash
ln -s /Users/rosemberg/projetos-gemini/cristal3/cristal-chat/bin/cristal ./cristal
```

### Erro: MCP não conecta
Verificar se data-orchestrator está OK:
```bash
cd /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp
venv/bin/python -m src.server
# Ctrl+C para sair
```

### Erro: timeout em queries
Aumentar timeout em `cristal-config.yaml`:
```yaml
mcp:
  data_orchestrator:
    timeout: 180  # 3 minutos
```

## 📁 Estrutura

```
cristal-data/
├── cristal                 # Binário (link simbólico)
├── cristal-config.yaml     # Config do chat
├── start-chat.sh           # Script de inicialização
├── start-chat-debug.sh     # Script debug
├── .sessions/              # Histórico de sessões
├── test-data/              # Dados de teste
└── CHAT_README.md          # Este arquivo
```

## 🎨 Próximas Melhorias (Sprint 2)

Quando implementar Sprint 2, você terá:
- ✨ Formatação bonita (tabelas, cores)
- 💰 Valores monetários formatados
- 📊 Comando /metrics
- 📄 Parse estruturado de páginas

## 📚 Documentação Completa

- **Plano de Implementação**: `../cristal3/PLANO_CHAT_CRISTAL.md`
- **MCP Orchestrator**: `../cristal3/data-orchestrator-mcp/README.md`
- **MCP Research**: `../cristal3/cmd/site-research-mcp/README.md`
```

#### 3.3.2 Atualizar README.md principal
Adicionar seção sobre o chat no `cristal-data/README.md`

### Fase 4: Testes de Integração

#### 3.4.1 Teste básico de inicialização
```bash
cd /Users/rosemberg/projetos-gemini/cristal-data
./start-chat.sh
# Verificar: welcome screen, prompt, conexão MCP
# Comando: /quit
```

#### 3.4.2 Teste de comandos
```bash
./start-chat.sh
/help       # Deve mostrar ajuda
/tools      # Deve listar 4 tools
/quit       # Deve sair gracefully
```

#### 3.4.3 Teste de query simples
```bash
./start-chat.sh
> teste
# Deve chamar data-orchestrator e retornar resposta
```

#### 3.4.4 Teste com dados locais
```bash
./start-chat.sh
> extrair test-data/diarias-janeiro-2026.csv
# Deve ler CSV e mostrar totais
```

#### 3.4.5 Teste de sessão
```bash
./start-chat.sh
> query 1
> query 2
> /quit
# Verificar: sessão salva em .sessions/
```

---

## 4. Estrutura Final

```
cristal-data/
├── .mcp.json                          # Config MCP para Claude Code
├── cristal                            # Link → cristal3/cristal-chat/bin/cristal
├── cristal-config.yaml                # Config do chat
├── start-chat.sh                      # Script de início
├── start-chat-debug.sh                # Script debug
├── CHAT_README.md                     # Guia de uso do chat
├── README.md                          # README principal (atualizado)
├── .sessions/                         # Sessões do chat
│   └── 2026-04-21_14-30/             # Sessão exemplo
│       ├── session_xxx.json
│       └── history_xxx.json
├── test-data/                         # Dados de teste
│   ├── diarias-janeiro-2026.csv
│   ├── diarias-fevereiro-2026.csv
│   └── ...
├── cache/                             # Cache (compartilhado com cristal3)
└── docs/                              # Documentação
```

---

## 5. Checklist de Implantação

### Pré-requisitos
- [ ] Cristal-chat compilado em cristal3/
- [ ] Data-orchestrator funcionando
- [ ] Site-research-mcp compilado (se necessário)

### Implantação
- [ ] Criar link simbólico do binário
- [ ] Criar cristal-config.yaml
- [ ] Criar start-chat.sh
- [ ] Criar start-chat-debug.sh
- [ ] Criar CHAT_README.md
- [ ] Atualizar README.md principal
- [ ] Criar diretório .sessions/

### Testes
- [ ] Teste de inicialização
- [ ] Teste de comandos (/help, /tools, /quit)
- [ ] Teste de query simples
- [ ] Teste com dados locais
- [ ] Teste de sessão/histórico
- [ ] Teste em modo debug

### Documentação
- [ ] CHAT_README.md completo
- [ ] Exemplos de uso documentados
- [ ] Troubleshooting documentado
- [ ] Atualizar README.md com seção do chat

---

## 6. Vantagens desta Implantação

### ✅ Isolamento
- Ambiente de teste separado
- Sessões próprias em .sessions/
- Não afeta cristal3/

### ✅ Integração
- Acessa os mesmos MCPs
- Cache compartilhado (economiza recursos)
- Config centralizada

### ✅ Manutenção
- Código fonte em um só lugar (cristal3/)
- Link simbólico facilita updates
- Scripts de inicialização padronizados

### ✅ Facilidade de Uso
- `./start-chat.sh` e pronto
- Debug mode disponível
- Documentação local (CHAT_README.md)

---

## 7. Próximos Passos Pós-Implantação

### Imediato
1. Testar com dados de test-data/
2. Validar integração MCP completa
3. Documentar casos de uso reais

### Curto Prazo (Sprint 2)
1. Implementar formatação bonita
2. Adicionar comando /metrics
3. Parse estruturado de respostas

### Médio Prazo
1. Integração com dados reais do TRE-PI
2. Workflows complexos
3. Análises comparativas

---

## 8. Comandos Úteis

### Rebuild do binário
```bash
cd /Users/rosemberg/projetos-gemini/cristal3/cristal-chat
go build -o bin/cristal ./cmd/cristal
# O link simbólico em cristal-data/ já aponta para o novo binário
```

### Limpar sessões antigas
```bash
cd /Users/rosemberg/projetos-gemini/cristal-data
rm -rf .sessions/*
```

### Verificar logs do MCP
```bash
cd /Users/rosemberg/projetos-gemini/cristal-data
./start-chat-debug.sh 2>&1 | tee chat-debug.log
```

### Ver cache do orchestrator
```bash
cd /Users/rosemberg/projetos-gemini/cristal3/data-orchestrator-mcp
python scripts/clean_cache.py --stats
```

---

## 9. Estimativa de Tempo

| Fase | Tempo Estimado |
|------|----------------|
| Fase 1: Preparação | 5 min |
| Fase 2: Configuração | 15 min |
| Fase 3: Documentação | 20 min |
| Fase 4: Testes | 15 min |
| **Total** | **~1 hora** |

---

## 10. Contato e Suporte

**Projeto**: Cristal Chat  
**Ambiente**: cristal-data  
**Data**: 2026-04-21  
**Status**: Planejamento completo  

**Documentação**:
- Este plano: `PLANO_IMPLANTACAO_CHAT.md`
- Plano de desenvolvimento: `PLANO_CHAT_CRISTAL.md`
- Guia de uso: `cristal-data/CHAT_README.md` (após implantação)

---

**Pronto para executar!** 🚀
