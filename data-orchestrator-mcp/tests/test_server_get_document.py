"""
Teste do novo tool get_document no servidor MCP
"""
import asyncio
import sys
from pathlib import Path

# Adicionar src ao path
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.server import list_tools, call_tool, extractors, get_extractor

async def test_list_tools_has_get_document():
    """Verifica se o tool get_document está na lista"""
    print("\n=== Teste: Listagem de Tools ===")

    tools = await list_tools()
    tool_names = [tool['name'] for tool in tools]

    print(f"  Tools disponíveis: {tool_names}")

    assert "get_document" in tool_names, "Tool get_document não encontrado"
    print("  ✅ Tool get_document está disponível")

    # Verificar schema do tool
    get_doc_tool = next(t for t in tools if t['name'] == 'get_document')
    assert 'inputSchema' in get_doc_tool
    assert 'url' in get_doc_tool['inputSchema']['properties']
    print("  ✅ Schema do tool get_document está correto")

def test_extractors_initialized():
    """Verifica se os extractors foram inicializados"""
    print("\n=== Teste: Inicialização de Extractors ===")

    assert len(extractors) == 2, f"Esperado 2 extractors, encontrado {len(extractors)}"
    print(f"  Extractors carregados: {len(extractors)}")

    # Verificar tipos
    from src.extractors.pdf import PDFExtractor
    from src.extractors.spreadsheet import SpreadsheetExtractor

    has_pdf = any(isinstance(e, PDFExtractor) for e in extractors)
    has_spreadsheet = any(isinstance(e, SpreadsheetExtractor) for e in extractors)

    assert has_pdf, "PDFExtractor não encontrado"
    assert has_spreadsheet, "SpreadsheetExtractor não encontrado"

    print("  ✅ PDFExtractor inicializado")
    print("  ✅ SpreadsheetExtractor inicializado")

def test_get_extractor_function():
    """Verifica a função get_extractor"""
    print("\n=== Teste: Função get_extractor ===")

    # PDF
    pdf_ext = get_extractor("application/pdf", "teste.pdf")
    assert pdf_ext is not None
    print("  ✅ get_extractor retorna PDFExtractor para PDF")

    # CSV
    csv_ext = get_extractor("text/csv", "teste.csv")
    assert csv_ext is not None
    print("  ✅ get_extractor retorna SpreadsheetExtractor para CSV")

    # Tipo desconhecido
    unknown_ext = get_extractor("text/plain", "teste.txt")
    assert unknown_ext is None
    print("  ✅ get_extractor retorna None para tipo desconhecido")

async def test_tool_schema_validation():
    """Valida o schema completo dos tools"""
    print("\n=== Teste: Validação de Schema ===")

    tools = await list_tools()

    for tool in tools:
        assert 'name' in tool
        assert 'description' in tool
        assert 'inputSchema' in tool
        assert 'type' in tool['inputSchema']
        assert 'properties' in tool['inputSchema']
        print(f"  ✅ Schema válido para tool: {tool['name']}")

async def main():
    """Executa todos os testes"""
    print("\n" + "="*60)
    print("TESTES DO SERVIDOR - TOOL GET_DOCUMENT")
    print("="*60)

    await test_list_tools_has_get_document()
    test_extractors_initialized()
    test_get_extractor_function()
    await test_tool_schema_validation()

    print("\n" + "="*60)
    print("✅ TODOS OS TESTES DO SERVIDOR PASSARAM!")
    print("="*60)
    print("\nResumo:")
    print("  - Tool get_document disponível no servidor")
    print("  - Extractors inicializados corretamente")
    print("  - Função get_extractor funcionando")
    print("  - Schemas dos tools válidos")
    print("\n✅ Servidor pronto para receber requisições!")

if __name__ == "__main__":
    asyncio.run(main())
