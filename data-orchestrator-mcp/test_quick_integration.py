#!/usr/bin/env python3
"""Teste rápido da integração MCP"""

import asyncio
import sys
from pathlib import Path

# Add src to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from clients.site_research import SiteResearchClient

async def test_integration():
    """Teste rápido de integração"""

    print("🔍 Testando integração MCP-to-MCP...\n")

    client = SiteResearchClient()

    try:
        print("1. Conectando ao site-research...")
        results = await client.search("diárias", limit=3)

        print(f"✅ Conectado! Encontrados {len(results)} resultados\n")

        if results:
            print("📄 Primeiros resultados:")
            for i, result in enumerate(results[:3], 1):
                title = result.get('title', 'Sem título')
                url = result.get('url', 'Sem URL')
                print(f"\n{i}. {title}")
                print(f"   URL: {url}")

        print("\n✅ Integração MCP-to-MCP funcionando!")
        return True

    except FileNotFoundError as e:
        print(f"❌ Erro: {e}")
        return False
    except Exception as e:
        print(f"❌ Erro inesperado: {e}")
        import traceback
        traceback.print_exc()
        return False
    finally:
        print("\n2. Desconectando...")
        await client.close()
        print("✅ Desconectado")

if __name__ == "__main__":
    success = asyncio.run(test_integration())
    sys.exit(0 if success else 1)
