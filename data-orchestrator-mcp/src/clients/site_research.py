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
            # Verificar se processo ainda está vivo
            if self.mcp_client.process and self.mcp_client.process.returncode is None:
                return
            else:
                log.warning("site_research_process_died_on_check",
                           returncode=self.mcp_client.process.returncode if self.mcp_client.process else "no_process")
                self._initialized = False

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
        data_dir = base_dir / "data"
        env = os.environ.copy()
        env["CRISTAL_CONFIG"] = str(config_file)
        env["SITE_RESEARCH_DATA_DIR"] = str(data_dir)

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

    async def _reconnect_if_dead(self):
        """Reconecta se processo morreu"""
        if self.mcp_client and hasattr(self.mcp_client, 'process'):
            if self.mcp_client.process and self.mcp_client.process.returncode is not None:
                log.warning("site_research_process_died", returncode=self.mcp_client.process.returncode)
                self._initialized = False
                await self._ensure_connected()

    async def search(self, query: str, limit: int = 10) -> List[Dict[str, Any]]:
        """Busca no catálogo via MCP site-research"""

        await self._ensure_connected()
        await self._reconnect_if_dead()

        # Retry logic: tenta até 2 vezes
        max_attempts = 2
        for attempt in range(max_attempts):
            try:
                return await self.mcp_client.search(query, limit)
            except Exception as e:
                log.warning("search_failed_attempt",
                           attempt=attempt + 1,
                           max_attempts=max_attempts,
                           error=str(e))
                if attempt < max_attempts - 1:
                    # Reconectar antes de tentar novamente
                    self._initialized = False
                    await self._ensure_connected()
                else:
                    # Última tentativa falhou, propagar erro
                    raise

    async def inspect_page(self, url: str) -> Optional[Dict[str, Any]]:
        """Inspeciona página específica via MCP"""

        await self._ensure_connected()
        await self._reconnect_if_dead()

        # Retry logic: tenta até 2 vezes
        max_attempts = 2
        for attempt in range(max_attempts):
            try:
                return await self.mcp_client.inspect_page(url)
            except Exception as e:
                log.warning("inspect_page_failed_attempt",
                           attempt=attempt + 1,
                           max_attempts=max_attempts,
                           error=str(e))
                if attempt < max_attempts - 1:
                    # Reconectar antes de tentar novamente
                    self._initialized = False
                    await self._ensure_connected()
                else:
                    # Última tentativa falhou, propagar erro
                    raise

    async def close(self):
        """Fecha conexão com site-research"""

        if self.mcp_client:
            await self.mcp_client.disconnect()
            self.mcp_client = None
            self._initialized = False
