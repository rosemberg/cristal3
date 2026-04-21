#!/usr/bin/env python
"""Teste de inicialização do servidor"""

import sys
import asyncio
from pathlib import Path

# Adicionar src ao path
sys.path.insert(0, str(Path(__file__).parent))

async def test_server():
    print("1. Testando imports...")
    from src.models import CacheEntry, DocumentMetadata, ResearchResponse
    from src.cache import CacheManager
    from src.clients.site_research import SiteResearchClient
    from src.clients.http import HTTPClient
    print("   ✓ Imports OK")

    print("\n2. Testando configuração...")
    import yaml
    config_path = Path(__file__).parent / "config.yaml"
    config = yaml.safe_load(config_path.read_text())
    print(f"   ✓ Config carregado: {config.keys()}")

    print("\n3. Testando componentes...")
    cache = CacheManager(
        cache_dir=config['cache']['directory'],
        ttl_queries=config['cache']['ttl_queries'],
        ttl_documents=config['cache']['ttl_documents']
    )
    print("   ✓ CacheManager inicializado")

    site_research = SiteResearchClient(config['mcp']['site_research_url'])
    print("   ✓ SiteResearchClient inicializado")

    http_client = HTTPClient(
        timeout=config['http']['timeout'],
        max_retries=config['http']['max_retries']
    )
    print("   ✓ HTTPClient inicializado")

    print("\n4. Testando cache...")
    test_query = "test query"
    test_summary = {"result": "ok", "found": 1}
    cache.set_query(test_query, test_summary)
    cached = cache.get_query(test_query)
    assert cached is not None
    assert cached.summary["result"] == "ok"
    print("   ✓ Cache funcionando (set/get)")

    print("\n5. Testando cliente site-research...")
    results = await site_research.search("test")
    assert isinstance(results, list)
    assert len(results) > 0
    print(f"   ✓ Site research retornou {len(results)} resultados (mock)")

    print("\n6. Testando servidor MCP...")
    from mcp.server import Server
    server = Server("data-orchestrator")
    print("   ✓ Servidor MCP criado")

    print("\n✓ TODOS OS TESTES PASSARAM!")
    print("\nServidor pronto para uso.")
    print("\nPara iniciar o servidor:")
    print("  python -m src.server")

    # Limpar
    await http_client.close()

if __name__ == "__main__":
    asyncio.run(test_server())
