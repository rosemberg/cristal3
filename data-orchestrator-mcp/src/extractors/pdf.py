import re
from typing import Dict, Any, List
from pypdf import PdfReader
from io import BytesIO
from .base import BaseExtractor
from ..models import SourceMetadata, ExtractedData

class PDFExtractor(BaseExtractor):
    def can_handle(self, content_type: str, url: str) -> bool:
        return 'pdf' in content_type.lower() or url.endswith('.pdf')

    async def extract(self, content: bytes, metadata: SourceMetadata) -> ExtractedData:
        """Extrai texto e valores monetários de PDF"""

        try:
            pdf = PdfReader(BytesIO(content))

            full_text = ""
            for page in pdf.pages:
                full_text += page.extract_text() + "\n"

            # Extrair valores monetários (formato brasileiro: 1.234,56)
            valores = self._extract_monetary_values(full_text)

            data = {
                "type": "pdf",
                "pages": len(pdf.pages),
                "text_length": len(full_text),
                "text": full_text[:1000],  # Primeiros 1000 chars
                "valores_encontrados": len(valores),
                "valores": valores,
                "total": sum(valores) if valores else 0
            }

            return ExtractedData(
                metadata=metadata,
                data=data,
                extraction_method="pypdf",
                success=True
            )

        except Exception as e:
            return ExtractedData(
                metadata=metadata,
                data={},
                extraction_method="pypdf",
                success=False,
                error=str(e)
            )

    def _extract_monetary_values(self, text: str) -> List[float]:
        """Extrai valores monetários do texto"""
        # Pattern: 1.234,56 ou 234,56
        pattern = r'\b\d{1,3}(?:\.\d{3})*,\d{2}\b'
        matches = re.findall(pattern, text)

        valores = []
        for match in matches:
            # Converter formato brasileiro para float
            valor_float = float(match.replace('.', '').replace(',', '.'))
            valores.append(valor_float)

        return valores
