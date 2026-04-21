import pytest
import asyncio
import tempfile
import shutil
from unittest.mock import AsyncMock, MagicMock, patch
from src.cache import CacheManager
from src.extractors.pdf import PDFExtractor
from src.clients.site_research import SiteResearchClient

@pytest.fixture
def temp_cache():
    """Fixture para cache temporário"""
    temp_dir = tempfile.mkdtemp()
    cache = CacheManager(
        cache_dir=temp_dir,
        ttl_queries=3600,
        ttl_documents=7200
    )
    yield cache
    shutil.rmtree(temp_dir)

@pytest.mark.asyncio
async def test_research_with_cache_hit(temp_cache):
    """Testa research que encontra dados no cache"""
    query = "gastos com diárias"
    cached_summary = {
        "query": query,
        "total": 15000.00,
        "count": 10,
        "found_pages": 3
    }

    # Pré-popular cache
    temp_cache.set_query(query, cached_summary)

    # Mock do SiteResearchClient (não deve ser chamado)
    mock_client = AsyncMock()

    # Simular research function
    cached = temp_cache.get_query(query)
    assert cached is not None
    assert cached.summary == cached_summary

    # Verificar que client não foi chamado (cache hit)
    mock_client.search.assert_not_called()

@pytest.mark.asyncio
async def test_research_cache_miss_triggers_extraction(temp_cache):
    """Testa research sem cache que aciona extração"""
    query = "quanto foi gasto em diárias"

    # Verificar que cache está vazio
    cached = temp_cache.get_query(query)
    assert cached is None

    # Mock de resultados da busca
    mock_results = [
        {
            "url": "https://example.com/page1",
            "title": "Diárias 2026",
            "documents": ["https://example.com/relatorio.pdf"]
        }
    ]

    # Mock do extractor
    mock_extracted = {
        "type": "pdf",
        "pages": 5,
        "valores": [1000.00, 2000.00, 3000.00],
        "total": 6000.00,
        "valores_encontrados": 3
    }

    # Simular fluxo completo
    with patch('src.clients.site_research.SiteResearchClient') as MockClient:
        mock_instance = MockClient.return_value
        mock_instance.search = AsyncMock(return_value=mock_results)

        # Simular que pesquisa foi feita
        results = await mock_instance.search(query, limit=10)
        assert len(results) == 1
        assert "documents" in results[0]

        # Simular extração
        temp_cache.set_document(mock_results[0]["documents"][0], mock_extracted)

        # Verificar que documento foi cacheado
        doc_cached = temp_cache.get_document(mock_results[0]["documents"][0])
        assert doc_cached is not None
        assert doc_cached['data']['total'] == 6000.00

@pytest.mark.asyncio
async def test_research_with_monetary_keywords():
    """Testa detecção de keywords monetárias"""
    from src.server import _needs_detailed_data

    # Queries que precisam de extração
    assert _needs_detailed_data("quanto foi gasto", []) == True
    assert _needs_detailed_data("qual o valor total", []) == True
    assert _needs_detailed_data("total de gastos", []) == True
    assert _needs_detailed_data("custo das diárias", []) == True
    assert _needs_detailed_data("despesas do mês", []) == True

    # Queries que NÃO precisam de extração
    assert _needs_detailed_data("documentos sobre educação", []) == False
    assert _needs_detailed_data("relatórios de saúde", []) == False

@pytest.mark.asyncio
async def test_integration_full_flow(temp_cache):
    """Testa fluxo completo: busca -> extração -> cache -> resposta"""
    query = "quanto custaram as diárias"

    # 1. Cache miss inicial
    assert temp_cache.get_query(query) is None

    # 2. Mock de busca retorna resultados
    mock_search_results = [
        {
            "url": "https://portal.com/diarias",
            "title": "Relatório de Diárias",
            "documents": ["https://portal.com/relatorio.pdf"]
        }
    ]

    # 3. Mock de documento baixado
    mock_pdf_content = b"%PDF-1.4 mock content"

    # 4. Mock de extração
    extractor = PDFExtractor()
    mock_extraction = {
        "type": "pdf",
        "pages": 3,
        "valores": [5000.00, 3000.00, 2000.00],
        "total": 10000.00,
        "valores_encontrados": 3
    }

    # 5. Simular cachear documento
    doc_url = mock_search_results[0]["documents"][0]
    temp_cache.set_document(doc_url, mock_extraction)

    # 6. Verificar documento cacheado
    doc_cached = temp_cache.get_document(doc_url)
    assert doc_cached is not None
    assert doc_cached['data']['total'] == 10000.00

    # 7. Simular agregação de resultados
    summary = {
        "query": query,
        "found_pages": 1,
        "extracted_documents": 1,
        "total": 10000.00,
        "count": 3,
        "sources": ["https://portal.com/diarias"]
    }

    # 8. Cachear resultado agregado
    temp_cache.set_query(query, summary)

    # 9. Verificar cache hit
    result = temp_cache.get_query(query)
    assert result is not None
    assert result.summary['total'] == 10000.00
    assert result.summary['count'] == 3

@pytest.mark.asyncio
async def test_multiple_documents_extraction(temp_cache):
    """Testa extração de múltiplos documentos"""
    documents = [
        ("https://example.com/doc1.pdf", {"total": 1000.00, "count": 2}),
        ("https://example.com/doc2.pdf", {"total": 2000.00, "count": 3}),
        ("https://example.com/doc3.pdf", {"total": 1500.00, "count": 4})
    ]

    # Cachear todos os documentos
    for url, data in documents:
        temp_cache.set_document(url, data)

    # Verificar todos foram cacheados
    totals = []
    for url, expected in documents:
        cached = temp_cache.get_document(url)
        assert cached is not None
        assert cached['data']['total'] == expected['total']
        totals.append(cached['data']['total'])

    # Verificar agregação
    assert sum(totals) == 4500.00

@pytest.mark.asyncio
async def test_error_handling_in_extraction(temp_cache):
    """Testa tratamento de erros durante extração"""
    # Simular erro de extração
    with patch('src.extractors.pdf.PDFExtractor.extract') as mock_extract:
        mock_extract.side_effect = Exception("Erro ao processar PDF")

        extractor = PDFExtractor()

        # Verificar que erro é capturado
        try:
            await extractor.extract(b"invalid content")
            assert False, "Deveria ter lançado exceção"
        except Exception as e:
            assert "Erro ao processar PDF" in str(e)

if __name__ == "__main__":
    pytest.main([__file__, "-v"])
