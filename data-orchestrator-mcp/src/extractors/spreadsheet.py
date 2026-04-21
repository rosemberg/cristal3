import polars as pl
from typing import Dict, Any
from io import BytesIO
from .base import BaseExtractor
from ..models import SourceMetadata, ExtractedData

class SpreadsheetExtractor(BaseExtractor):
    def can_handle(self, content_type: str, url: str) -> bool:
        return any(ext in url.lower() for ext in ['.csv', '.xlsx', '.xls'])

    async def extract(self, content: bytes, metadata: SourceMetadata) -> ExtractedData:
        """Extrai dados de CSV ou Excel"""

        try:
            # Detectar tipo
            if b'PK' in content[:4]:  # Excel (ZIP format)
                df = pl.read_excel(BytesIO(content))
            else:  # CSV
                df = pl.read_csv(BytesIO(content))

            # Calcular estatísticas básicas
            data = {
                "type": "spreadsheet",
                "rows": df.height,
                "columns": df.width,
                "column_names": df.columns
            }

            # Se houver coluna com valores, tentar somar
            valor_cols = [col for col in df.columns if 'valor' in col.lower()]
            if valor_cols:
                total = df[valor_cols[0]].sum()
                data["total"] = float(total)

            return ExtractedData(
                metadata=metadata,
                data=data,
                extraction_method="polars",
                success=True
            )

        except Exception as e:
            return ExtractedData(
                metadata=metadata,
                data={},
                extraction_method="polars",
                success=False,
                error=str(e)
            )
