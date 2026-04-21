import pytest
from datetime import datetime, timedelta
from pathlib import Path
import json
import tempfile
import shutil
from src.cache import CacheManager
from src.models import CacheEntry

@pytest.fixture
def temp_cache():
    """Fixture para criar cache temporário"""
    temp_dir = tempfile.mkdtemp()
    cache = CacheManager(
        cache_dir=temp_dir,
        ttl_queries=3600,
        ttl_documents=7200
    )
    yield cache
    # Cleanup
    shutil.rmtree(temp_dir)

def test_cache_set_and_get_query(temp_cache):
    """Testa set e get de query"""
    query = "teste query"
    summary = {"total": 1500.00, "count": 5}

    temp_cache.set_query(query, summary)

    result = temp_cache.get_query(query)
    assert result is not None
    assert result.query == query
    assert result.summary == summary

def test_cache_get_nonexistent_query(temp_cache):
    """Testa get de query inexistente"""
    result = temp_cache.get_query("query que nao existe")
    assert result is None

def test_cache_query_ttl_expired(temp_cache):
    """Testa expiração de TTL de query"""
    # Criar cache com TTL muito curto
    temp_dir = tempfile.mkdtemp()
    short_cache = CacheManager(
        cache_dir=temp_dir,
        ttl_queries=1,  # 1 segundo
        ttl_documents=3600
    )

    query = "teste ttl"
    summary = {"test": "data"}

    short_cache.set_query(query, summary)

    # Manipular timestamp para simular expiração
    cache_hash = short_cache._hash_key(query)
    cache_file = short_cache.queries_dir / f"{cache_hash}.json"

    # Ler e modificar timestamp
    data = json.loads(cache_file.read_text())
    old_time = datetime.now() - timedelta(seconds=10)
    data['timestamp'] = old_time.isoformat()
    cache_file.write_text(json.dumps(data))

    # Tentar recuperar - deve retornar None e deletar arquivo
    result = short_cache.get_query(query)
    assert result is None
    assert not cache_file.exists()

    shutil.rmtree(temp_dir)

def test_cache_set_and_get_document(temp_cache):
    """Testa set e get de documento"""
    url = "https://example.com/doc.pdf"
    data = {
        "type": "pdf",
        "pages": 10,
        "total": 5000.00,
        "valores": [1000.00, 2000.00, 2000.00]
    }

    temp_cache.set_document(url, data)

    result = temp_cache.get_document(url)
    assert result is not None
    assert result['url'] == url
    assert result['data'] == data

def test_cache_document_ttl_expired(temp_cache):
    """Testa expiração de TTL de documento"""
    temp_dir = tempfile.mkdtemp()
    short_cache = CacheManager(
        cache_dir=temp_dir,
        ttl_queries=3600,
        ttl_documents=1  # 1 segundo
    )

    url = "https://example.com/test.pdf"
    data = {"test": "data"}

    short_cache.set_document(url, data)

    # Manipular timestamp
    cache_hash = short_cache._hash_key(url)
    cache_file = short_cache.documents_dir / f"{cache_hash}.json"

    doc_data = json.loads(cache_file.read_text())
    old_time = datetime.now() - timedelta(seconds=10)
    doc_data['extracted_at'] = old_time.isoformat()
    cache_file.write_text(json.dumps(doc_data))

    result = short_cache.get_document(url)
    assert result is None
    assert not cache_file.exists()

    shutil.rmtree(temp_dir)

def test_cache_hash_consistency(temp_cache):
    """Testa consistência do hash"""
    key1 = "test query"
    key2 = "test query"
    key3 = "different query"

    hash1 = temp_cache._hash_key(key1)
    hash2 = temp_cache._hash_key(key2)
    hash3 = temp_cache._hash_key(key3)

    assert hash1 == hash2
    assert hash1 != hash3

def test_cache_save_and_load_parquet(temp_cache):
    """Testa salvamento e leitura de parquet"""
    data = [
        {"nome": "João", "valor": 1000.00, "mes": "janeiro"},
        {"nome": "Maria", "valor": 1500.00, "mes": "janeiro"},
        {"nome": "Pedro", "valor": 800.00, "mes": "fevereiro"}
    ]

    filepath = temp_cache.save_parquet(data, "test_data")
    assert Path(filepath).exists()

    df = temp_cache.load_parquet("test_data")
    assert len(df) == 3
    assert "nome" in df.columns
    assert "valor" in df.columns

def test_cache_parquet_empty_data(temp_cache):
    """Testa erro ao salvar dados vazios"""
    with pytest.raises(ValueError, match="Cannot save empty data"):
        temp_cache.save_parquet([], "empty")

def test_cache_parquet_file_not_found(temp_cache):
    """Testa erro ao carregar arquivo inexistente"""
    with pytest.raises(FileNotFoundError):
        temp_cache.load_parquet("nonexistent_file")

if __name__ == "__main__":
    pytest.main([__file__, "-v"])
