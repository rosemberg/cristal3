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

    async def connect(self, timeout: int = 10):
        """Inicia o processo site-research e conecta via stdio com timeout"""

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

            log.debug("site_research_process_started", pid=self.process.pid)

            # Enviar initialize com timeout
            init_response = await self._send_request("initialize", {
                "protocolVersion": "2025-11-25",
                "capabilities": {},
                "clientInfo": {
                    "name": "data-orchestrator",
                    "version": "1.0.0"
                }
            }, timeout=timeout)

            if "error" in init_response:
                raise Exception(f"Initialize failed: {init_response['error']}")

            # NOTE: NÃO enviar notifications/initialized - não é necessário no protocol 2025-11-25
            # e causa warnings no servidor Python MCP

            self.connected = True
            log.info("connected_to_site_research", protocol="2025-11-25")

        except Exception as e:
            log.error("connection_failed", error=str(e), exc_info=True)
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

    async def search(self, query: str, limit: int = 10) -> str:
        """Chama o tool 'search' do site-research - retorna Markdown formatado"""

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
                return "❌ Erro ao buscar: " + str(response["error"])

            result = response.get("result", {})
            content = result.get("content", [])

            # Extrair texto Markdown do content
            text_parts = []
            for item in content:
                if item.get("type") == "text":
                    text_parts.append(item.get("text", ""))

            markdown = "\n".join(text_parts)
            log.info("search_results", len=len(markdown))
            return markdown

        except Exception as e:
            log.error("search_failed", error=str(e))
            return f"❌ Erro ao buscar: {str(e)}"

    async def inspect_page(self, url: str) -> Optional[Dict[str, Any]]:
        """Chama o tool 'inspect_page' do site-research - retorna dados estruturados"""

        if not self.connected:
            await self.connect()

        log.info("calling_site_research_inspect", url=url)

        try:
            # Chamar tool via MCP
            response = await self._call_tool("inspect_page", {"url": url})

            if "error" in response:
                log.error("inspect_error", error=response["error"])
                return None

            result = response.get("result", {})
            content = result.get("content", [])

            # Parsear JSON do primeiro item de texto
            for item in content:
                if item.get("type") == "text":
                    text = item.get("text", "")
                    try:
                        # inspect_page retorna JSON estruturado
                        parsed = json.loads(text)
                        return parsed
                    except json.JSONDecodeError:
                        log.warning("failed_to_parse_inspect_result", text=text[:200])
                        return None

            return None

        except Exception as e:
            log.error("inspect_failed", error=str(e))
            return None

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

    async def _send_request(self, method: str, params: Dict[str, Any], timeout: int = 30) -> Dict:
        """Envia request JSON-RPC e espera resposta com timeout"""

        self._request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self._request_id,
            "method": method,
            "params": params
        }

        log.debug("sending_mcp_request", method=method, id=self._request_id)

        # Enviar via stdin
        request_json = json.dumps(request) + "\n"
        self.process.stdin.write(request_json.encode())
        await self.process.stdin.drain()

        log.debug("mcp_request_sent", method=method, waiting_for_response=True)

        # Ler resposta via stdout com timeout
        response = await self._read_response(timeout=timeout)

        log.debug("mcp_response_received", method=method, id=self._request_id)

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

    async def _read_response(self, timeout: int = 30) -> Dict:
        """Lê resposta JSON-RPC do stdout com timeout"""

        try:
            line = await asyncio.wait_for(
                self.process.stdout.readline(),
                timeout=timeout
            )
        except asyncio.TimeoutError:
            raise Exception(f"site-research não respondeu em {timeout}s - processo pode estar travado")

        if not line:
            # Tentar ler stderr para debug
            stderr_output = ""
            try:
                # Non-blocking read do stderr se disponível
                stderr_line = await asyncio.wait_for(
                    self.process.stderr.readline(),
                    timeout=0.1
                )
                if stderr_line:
                    stderr_output = stderr_line.decode()
            except asyncio.TimeoutError:
                pass

            error_msg = "Connection closed by site-research"
            if stderr_output:
                error_msg += f" - stderr: {stderr_output}"

            raise Exception(error_msg)

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
