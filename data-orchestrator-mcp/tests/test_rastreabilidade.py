import pytest
from datetime import datetime
from src.models import SourceMetadata, ExtractedData, ResearchResponse
from src.clients.site_research import SiteResearchClient
from src.server import _format_response


# Teste 1: Erro sem site-research
@pytest.mark.asyncio
async def test_erro_sem_site_research():
    """Testa que sistema falha explicitamente sem site-research configurado"""
    client = SiteResearchClient()

    with pytest.raises(NotImplementedError) as exc_info:
        await client.search("teste")

    error_msg = str(exc_info.value)
    assert "site-research MCP" in error_msg
    # A mensagem de erro pode vir de connect() ou search()
    assert ("não está implementado" in error_msg or "rastreabilidade" in error_msg)


# Teste 2: Metadados completos
@pytest.mark.asyncio
async def test_metadados_completos():
    """Testa que ExtractedData contém todos os metadados obrigatórios"""

    # Criar metadados
    metadata = SourceMetadata(
        url="https://www.tre-pi.jus.br/docs/diarias-2026.pdf",
        source_type="pdf",
        document_title="Diárias 2026",
        portal_section="Recursos Humanos",
        file_size=1024
    )

    # Criar dados extraídos
    extracted = ExtractedData(
        metadata=metadata,
        data={"total": 15000.50, "pages": 3},
        extraction_method="pypdf",
        success=True
    )

    # Validar campos obrigatórios
    assert extracted.metadata.url is not None
    assert str(extracted.metadata.url) == "https://www.tre-pi.jus.br/docs/diarias-2026.pdf"
    assert extracted.metadata.source_type == "pdf"
    assert extracted.metadata.extracted_at is not None
    assert isinstance(extracted.metadata.extracted_at, datetime)
    assert extracted.extraction_method == "pypdf"
    assert extracted.success is True


# Teste 3: Formatação com fontes
def test_formatacao_com_fontes():
    """Testa que _format_response inclui seção de fontes com URLs"""

    # Criar resposta com fontes
    metadata1 = SourceMetadata(
        url="https://www.tre-pi.jus.br/docs/diarias-jan-2026.pdf",
        source_type="pdf",
        document_title="Diárias Janeiro 2026",
        portal_section="Recursos Humanos"
    )

    metadata2 = SourceMetadata(
        url="https://www.tre-pi.jus.br/docs/diarias-fev-2026.pdf",
        source_type="pdf",
        document_title="Diárias Fevereiro 2026",
        portal_section="Recursos Humanos"
    )

    source1 = ExtractedData(
        metadata=metadata1,
        data={"total": 10000.00, "pages": 2},
        extraction_method="pypdf",
        success=True
    )

    source2 = ExtractedData(
        metadata=metadata2,
        data={"total": 5000.00, "pages": 1},
        extraction_method="pypdf",
        success=True
    )

    response = ResearchResponse(
        query="diárias 2026",
        total_sources=2,
        sources=[source1, source2],
        aggregated_data={"total": 15000.00, "count": 2}
    )

    # Formatar
    result = _format_response(response)

    # Validar estrutura
    assert "content" in result
    assert len(result["content"]) == 1
    assert result["content"][0]["type"] == "text"

    text = result["content"][0]["text"]

    # Validar conteúdo
    assert "Resultados: diárias 2026" in text
    assert "📄 Fontes de Dados (2)" in text
    assert "https://www.tre-pi.jus.br/docs/diarias-jan-2026.pdf" in text
    assert "https://www.tre-pi.jus.br/docs/diarias-fev-2026.pdf" in text
    assert "Diárias Janeiro 2026" in text
    assert "Diárias Fevereiro 2026" in text
    assert "Recursos Humanos" in text
    assert "🔗 URL:" in text
    assert "📁 Tipo:" in text
    assert "📂 Seção:" in text
    assert "🕐 Extraído em:" in text
    assert "💰 Valor:" in text
    assert "📌 Rastreabilidade" in text


# Teste 4: Timestamp de consulta
def test_timestamp_consulta():
    """Testa que ResearchResponse tem timestamp de consulta"""

    response = ResearchResponse(
        query="teste",
        total_sources=0,
        sources=[]
    )

    assert response.search_timestamp is not None
    assert isinstance(response.search_timestamp, datetime)


# Teste 5: Cache hit flag
def test_cache_hit_flag():
    """Testa que cache_hit flag é corretamente setado"""

    # Cache hit
    response_cached = ResearchResponse(
        query="teste cache",
        total_sources=0,
        sources=[],
        cache_hit=True
    )

    result = _format_response(response_cached)
    text = result["content"][0]["text"]

    assert "💾 **Fonte:** Cache local" in text


# Teste 6: Alerta sem fontes
def test_alerta_sem_fontes():
    """Testa que sistema alerta quando não há fontes identificadas"""

    response = ResearchResponse(
        query="teste sem fontes",
        total_sources=0,
        sources=[]
    )

    result = _format_response(response)
    text = result["content"][0]["text"]

    assert "⚠️ **ALERTA:** Nenhuma fonte identificada!" in text


# Teste 7: Erro de extração
def test_erro_extracao():
    """Testa que erros de extração são mostrados corretamente"""

    metadata = SourceMetadata(
        url="https://www.tre-pi.jus.br/docs/corrupted.pdf",
        source_type="pdf",
        document_title="Documento Corrompido"
    )

    source = ExtractedData(
        metadata=metadata,
        data={},
        extraction_method="pypdf",
        success=False,
        error="Arquivo PDF corrompido"
    )

    response = ResearchResponse(
        query="teste erro",
        total_sources=1,
        sources=[source]
    )

    result = _format_response(response)
    text = result["content"][0]["text"]

    # Verificar que o erro está presente (com ou sem emoji no início)
    assert "Erro:** Arquivo PDF corrompido" in text or "Arquivo PDF corrompido" in text


# Teste 8: Validação de HttpUrl
def test_validacao_httpurl():
    """Testa que apenas URLs válidas são aceitas"""

    # URL válida
    metadata_valid = SourceMetadata(
        url="https://www.tre-pi.jus.br/docs/test.pdf",
        source_type="pdf"
    )
    assert str(metadata_valid.url).startswith("https://")

    # URL inválida deve falhar
    with pytest.raises(Exception):  # Pydantic validation error
        SourceMetadata(
            url="not-a-valid-url",
            source_type="pdf"
        )
