import pytest
import asyncio
from src.extractors.pdf import PDFExtractor
from src.extractors.spreadsheet import SpreadsheetExtractor

# Test PDFExtractor - Monetary Values Extraction
@pytest.mark.asyncio
async def test_pdf_extractor_monetary_values():
    """Testa extração de valores monetários de texto mock"""
    extractor = PDFExtractor()

    # Texto mock com valores brasileiros
    mock_text = """
    Relatório de Diárias - Fevereiro 2026

    João Silva - R$ 1.234,56
    Maria Santos - R$ 2.500,00
    Pedro Oliveira - R$ 750,50
    Ana Costa - R$ 10.000,75

    Total: R$ 14.485,81
    """

    valores = extractor._extract_monetary_values(mock_text)

    assert len(valores) == 5
    assert valores[0] == 1234.56
    assert valores[1] == 2500.00
    assert valores[2] == 750.50
    assert valores[3] == 10000.75
    assert valores[4] == 14485.81
    assert sum(valores) == 28971.62

def test_pdf_can_handle():
    """Testa detecção de arquivos PDF"""
    extractor = PDFExtractor()

    assert extractor.can_handle("application/pdf", "teste.pdf") == True
    assert extractor.can_handle("application/pdf", "teste.xlsx") == True
    assert extractor.can_handle("text/csv", "teste.pdf") == True
    assert extractor.can_handle("text/csv", "teste.csv") == False

def test_spreadsheet_can_handle():
    """Testa detecção de arquivos CSV/Excel"""
    extractor = SpreadsheetExtractor()

    assert extractor.can_handle("text/csv", "teste.csv") == True
    assert extractor.can_handle("application/xlsx", "teste.xlsx") == True
    assert extractor.can_handle("application/xls", "teste.xls") == True
    assert extractor.can_handle("application/pdf", "teste.pdf") == False

def test_monetary_values_formats():
    """Testa diferentes formatos de valores monetários"""
    extractor = PDFExtractor()

    # Valores com diferentes formatações
    text1 = "R$ 100,50"
    assert extractor._extract_monetary_values(text1) == [100.50]

    text2 = "R$ 1.000,00"
    assert extractor._extract_monetary_values(text2) == [1000.00]

    text3 = "R$ 123.456,78"
    assert extractor._extract_monetary_values(text3) == [123456.78]

    # Valores sem separador de milhar
    text4 = "R$ 999,99"
    assert extractor._extract_monetary_values(text4) == [999.99]

if __name__ == "__main__":
    # Executar teste síncrono
    test_pdf_can_handle()
    test_spreadsheet_can_handle()
    test_monetary_values_formats()

    # Executar teste assíncrono
    asyncio.run(test_pdf_extractor_monetary_values())

    print("\n✅ Todos os testes de extractors passaram!")
