"""
Testes de integração completa da Fase 3

Valida:
- Detecção automática de necessidade de extração
- Extração automática de documentos
- Agregação de valores correta
- Formato de resposta adequado
"""

import pytest
import asyncio
import sys
from pathlib import Path
from io import BytesIO
from pypdf import PdfWriter, PdfReader

# Adicionar src ao path
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.server import research, _needs_detailed_data, _format_response
from src.models import ResearchResponse, SourceMetadata, ExtractedData
from src.cache import CacheManager
from src.clients.site_research import SiteResearchClient
from src.clients.http import HTTPClient
from src.extractors.pdf import PDFExtractor
import tempfile


@pytest.fixture
def temp_cache():
    """Cache temporário para testes"""
    with tempfile.TemporaryDirectory() as tmpdir:
        yield CacheManager(tmpdir, ttl_queries=3600, ttl_documents=7200)


class TestNeedsDetailedData:
    """Testa detecção de necessidade de dados detalhados"""

    def test_detect_quanto_keyword(self):
        """Deve detectar keyword 'quanto'"""
        query = "quanto foi gasto em diárias"
        results = []
        assert _needs_detailed_data(query, results) is True

    def test_detect_valor_keyword(self):
        """Deve detectar keyword 'valor'"""
        query = "qual o valor total em passagens"
        results = []
        assert _needs_detailed_data(query, results) is True

    def test_detect_total_keyword(self):
        """Deve detectar keyword 'total'"""
        query = "total de gastos em 2026"
        results = []
        assert _needs_detailed_data(query, results) is True

    def test_detect_gasto_keyword(self):
        """Deve detectar keyword 'gasto'"""
        query = "gastos com diárias"
        results = []
        assert _needs_detailed_data(query, results) is True

    def test_no_detection_for_generic_query(self):
        """Não deve detectar para query genérica"""
        query = "diárias e passagens"
        results = []
        assert _needs_detailed_data(query, results) is False

    def test_case_insensitive(self):
        """Deve ser case-insensitive"""
        query = "QUANTO FOI O GASTO"
        results = []
        assert _needs_detailed_data(query, results) is True


# DEPRECATED: TestAggregateResults
# A função _aggregate_results foi removida na refatoração de rastreabilidade.
# A agregação agora está integrada na função research() e usa ResearchResponse.
# Para testes de agregação, veja tests/test_rastreabilidade.py


class TestFormatResponse:
    """Testa formatação de resposta MCP"""

    def test_format_basic_response(self):
        """Deve formatar resposta básica"""
        research_response = ResearchResponse(
            query="test query",
            total_sources=0,
            sources=[]
        )

        response = _format_response(research_response)

        assert "content" in response
        assert len(response["content"]) == 1
        assert response["content"][0]["type"] == "text"

        text = response["content"][0]["text"]
        assert "test query" in text
        assert "Fontes de Dados (0)" in text

    def test_format_response_with_totals(self):
        """Deve formatar resposta com totais"""
        metadata1 = SourceMetadata(
            url="https://example.com/1",
            source_type="pdf"
        )
        source1 = ExtractedData(
            metadata=metadata1,
            data={"total": 1500.75},
            extraction_method="pypdf",
            success=True
        )
        metadata2 = SourceMetadata(
            url="https://example.com/2",
            source_type="pdf"
        )
        source2 = ExtractedData(
            metadata=metadata2,
            data={"total": 2300.50},
            extraction_method="pypdf",
            success=True
        )

        research_response = ResearchResponse(
            query="gastos",
            total_sources=2,
            sources=[source1, source2],
            aggregated_data={"total": 3801.25, "count": 25}
        )

        response = _format_response(research_response)
        text = response["content"][0]["text"]

        assert "Total:** R$ 3,801.25" in text or "Total:** R$ 3.801,25" in text
        assert "Registros:** 25" in text
        assert "Fontes de Dados (2)" in text

    def test_format_shows_urls(self):
        """Deve mostrar URLs das fontes"""
        sources = []
        for i in range(3):
            metadata = SourceMetadata(
                url=f"https://example.com/{i}",
                source_type="pdf"
            )
            source = ExtractedData(
                metadata=metadata,
                data={},
                extraction_method="pypdf",
                success=True
            )
            sources.append(source)

        research_response = ResearchResponse(
            query="test",
            total_sources=3,
            sources=sources
        )

        response = _format_response(research_response)
        text = response["content"][0]["text"]

        # Todas as URLs devem aparecer
        for i in range(3):
            assert f"https://example.com/{i}" in text


class TestCacheParquet:
    """Testa salvamento e carregamento de Parquet"""

    def test_save_and_load_parquet(self, temp_cache):
        """Deve salvar e carregar dados em Parquet"""
        data = [
            {"nome": "João", "valor": 1500.50, "mes": "janeiro"},
            {"nome": "Maria", "valor": 2300.75, "mes": "fevereiro"}
        ]

        # Salvar
        filepath = temp_cache.save_parquet(data, "test_data")
        assert Path(filepath).exists()

        # Carregar
        df = temp_cache.load_parquet("test_data")
        assert df.height == 2
        assert df.width == 3
        assert "nome" in df.columns
        assert "valor" in df.columns

    def test_save_empty_data_raises_error(self, temp_cache):
        """Deve lançar erro ao salvar dados vazios"""
        with pytest.raises(ValueError, match="Cannot save empty data"):
            temp_cache.save_parquet([], "empty")

    def test_load_nonexistent_file_raises_error(self, temp_cache):
        """Deve lançar erro ao carregar arquivo inexistente"""
        with pytest.raises(FileNotFoundError):
            temp_cache.load_parquet("nonexistent")


class TestMockSiteResearchClient:
    """Testa cliente do site-research"""

    @pytest.mark.asyncio
    async def test_search_raises_not_implemented(self):
        """Cliente deve lançar NotImplementedError quando não configurado"""
        client = SiteResearchClient()

        with pytest.raises(NotImplementedError) as exc_info:
            await client.search("diárias", limit=3)

        assert "site-research MCP" in str(exc_info.value)


def create_test_pdf_with_values():
    """Cria PDF de teste com valores monetários"""
    # Criar PDF simples usando pypdf
    writer = PdfWriter()

    # Nota: pypdf não suporta criação de conteúdo de texto facilmente
    # Este é um placeholder - em testes reais, usaríamos PDFs existentes
    # ou uma biblioteca como reportlab (que não está nas dependências)

    # Para os testes, vamos apenas retornar bytes vazios
    # Os testes principais não dependem desta função
    buffer = BytesIO()
    return buffer.getvalue()


class TestEndToEndIntegration:
    """Testes end-to-end da integração completa"""

    @pytest.mark.asyncio
    async def test_research_fails_without_site_research(self, temp_cache):
        """Teste end-to-end: research deve falhar sem site-research configurado"""

        # Preparar ambiente
        from src import server
        original_cache = server.cache
        server.cache = temp_cache

        try:
            # Executar research - deve retornar erro, não lançar exceção
            result = await research(query="quanto foi gasto em diárias")

            # Validar resposta
            assert "content" in result
            assert len(result["content"]) > 0
            text = result["content"][0]["text"]

            # Deve conter mensagem de erro sobre site-research não estar configurado
            assert "ERRO CRÍTICO" in text or "site-research" in text

        finally:
            server.cache = original_cache

    # NOTA: Testes de integração real com site-research funcionando
    # devem ser feitos manualmente ou em ambiente de staging com
    # site-research MCP configurado corretamente.


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
