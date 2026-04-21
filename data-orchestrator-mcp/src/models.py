from pydantic import BaseModel, Field, HttpUrl
from typing import Optional, Dict, Any, List
from datetime import datetime

class CacheEntry(BaseModel):
    query: str
    timestamp: datetime
    ttl: int
    summary: Dict[str, Any]
    data_file: Optional[str] = None

class DocumentMetadata(BaseModel):
    url: str
    type: str
    extracted_at: datetime
    data: Dict[str, Any]

class SourceMetadata(BaseModel):
    """Metadados obrigatórios de fonte de dados"""

    url: HttpUrl                                # 🔥 OBRIGATÓRIO
    source_type: str                            # "pdf", "excel", "csv", "html"
    extracted_at: datetime = Field(default_factory=datetime.now)
    document_title: Optional[str] = None
    document_date: Optional[str] = None
    portal_section: Optional[str] = None
    file_size: Optional[int] = None
    checksum: Optional[str] = None              # MD5 do documento

class ExtractedData(BaseModel):
    """Dados extraídos com rastreabilidade completa"""

    metadata: SourceMetadata                    # 🔥 OBRIGATÓRIO
    data: Dict[str, Any]                        # Dados extraídos
    extraction_method: str                      # "pypdf", "polars", etc.
    success: bool = True
    error: Optional[str] = None

class ResearchResponse(BaseModel):
    query: str
    search_timestamp: datetime = Field(default_factory=datetime.now)
    total_sources: int
    sources: List[ExtractedData]                # 🔥 Com metadados completos
    aggregated_data: Optional[Dict[str, Any]] = None
    cache_hit: bool = False
