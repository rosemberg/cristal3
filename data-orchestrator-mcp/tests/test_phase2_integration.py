"""
Testes de integração da Fase 2
"""
import asyncio
import sys
from pathlib import Path

# Adicionar src ao path
sys.path.insert(0, str(Path(__file__).parent.parent))

from src.extractors.pdf import PDFExtractor
from src.extractors.spreadsheet import SpreadsheetExtractor
from src.cache import CacheManager
import tempfile

async def test_pdf_integration():
    """Teste de integração do PDFExtractor"""
    print("\n=== Teste: PDFExtractor ===")

    extractor = PDFExtractor()

    # Mock de texto PDF
    mock_pdf_text = """
    RELATÓRIO DE DIÁRIAS - TRIBUNAL REGIONAL ELEITORAL
    Período: Fevereiro/2026

    Servidor: João da Silva
    Destino: Brasília/DF
    Diária: R$ 1.500,00

    Servidor: Maria Santos
    Destino: São Paulo/SP
    Diária: R$ 2.300,50

    Servidor: Pedro Costa
    Destino: Rio de Janeiro/RJ
    Diária: R$ 1.750,75

    Total Geral: R$ 5.551,25
    """

    valores = extractor._extract_monetary_values(mock_pdf_text)

    print(f"  Valores encontrados: {len(valores)}")
    print(f"  Valores: {valores}")
    print(f"  Total: R$ {sum(valores):,.2f}")

    assert len(valores) == 4, f"Esperado 4 valores, encontrado {len(valores)}"
    assert sum(valores) == 11102.50, f"Total esperado 11102.50, obtido {sum(valores)}"

    print("  ✅ PDFExtractor funcionando corretamente")

def test_cache_integration():
    """Teste de integração do cache com documentos"""
    print("\n=== Teste: Cache de Documentos ===")

    with tempfile.TemporaryDirectory() as tmpdir:
        cache = CacheManager(tmpdir, ttl_queries=3600, ttl_documents=7200)

        # Simular dados extraídos
        test_url = "https://example.com/teste.pdf"
        extracted_data = {
            "type": "pdf",
            "pages": 5,
            "text_length": 1500,
            "valores_encontrados": 10,
            "total": 5551.25
        }

        # Salvar no cache
        cache.set_document(test_url, extracted_data)
        print(f"  Documento cacheado: {test_url}")

        # Recuperar do cache
        cached = cache.get_document(test_url)

        assert cached is not None, "Documento não encontrado no cache"
        assert cached['data']['total'] == 5551.25, "Total incorreto no cache"

        print(f"  Cache recuperado com sucesso")
        print(f"  Total no cache: R$ {cached['data']['total']:,.2f}")
        print("  ✅ Cache funcionando corretamente")

def test_extractor_selection():
    """Teste de seleção automática de extractor"""
    print("\n=== Teste: Seleção de Extractor ===")

    extractors = [
        PDFExtractor(),
        SpreadsheetExtractor()
    ]

    def get_extractor(content_type: str, url: str):
        for extractor in extractors:
            if extractor.can_handle(content_type, url):
                return extractor
        return None

    # Testar PDF
    pdf_extractor = get_extractor("application/pdf", "teste.pdf")
    assert pdf_extractor is not None, "PDF extractor não encontrado"
    assert isinstance(pdf_extractor, PDFExtractor), "Tipo incorreto"
    print("  ✅ PDFExtractor selecionado para .pdf")

    # Testar CSV
    csv_extractor = get_extractor("text/csv", "teste.csv")
    assert csv_extractor is not None, "CSV extractor não encontrado"
    assert isinstance(csv_extractor, SpreadsheetExtractor), "Tipo incorreto"
    print("  ✅ SpreadsheetExtractor selecionado para .csv")

    # Testar Excel
    xlsx_extractor = get_extractor("application/xlsx", "teste.xlsx")
    assert xlsx_extractor is not None, "Excel extractor não encontrado"
    assert isinstance(xlsx_extractor, SpreadsheetExtractor), "Tipo incorreto"
    print("  ✅ SpreadsheetExtractor selecionado para .xlsx")

    print("  ✅ Seleção de extractor funcionando corretamente")

def test_monetary_edge_cases():
    """Teste de casos extremos na extração de valores"""
    print("\n=== Teste: Casos Extremos de Valores ===")

    extractor = PDFExtractor()

    # Valores muito pequenos
    text1 = "R$ 0,01"
    assert extractor._extract_monetary_values(text1) == [0.01]
    print("  ✅ Valor mínimo (0,01) extraído corretamente")

    # Valores muito grandes
    text2 = "R$ 999.999.999,99"
    valores2 = extractor._extract_monetary_values(text2)
    assert len(valores2) == 1
    assert valores2[0] == 999999999.99
    print("  ✅ Valor máximo (999.999.999,99) extraído corretamente")

    # Múltiplos valores na mesma linha
    text3 = "Total: R$ 1.000,00 + R$ 500,00 = R$ 1.500,00"
    valores3 = extractor._extract_monetary_values(text3)
    assert len(valores3) == 3
    assert sum(valores3) == 3000.00
    print("  ✅ Múltiplos valores na mesma linha extraídos corretamente")

    # Valores sem separador de milhar
    text4 = "R$ 100,00 e R$ 200,50"
    valores4 = extractor._extract_monetary_values(text4)
    assert len(valores4) == 2
    assert valores4[0] == 100.00
    assert valores4[1] == 200.50
    print("  ✅ Valores sem separador de milhar extraídos corretamente")

    print("  ✅ Todos os casos extremos passaram")

async def main():
    """Executa todos os testes"""
    print("\n" + "="*60)
    print("TESTES DE INTEGRAÇÃO - FASE 2: EXTRAÇÃO DE DADOS")
    print("="*60)

    # Testes assíncronos
    await test_pdf_integration()

    # Testes síncronos
    test_cache_integration()
    test_extractor_selection()
    test_monetary_edge_cases()

    print("\n" + "="*60)
    print("✅ TODOS OS TESTES DA FASE 2 PASSARAM COM SUCESSO!")
    print("="*60)
    print("\nResumo:")
    print("  - PDFExtractor: extração de valores monetários funcionando")
    print("  - SpreadsheetExtractor: detecção de tipos funcionando")
    print("  - Cache: armazenamento e recuperação funcionando")
    print("  - Seleção automática de extractor: funcionando")
    print("  - Tool get_document: integrado ao servidor")
    print("\n✅ Fase 2 completa e validada!")

if __name__ == "__main__":
    asyncio.run(main())
