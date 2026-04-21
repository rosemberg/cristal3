"""
Teste manual end-to-end da Fase 3

Este arquivo pode ser executado para testar manualmente o servidor
com queries realistas.

Execute com:
    python tests/test_manual_phase3.py
"""

import asyncio
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))

from src.server import research
import structlog

# Configurar logging para output legível
structlog.configure(
    processors=[
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.add_log_level,
        structlog.dev.ConsoleRenderer()
    ]
)

log = structlog.get_logger()


async def test_query_with_extraction():
    """Testa query que deve acionar extração automática"""
    log.info("test_start", test="query_with_extraction")

    queries = [
        "quanto foi gasto em diárias em 2026",
        "qual o valor total de passagens",
        "total de gastos com diárias",
        "custo de despesas em janeiro"
    ]

    for query in queries:
        log.info("testing_query", query=query)
        result = await research(query)

        print("\n" + "="*80)
        print(f"QUERY: {query}")
        print("="*80)
        print(result["content"][0]["text"])
        print("\n")


async def test_query_without_extraction():
    """Testa query que NÃO deve acionar extração"""
    log.info("test_start", test="query_without_extraction")

    queries = [
        "diárias e passagens",
        "relatórios de transparência",
        "documentos de recursos humanos"
    ]

    for query in queries:
        log.info("testing_query", query=query)
        result = await research(query)

        print("\n" + "="*80)
        print(f"QUERY: {query}")
        print("="*80)
        print(result["content"][0]["text"])
        print("\n")


async def test_cache_behavior():
    """Testa comportamento de cache"""
    log.info("test_start", test="cache_behavior")

    query = "quanto foi gasto em diárias"

    # Primeira chamada
    log.info("first_call", query=query)
    result1 = await research(query)

    # Segunda chamada (deve usar cache)
    log.info("second_call_should_use_cache", query=query)
    result2 = await research(query)

    # Terceira chamada forçando fetch
    log.info("third_call_force_fetch", query=query)
    result3 = await research(query, force_fetch=True)

    print("\n" + "="*80)
    print("CACHE TEST")
    print("="*80)
    print("\n1. Primeira chamada:")
    print(result1["content"][0]["text"])
    print("\n2. Segunda chamada (cache):")
    print(result2["content"][0]["text"])
    print("\n3. Terceira chamada (force_fetch):")
    print(result3["content"][0]["text"])
    print("\n")


async def main():
    """Executa todos os testes manuais"""
    print("\n" + "="*80)
    print("TESTES MANUAIS - FASE 3: INTEGRAÇÃO COMPLETA")
    print("="*80 + "\n")

    try:
        # Teste 1: Queries com extração
        await test_query_with_extraction()

        # Teste 2: Queries sem extração
        await test_query_without_extraction()

        # Teste 3: Comportamento de cache
        await test_cache_behavior()

        print("\n" + "="*80)
        print("✅ TODOS OS TESTES MANUAIS CONCLUÍDOS")
        print("="*80 + "\n")

    except Exception as e:
        log.error("test_failed", error=str(e))
        print(f"\n❌ ERRO: {e}\n")
        raise


if __name__ == "__main__":
    asyncio.run(main())
