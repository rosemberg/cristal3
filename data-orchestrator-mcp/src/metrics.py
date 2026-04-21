"""
Módulo de métricas e observabilidade do Data Orchestrator

Rastreia operações, performance e uso do sistema.
"""

from datetime import datetime
from typing import Dict, Any
from dataclasses import dataclass, field
import threading

@dataclass
class Metrics:
    """Classe para rastreamento de métricas do sistema"""

    # Contadores
    cache_hits: int = 0
    cache_misses: int = 0
    extractions_success: int = 0
    extractions_failed: int = 0
    searches_performed: int = 0
    researches_performed: int = 0
    documents_fetched: int = 0
    errors_total: int = 0

    # Valores acumulados
    total_bytes_fetched: int = 0
    total_values_extracted: float = 0.0
    total_pages_processed: int = 0

    # Timing
    started_at: datetime = field(default_factory=datetime.now)

    # Thread safety
    _lock: threading.Lock = field(default_factory=threading.Lock)

    def increment_cache_hit(self):
        """Incrementa contador de cache hits"""
        with self._lock:
            self.cache_hits += 1

    def increment_cache_miss(self):
        """Incrementa contador de cache misses"""
        with self._lock:
            self.cache_misses += 1

    def increment_extraction_success(self):
        """Incrementa contador de extrações bem-sucedidas"""
        with self._lock:
            self.extractions_success += 1

    def increment_extraction_failed(self):
        """Incrementa contador de extrações falhas"""
        with self._lock:
            self.extractions_failed += 1

    def increment_search(self):
        """Incrementa contador de buscas"""
        with self._lock:
            self.searches_performed += 1

    def increment_research(self):
        """Incrementa contador de research (busca + extração)"""
        with self._lock:
            self.researches_performed += 1

    def increment_document_fetch(self):
        """Incrementa contador de documentos baixados"""
        with self._lock:
            self.documents_fetched += 1

    def increment_error(self):
        """Incrementa contador de erros"""
        with self._lock:
            self.errors_total += 1

    def add_bytes_fetched(self, bytes_count: int):
        """Adiciona bytes baixados ao total"""
        with self._lock:
            self.total_bytes_fetched += bytes_count

    def add_values_extracted(self, value: float):
        """Adiciona valor monetário extraído ao total"""
        with self._lock:
            self.total_values_extracted += value

    def add_pages_processed(self, pages: int):
        """Adiciona páginas processadas ao total"""
        with self._lock:
            self.total_pages_processed += pages

    @property
    def uptime_seconds(self) -> float:
        """Retorna uptime em segundos"""
        return (datetime.now() - self.started_at).total_seconds()

    @property
    def cache_hit_rate(self) -> float:
        """Calcula taxa de acerto do cache"""
        total = self.cache_hits + self.cache_misses
        if total == 0:
            return 0.0
        return (self.cache_hits / total) * 100

    @property
    def extraction_success_rate(self) -> float:
        """Calcula taxa de sucesso das extrações"""
        total = self.extractions_success + self.extractions_failed
        if total == 0:
            return 0.0
        return (self.extractions_success / total) * 100

    def get_summary(self) -> Dict[str, Any]:
        """Retorna resumo completo das métricas"""
        return {
            "uptime": {
                "seconds": self.uptime_seconds,
                "hours": self.uptime_seconds / 3600,
                "started_at": self.started_at.isoformat()
            },
            "cache": {
                "hits": self.cache_hits,
                "misses": self.cache_misses,
                "hit_rate": f"{self.cache_hit_rate:.2f}%"
            },
            "extractions": {
                "success": self.extractions_success,
                "failed": self.extractions_failed,
                "success_rate": f"{self.extraction_success_rate:.2f}%",
                "total_pages": self.total_pages_processed,
                "total_values": f"R$ {self.total_values_extracted:,.2f}"
            },
            "operations": {
                "searches": self.searches_performed,
                "researches": self.researches_performed,
                "documents_fetched": self.documents_fetched,
                "total_bytes": self.total_bytes_fetched,
                "total_mb": f"{self.total_bytes_fetched / 1024 / 1024:.2f} MB"
            },
            "errors": {
                "total": self.errors_total
            }
        }

    def reset(self):
        """Reseta todas as métricas"""
        with self._lock:
            self.cache_hits = 0
            self.cache_misses = 0
            self.extractions_success = 0
            self.extractions_failed = 0
            self.searches_performed = 0
            self.researches_performed = 0
            self.documents_fetched = 0
            self.errors_total = 0
            self.total_bytes_fetched = 0
            self.total_values_extracted = 0.0
            self.total_pages_processed = 0
            self.started_at = datetime.now()

# Instância global de métricas
_metrics = Metrics()

def get_metrics() -> Metrics:
    """Retorna instância global de métricas"""
    return _metrics
