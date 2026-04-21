# Plano de Implementação - Integração MCP-to-MCP

## Objetivo

Permitir que o **data-orchestrator** (Python) chame o **site-research** (Go) via protocolo MCP.

## Desafio

- data-orchestrator é um MCP **server**
- site-research é um MCP **server**
- Precisamos que data-orchestrator seja **client** do site-research

## Solução: Cliente MCP Interno

O data-orchestrator vai:
1. Inicializar um **cliente MCP** interno
2. Conectar ao site-research via **stdio**
3. Chamar tools do site-research
4. Processar resultados

## Arquitetura

```
Claude Code (client MCP)
    ↓ stdio
┌─────────────────────────────────┐
│ data-orchestrator (server MCP)  │
│                                 │
│  ┌───────────────────────────┐ │
│  │ MCP Client Interno        │ │
│  │   ↓ stdio subprocess      │ │
│  │ ┌─────────────────────┐   │ │
│  │ │ site-research       │   │ │
│  │ │ (server MCP Go)     │   │ │
│  │ └─────────────────────┘   │ │
│  └───────────────────────────┘ │
└─────────────────────────────────┘
```

## Implementação

### 1. Atualizar requirements.txt

Adicionar dependências do cliente MCP:

```txt
# Já existente
mcp>=1.0.0

# Cliente MCP (já incluído no mcp)
# Não precisa adicionar nada novo
```

### 2. Criar Cliente MCP para Site-Research

**Novo arquivo: `src/clients/site_research_mcp_client.py`**

```python
"""Cliente MCP para site-research server"""

import asyncio
import json
from typing import List, Dict, Any, Optional
from contextlib import asynccontextmanager
import structlog

log = structlog.get_logger()

class SiteResearchMCPClient:
    """Cliente MCP que conecta ao site-research server via stdio"""
    
    def __init__(self, command: str, args: List[str] = None, cwd: str = None, env: Dict[str, str] = None):
        """
        Args:
            command: Comando para iniciar o site-research MCP (ex: /path/to/site-research-mcp)
            args: Argumentos do comando
            cwd: Working directory
            env: Variáveis de ambiente
        """
        self.command = command
        self.args = args or []
        self.cwd = cwd
        self.env = env or {}
        self.process = None
        self.connected = False
        self._request_id = 0
    
    async def connect(self):
        """Inicia o processo site-research e conecta via stdio"""
        
        if self.connected:
            return
        
        log.info("connecting_to_site_research", command=self.command)
        
        try:
            # Iniciar processo site-research-mcp
            self.process = await asyncio.create_subprocess_exec(
                self.command,
                *self.args,
                stdin=asyncio.subprocess.PIPE,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                cwd=self.cwd,
                env=self.env
            )
            
            # Enviar initialize
            await self._send_request("initialize", {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {
                    "name": "data-orchestrator",
                    "version": "1.0.0"
                }
            })
            
            # Esperar resposta
            response = await self._read_response()
            
            if "error" in response:
                raise Exception(f"Initialize failed: {response['error']}")
            
            # Enviar initialized notification
            await self._send_notification("notifications/initialized", {})
            
            self.connected = True
            log.info("connected_to_site_research")
            
        except Exception as e:
            log.error("connection_failed", error=str(e))
            if self.process:
                self.process.kill()
                await self.process.wait()
            raise
    
    async def disconnect(self):
        """Desconecta do site-research"""
        
        if not self.connected:
            return
        
        log.info("disconnecting_from_site_research")
        
        if self.process:
            self.process.terminate()
            await self.process.wait()
            self.process = None
        
        self.connected = False
    
    async def search(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Chama o tool 'search' do site-research"""
        
        if not self.connected:
            await self.connect()
        
        log.info("calling_site_research_search", query=query, limit=limit)
        
        try:
            # Chamar tool via MCP
            response = await self._call_tool("search", {
                "query": query,
                "limit": limit
            })
            
            # Parsear resposta
            if "error" in response:
                log.error("search_error", error=response["error"])
                return []
            
            result = response.get("result", {})
            content = result.get("content", [])
            
            # Extrair resultados do content
            results = []
            for item in content:
                if item.get("type") == "text":
                    # Parsear JSON do texto
                    text = item.get("text", "")
                    try:
                        parsed = json.loads(text)
                        if isinstance(parsed, list):
                            results.extend(parsed)
                        else:
                            results.append(parsed)
                    except json.JSONDecodeError:
                        log.warning("failed_to_parse_result", text=text)
            
            log.info("search_results", count=len(results))
            return results
            
        except Exception as e:
            log.error("search_failed", error=str(e))
            return []
    
    async def inspect_page(self, url: str) -> Optional[Dict[str, Any]]:
        """Chama o tool 'inspect_page' do site-research"""
        
        if not self.connected:
            await self.connect()
        
        log.info("calling_site_research_inspect", url=url)
        
        try:
            response = await self._call_tool("inspect_page", {
                "url": url
            })
            
            if "error" in response:
                log.error("inspect_error", error=response["error"])
                return None
            
            result = response.get("result", {})
            content = result.get("content", [])
            
            # Extrair dados
            if content and content[0].get("type") == "text":
                text = content[0].get("text", "")
                try:
                    return json.loads(text)
                except json.JSONDecodeError:
                    log.warning("failed_to_parse_inspect", text=text)
            
            return None
            
        except Exception as e:
            log.error("inspect_failed", error=str(e))
            return None
    
    async def _call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Dict:
        """Chama um tool via MCP protocol"""
        
        response = await self._send_request("tools/call", {
            "name": tool_name,
            "arguments": arguments
        })
        
        return response
    
    async def _send_request(self, method: str, params: Dict[str, Any]) -> Dict:
        """Envia request JSON-RPC e espera resposta"""
        
        self._request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self._request_id,
            "method": method,
            "params": params
        }
        
        # Enviar via stdin
        request_json = json.dumps(request) + "\n"
        self.process.stdin.write(request_json.encode())
        await self.process.stdin.drain()
        
        # Ler resposta via stdout
        response = await self._read_response()
        return response
    
    async def _send_notification(self, method: str, params: Dict[str, Any]):
        """Envia notification JSON-RPC (sem esperar resposta)"""
        
        notification = {
            "jsonrpc": "2.0",
            "method": method,
            "params": params
        }
        
        notification_json = json.dumps(notification) + "\n"
        self.process.stdin.write(notification_json.encode())
        await self.process.stdin.drain()
    
    async def _read_response(self) -> Dict:
        """Lê resposta JSON-RPC do stdout"""
        
        line = await self.process.stdout.readline()
        
        if not line:
            raise Exception("Connection closed by site-research")
        
        try:
            return json.loads(line.decode())
        except json.JSONDecodeError as e:
            log.error("invalid_json_response", line=line.decode(), error=str(e))
            raise


@asynccontextmanager
async def site_research_client(command: str, args: List[str] = None, cwd: str = None, env: Dict[str, str] = None):
    """Context manager para cliente site-research"""
    
    client = SiteResearchMCPClient(command, args, cwd, env)
    try:
        await client.connect()
        yield client
    finally:
        await client.disconnect()
```

### 3. Atualizar src/clients/site_research.py

Substituir o código atual por wrapper que usa o cliente MCP:

```python
"""Cliente para site-research MCP server"""

from typing import List, Dict, Any, Optional
import os
from pathlib import Path
import structlog

from .site_research_mcp_client import SiteResearchMCPClient

log = structlog.get_logger()

class SiteResearchClient:
    """Wrapper para site-research MCP client"""
    
    def __init__(self, url: str = "stdio"):
        """
        Args:
            url: "stdio" para local, ou URL para remoto (não implementado ainda)
        """
        self.url = url
        self.mcp_client: Optional[SiteResearchMCPClient] = None
        self._initialized = False
    
    async def _ensure_connected(self):
        """Garante que está conectado ao site-research"""
        
        if self._initialized and self.mcp_client and self.mcp_client.connected:
            return
        
        # Configurar comando do site-research
        # Por padrão, procurar no diretório pai
        base_dir = Path(__file__).parent.parent.parent.parent  # cristal3/
        site_research_bin = base_dir / "bin" / "site-research-mcp"
        
        if not site_research_bin.exists():
            raise FileNotFoundError(
                f"site-research-mcp não encontrado em: {site_research_bin}\n"
                f"Compile o site-research primeiro:\n"
                f"  cd {base_dir}\n"
                f"  go build -o bin/site-research-mcp ./cmd/site-research-mcp"
            )
        
        # Configurar ambiente
        config_file = base_dir / "config.yaml"
        env = os.environ.copy()
        env["CRISTAL_CONFIG"] = str(config_file)
        
        log.info("initializing_site_research_client", 
                 command=str(site_research_bin),
                 config=str(config_file))
        
        # Criar cliente MCP
        self.mcp_client = SiteResearchMCPClient(
            command=str(site_research_bin),
            args=[],
            cwd=str(base_dir),
            env=env
        )
        
        # Conectar
        await self.mcp_client.connect()
        self._initialized = True
        
        log.info("site_research_client_ready")
    
    async def search(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Busca no catálogo via MCP site-research"""
        
        await self._ensure_connected()
        return await self.mcp_client.search(query, limit)
    
    async def inspect_page(self, url: str) -> Optional[Dict[str, Any]]:
        """Inspeciona página específica via MCP"""
        
        await self._ensure_connected()
        return await self.mcp_client.inspect_page(url)
    
    async def close(self):
        """Fecha conexão com site-research"""
        
        if self.mcp_client:
            await self.mcp_client.disconnect()
            self.mcp_client = None
            self._initialized = False
```

### 4. Atualizar src/server.py

Garantir que a conexão seja gerenciada corretamente:

```python
# No início do arquivo, após imports
from .clients.site_research import SiteResearchClient

# Criar cliente global
site_research = SiteResearchClient(config['mcp']['site_research_url'])

# Adicionar cleanup no shutdown
async def main():
    log.info("starting_server")
    
    try:
        async with stdio_server() as (read_stream, write_stream):
            await server.run(read_stream, write_stream, server.create_initialization_options())
    finally:
        # Cleanup
        log.info("shutting_down")
        await site_research.close()
        await http_client.close()
```

### 5. Tratamento de Erros Específicos

Atualizar `research()` para lidar com erros do site-research:

```python
async def research(query: str, force_fetch: bool = False):
    """Busca completa com dados extraídos"""
    
    # ... cache check ...
    
    # Buscar no site-research
    try:
        log.info("searching_portal", query=query)
        results = await site_research.search(query, limit=10)
        
    except FileNotFoundError as e:
        # site-research-mcp não compilado
        return {
            "content": [{
                "type": "text",
                "text": f"""
❌ **ERRO:** site-research MCP não encontrado

{str(e)}

**Solução:**
1. Compile o site-research:
   cd /Users/rosemberg/projetos-gemini/cristal3
   go build -o bin/site-research-mcp ./cmd/site-research-mcp

2. Verifique se o índice existe:
   ls -la data/index/
"""
            }]
        }
    
    except Exception as e:
        # Outros erros de conexão
        log.error("site_research_failed", error=str(e))
        return {
            "content": [{
                "type": "text",
                "text": f"""
❌ **ERRO:** Falha ao conectar com site-research

{str(e)}

**Possíveis causas:**
1. site-research-mcp não está rodando
2. Índice do portal não foi criado
3. Erro de configuração

**Para debugar:**
1. Teste o site-research diretamente:
   /path/to/site-research-mcp

2. Verifique os logs
"""
            }]
        }
    
    # ... resto do código ...
```

## Teste de Integração

**Novo arquivo: `tests/test_mcp_integration.py`**

```python
"""Testes de integração MCP-to-MCP"""

import pytest
from src.clients.site_research_mcp_client import SiteResearchMCPClient

@pytest.mark.asyncio
async def test_site_research_mcp_connection():
    """Testa conexão básica com site-research MCP"""
    
    client = SiteResearchMCPClient(
        command="/Users/rosemberg/projetos-gemini/cristal3/bin/site-research-mcp",
        cwd="/Users/rosemberg/projetos-gemini/cristal3"
    )
    
    try:
        await client.connect()
        assert client.connected
    finally:
        await client.disconnect()

@pytest.mark.asyncio
async def test_site_research_search():
    """Testa busca via MCP"""
    
    client = SiteResearchMCPClient(
        command="/Users/rosemberg/projetos-gemini/cristal3/bin/site-research-mcp",
        cwd="/Users/rosemberg/projetos-gemini/cristal3"
    )
    
    try:
        await client.connect()
        
        results = await client.search("diárias", limit=5)
        
        assert isinstance(results, list)
        assert len(results) > 0
        
        # Verificar estrutura
        for result in results:
            assert "url" in result or "title" in result
            
    finally:
        await client.disconnect()
```

## Ordem de Implementação

1. ✅ Criar `src/clients/site_research_mcp_client.py`
2. ✅ Atualizar `src/clients/site_research.py`
3. ✅ Atualizar `src/server.py` (cleanup)
4. ✅ Melhorar tratamento de erros em `research()`
5. ✅ Criar testes de integração
6. ✅ Testar manualmente

## Critérios de Aceitação

- [ ] data-orchestrator inicia subprocesso site-research-mcp
- [ ] Comunicação via stdio funciona
- [ ] Tool `search` retorna resultados reais
- [ ] Tool `inspect_page` funciona
- [ ] Erros são tratados graciosamente
- [ ] Conexão é fechada no shutdown

## Benefícios

✅ **Integração real** site-research ↔ data-orchestrator  
✅ **Busca no portal** funcionando  
✅ **Dados reais** do TRE-PI  
✅ **Rastreabilidade** completa  
✅ **Sistema end-to-end** funcional  

## Tempo Estimado

- Cliente MCP: 1h
- Wrapper: 30min
- Tratamento de erros: 30min
- Testes: 1h

**Total: ~3h**
