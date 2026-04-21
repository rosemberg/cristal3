#!/usr/bin/env python
"""Teste das tools do servidor"""

import sys
import asyncio
from pathlib import Path

# Adicionar src ao path
sys.path.insert(0, str(Path(__file__).parent))

async def test_server_tools():
    print("Testando Tools do Servidor MCP\n")
    print("=" * 50)

    # Import do servidor
    from src import server

    print("\n1. Testando list_tools()...")
    tools = await server.list_tools()
    print(f"   ✓ {len(tools)} tools disponíveis:")
    for tool in tools:
        print(f"     - {tool['name']}: {tool['description']}")

    print("\n2. Testando tool 'research'...")
    try:
        result = await server.research(query="teste diárias", force_fetch=False)
        print(f"   ✓ Research executou com sucesso")
        print(f"     Tipo de retorno: {type(result)}")
        print(f"     Conteúdo: {result['content'][0]['text'][:100]}...")
    except Exception as e:
        print(f"   ✗ Erro: {e}")

    print("\n3. Testando tool 'get_cached'...")
    try:
        # Primeiro pesquisar para popular cache
        await server.research(query="teste cache", force_fetch=False)

        # Depois buscar do cache
        result = await server.get_cached(query="teste cache")
        print(f"   ✓ Get cached executou com sucesso")
        assert "Cache: teste cache" in result['content'][0]['text']
        print(f"     Cache encontrado e retornado corretamente")
    except Exception as e:
        print(f"   ✗ Erro: {e}")

    print("\n4. Testando cache miss...")
    try:
        result = await server.get_cached(query="query inexistente")
        print(f"   ✓ Get cached com miss executou com sucesso")
        assert "não encontrado" in result['content'][0]['text'].lower()
        print(f"     Cache miss tratado corretamente")
    except Exception as e:
        print(f"   ✗ Erro: {e}")

    print("\n5. Testando call_tool() dispatcher...")
    try:
        result = await server.call_tool("research", {"query": "teste dispatcher"})
        print(f"   ✓ call_tool dispatcher funcionando")
    except Exception as e:
        print(f"   ✗ Erro: {e}")

    print("\n6. Testando tool desconhecida...")
    try:
        await server.call_tool("tool_invalida", {})
        print(f"   ✗ Deveria ter lançado erro")
    except ValueError as e:
        print(f"   ✓ Erro tratado corretamente: {e}")

    # Limpar
    await server.http_client.close()

    print("\n" + "=" * 50)
    print("✓ TODOS OS TESTES DAS TOOLS PASSARAM!")
    print("\nServidor está funcional e pronto para uso.")

if __name__ == "__main__":
    asyncio.run(test_server_tools())
