import json
import hashlib
from pathlib import Path
from datetime import datetime, timedelta
from typing import Optional, List, Dict
import polars as pl
from .models import CacheEntry

class CacheManager:
    def __init__(self, cache_dir: str, ttl_queries: int, ttl_documents: int):
        self.cache_dir = Path(cache_dir)
        self.queries_dir = self.cache_dir / "queries"
        self.documents_dir = self.cache_dir / "documents"
        self.extracted_dir = self.cache_dir / "extracted"

        # Criar diretórios
        self.queries_dir.mkdir(parents=True, exist_ok=True)
        self.documents_dir.mkdir(parents=True, exist_ok=True)
        self.extracted_dir.mkdir(parents=True, exist_ok=True)

        self.ttl_queries = ttl_queries
        self.ttl_documents = ttl_documents

    def _hash_key(self, key: str) -> str:
        return hashlib.md5(key.encode()).hexdigest()

    def get_query(self, query: str) -> Optional[CacheEntry]:
        cache_file = self.queries_dir / f"{self._hash_key(query)}.json"
        if not cache_file.exists():
            return None

        data = json.loads(cache_file.read_text())
        entry = CacheEntry(**data)

        # Verificar TTL
        if datetime.now() - entry.timestamp > timedelta(seconds=self.ttl_queries):
            cache_file.unlink()  # Remove expirado
            return None

        return entry

    def set_query(self, query: str, summary: dict, data_file: Optional[str] = None):
        entry = CacheEntry(
            query=query,
            timestamp=datetime.now(),
            ttl=self.ttl_queries,
            summary=summary,
            data_file=data_file
        )

        cache_file = self.queries_dir / f"{self._hash_key(query)}.json"
        cache_file.write_text(entry.model_dump_json(indent=2))

    def get_document(self, url: str) -> Optional[dict]:
        cache_file = self.documents_dir / f"{self._hash_key(url)}.json"
        if not cache_file.exists():
            return None

        data = json.loads(cache_file.read_text())

        # Verificar TTL
        extracted_at = datetime.fromisoformat(data['extracted_at'])
        if datetime.now() - extracted_at > timedelta(seconds=self.ttl_documents):
            cache_file.unlink()
            return None

        return data

    def set_document(self, url: str, data: dict):
        metadata = {
            "url": url,
            "extracted_at": datetime.now().isoformat(),
            "data": data
        }

        cache_file = self.documents_dir / f"{self._hash_key(url)}.json"
        cache_file.write_text(json.dumps(metadata, indent=2))

    def save_parquet(self, data: List[Dict], filename: str) -> str:
        """Salva dados tabulares em Parquet"""
        if not data:
            raise ValueError("Cannot save empty data to parquet")

        df = pl.DataFrame(data)
        filepath = self.extracted_dir / f"{filename}.parquet"
        df.write_parquet(filepath)
        return str(filepath)

    def load_parquet(self, filename: str) -> pl.DataFrame:
        """Carrega dados de Parquet"""
        filepath = self.extracted_dir / f"{filename}.parquet"
        if not filepath.exists():
            raise FileNotFoundError(f"Parquet file not found: {filepath}")
        return pl.read_parquet(filepath)
