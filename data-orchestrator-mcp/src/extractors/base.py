from abc import ABC, abstractmethod
from typing import Dict, Any
from ..models import SourceMetadata, ExtractedData

class BaseExtractor(ABC):
    @abstractmethod
    async def extract(
        self,
        content: bytes,
        metadata: SourceMetadata  # 🔥 NOVO: metadados obrigatórios
    ) -> ExtractedData:
        """Extrai dados com metadados de rastreabilidade"""
        pass

    @abstractmethod
    def can_handle(self, content_type: str, url: str) -> bool:
        """Verifica se pode processar este tipo"""
        pass
