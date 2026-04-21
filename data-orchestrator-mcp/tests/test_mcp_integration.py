"""Testes de integração MCP-to-MCP"""

import pytest
import os
from pathlib import Path
from src.clients.site_research_mcp_client import SiteResearchMCPClient

# Caminho absoluto para o binário
SITE_RESEARCH_BIN = "/Users/rosemberg/projetos-gemini/cristal3/bin/site-research-mcp"
BASE_DIR = "/Users/rosemberg/projetos-gemini/cristal3"

# Configurar ambiente padrão para testes
def get_test_env():
    """Retorna ambiente configurado para site-research"""
    env = os.environ.copy()
    env["CRISTAL_CONFIG"] = f"{BASE_DIR}/config.yaml"
    env["SITE_RESEARCH_DATA_DIR"] = f"{BASE_DIR}/data"
    return env


@pytest.mark.asyncio
async def test_site_research_mcp_connection():
    """Testa conexão básica com site-research MCP"""

    # Skip se binário não existe
    if not Path(SITE_RESEARCH_BIN).exists():
        pytest.skip(f"site-research-mcp não encontrado em {SITE_RESEARCH_BIN}")

    client = SiteResearchMCPClient(
        command=SITE_RESEARCH_BIN,
        cwd=BASE_DIR,
        env=get_test_env()
    )

    try:
        await client.connect()
        assert client.connected
        assert client.process is not None
    finally:
        await client.disconnect()
        assert not client.connected


@pytest.mark.asyncio
async def test_site_research_search():
    """Testa busca via MCP"""

    # Skip se binário não existe
    if not Path(SITE_RESEARCH_BIN).exists():
        pytest.skip(f"site-research-mcp não encontrado em {SITE_RESEARCH_BIN}")

    client = SiteResearchMCPClient(
        command=SITE_RESEARCH_BIN,
        cwd=BASE_DIR,
        env=get_test_env()
    )

    try:
        await client.connect()

        # Executar busca
        results = await client.search("diárias", limit=5)

        # Validações
        assert isinstance(results, list)

        # Se o índice estiver criado, deve retornar resultados
        if len(results) > 0:
            # Verificar estrutura dos resultados
            for result in results:
                # Deve ter pelo menos url ou title
                assert isinstance(result, dict)
                assert "url" in result or "title" in result

    finally:
        await client.disconnect()


@pytest.mark.asyncio
async def test_site_research_error_handling():
    """Testa tratamento de erros quando binário não existe"""

    # Usar caminho inválido
    invalid_path = "/caminho/invalido/site-research-mcp"

    client = SiteResearchMCPClient(
        command=invalid_path,
        cwd=BASE_DIR
    )

    # Deve lançar exceção ao tentar conectar
    with pytest.raises(Exception):
        await client.connect()

    # Cliente não deve estar conectado
    assert not client.connected


@pytest.mark.asyncio
async def test_site_research_inspect_page():
    """Testa inspecionar página específica via MCP"""

    # Skip se binário não existe
    if not Path(SITE_RESEARCH_BIN).exists():
        pytest.skip(f"site-research-mcp não encontrado em {SITE_RESEARCH_BIN}")

    client = SiteResearchMCPClient(
        command=SITE_RESEARCH_BIN,
        cwd=BASE_DIR,
        env=get_test_env()
    )

    try:
        await client.connect()

        # Primeiro buscar para obter uma URL válida
        results = await client.search("diárias", limit=1)

        if results and len(results) > 0:
            # Pegar primeira URL
            url = results[0].get("url")

            if url:
                # Inspecionar a página
                page_data = await client.inspect_page(url)

                # Verificar resultado
                if page_data:
                    assert isinstance(page_data, dict)
                    # Pode ter vários campos dependendo do que o site-research retorna

    finally:
        await client.disconnect()


@pytest.mark.asyncio
async def test_site_research_multiple_requests():
    """Testa múltiplas chamadas na mesma conexão"""

    # Skip se binário não existe
    if not Path(SITE_RESEARCH_BIN).exists():
        pytest.skip(f"site-research-mcp não encontrado em {SITE_RESEARCH_BIN}")

    client = SiteResearchMCPClient(
        command=SITE_RESEARCH_BIN,
        cwd=BASE_DIR,
        env=get_test_env()
    )

    try:
        await client.connect()

        # Fazer várias buscas sequenciais
        queries = ["diárias", "contratos", "licitações"]

        for query in queries:
            results = await client.search(query, limit=2)
            assert isinstance(results, list)

        # Conexão deve permanecer ativa
        assert client.connected

    finally:
        await client.disconnect()
