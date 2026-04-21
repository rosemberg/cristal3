# Próximos Passos - Data Orchestrator MCP

## Status Atual

O Data Orchestrator MCP está **100% completo** e **pronto para produção**. Todas as 4 fases foram concluídas com sucesso.

Este documento sugere melhorias opcionais para evolução futura do sistema.

---

## Melhorias Opcionais

### 1. Dashboard Web para Métricas

**Objetivo:** Visualizar métricas em tempo real via interface web

**Implementação:**
- FastAPI endpoint para servir métricas
- Frontend com gráficos (Chart.js ou Plotly)
- WebSocket para updates em tempo real

**Exemplo:**
```python
from fastapi import FastAPI
from fastapi.responses import HTMLResponse

app = FastAPI()

@app.get("/metrics")
async def get_metrics():
    from src.metrics import get_metrics
    return get_metrics().get_summary()

@app.get("/dashboard")
async def dashboard():
    return HTMLResponse("""
    <html>
        <head><title>Metrics Dashboard</title></head>
        <body>
            <h1>Data Orchestrator Metrics</h1>
            <div id="metrics"></div>
            <script>
                // Fetch e renderizar métricas
            </script>
        </body>
    </html>
    """)
```

**Esforço:** 4-6 horas

---

### 2. Alertas Automáticos

**Objetivo:** Notificar sobre erros ou degradação de performance

**Implementação:**
- Monitor de taxa de erros
- Alertas via email/Slack/Discord
- Thresholds configuráveis

**Exemplo:**
```python
class AlertManager:
    def __init__(self, error_threshold=10):
        self.error_threshold = error_threshold
        
    def check_alerts(self, metrics):
        if metrics.errors_total >= self.error_threshold:
            self.send_alert(
                "High error rate",
                f"{metrics.errors_total} errors detected"
            )
            
    def send_alert(self, title, message):
        # Enviar via webhook, email, etc.
        pass
```

**Esforço:** 2-3 horas

---

### 3. Exportação para Prometheus

**Objetivo:** Integrar com stack de observabilidade (Prometheus + Grafana)

**Implementação:**
- Endpoint /metrics em formato Prometheus
- Exportador de métricas

**Exemplo:**
```python
from prometheus_client import Counter, Histogram, Gauge, generate_latest

cache_hits = Counter('cache_hits_total', 'Total cache hits')
cache_misses = Counter('cache_misses_total', 'Total cache misses')
extraction_duration = Histogram('extraction_duration_seconds', 'Extraction duration')

@app.get("/metrics")
async def prometheus_metrics():
    return Response(generate_latest(), media_type="text/plain")
```

**Esforço:** 3-4 horas

---

### 4. Testes de Carga

**Objetivo:** Validar performance sob carga

**Implementação:**
- Locust ou k6 para testes de carga
- Benchmarks de performance
- Identificação de gargalos

**Exemplo:**
```python
from locust import HttpUser, task, between

class DataOrchestratorUser(HttpUser):
    wait_time = between(1, 3)
    
    @task
    def research_query(self):
        self.client.post("/research", json={
            "query": "gastos com diárias"
        })
```

**Esforço:** 2-3 horas

---

### 5. CI/CD Pipeline

**Objetivo:** Automatizar testes e deploy

**Implementação:**
- GitHub Actions ou GitLab CI
- Testes automáticos em PR
- Deploy automático em merge

**Exemplo (.github/workflows/test.yml):**
```yaml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-python@v4
        with:
          python-version: '3.10'
      - run: pip install -r requirements.txt
      - run: pytest tests/ -v
```

**Esforço:** 2-3 horas

---

### 6. Docker Containerização

**Objetivo:** Facilitar deploy e isolamento

**Implementação:**
- Dockerfile otimizado
- docker-compose.yml
- Multi-stage build

**Exemplo (Dockerfile):**
```dockerfile
FROM python:3.10-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY src/ ./src/
COPY config.yaml .

CMD ["python", "-m", "src.server"]
```

**docker-compose.yml:**
```yaml
version: '3.8'
services:
  data-orchestrator:
    build: .
    volumes:
      - ./cache:/app/cache
    environment:
      - LOG_LEVEL=INFO
```

**Esforço:** 1-2 horas

---

### 7. Documentação API Completa

**Objetivo:** OpenAPI/Swagger para a API MCP

**Implementação:**
- JSON Schema completo
- Exemplos de uso
- Swagger UI

**Exemplo:**
```python
# Gerar OpenAPI spec
spec = {
    "openapi": "3.0.0",
    "info": {
        "title": "Data Orchestrator MCP",
        "version": "1.0.0"
    },
    "paths": {
        "/research": {
            "post": {
                "summary": "Busca completa com extração",
                "requestBody": {
                    "content": {
                        "application/json": {
                            "schema": {
                                "type": "object",
                                "properties": {
                                    "query": {"type": "string"},
                                    "force_fetch": {"type": "boolean"}
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
```

**Esforço:** 3-4 horas

---

### 8. Suporte a Mais Tipos de Documentos

**Objetivo:** Expandir extractors

**Tipos sugeridos:**
- DOCX (Word)
- PPTX (PowerPoint)
- HTML/Markdown
- JSON estruturado
- XML

**Exemplo (DOCX):**
```python
from docx import Document
from .base import BaseExtractor

class DOCXExtractor(BaseExtractor):
    def can_handle(self, content_type: str, url: str) -> bool:
        return url.endswith('.docx')
        
    async def extract(self, content: bytes) -> dict:
        doc = Document(BytesIO(content))
        
        full_text = ""
        for para in doc.paragraphs:
            full_text += para.text + "\n"
            
        valores = self._extract_monetary_values(full_text)
        
        return {
            "type": "docx",
            "paragraphs": len(doc.paragraphs),
            "text": full_text,
            "valores": valores,
            "total": sum(valores)
        }
```

**Esforço:** 2-3 horas por tipo

---

### 9. Cache Distribuído

**Objetivo:** Suportar múltiplas instâncias

**Implementação:**
- Redis para cache compartilhado
- Invalidação coordenada
- Lock distribuído

**Exemplo:**
```python
import redis

class RedisCacheManager(CacheManager):
    def __init__(self, redis_url, ttl_queries, ttl_documents):
        self.redis = redis.from_url(redis_url)
        self.ttl_queries = ttl_queries
        self.ttl_documents = ttl_documents
        
    def get_query(self, query: str):
        key = f"query:{self._hash_key(query)}"
        data = self.redis.get(key)
        if data:
            return CacheEntry.model_validate_json(data)
        return None
        
    def set_query(self, query: str, summary: dict):
        key = f"query:{self._hash_key(query)}"
        entry = CacheEntry(query=query, summary=summary, ...)
        self.redis.setex(key, self.ttl_queries, entry.model_dump_json())
```

**Esforço:** 4-6 horas

---

### 10. Rate Limiting

**Objetivo:** Proteger contra abuso

**Implementação:**
- Limites por IP/usuário
- Sliding window
- Resposta 429 Too Many Requests

**Exemplo:**
```python
from collections import defaultdict
from datetime import datetime, timedelta

class RateLimiter:
    def __init__(self, max_requests=100, window_seconds=60):
        self.max_requests = max_requests
        self.window = timedelta(seconds=window_seconds)
        self.requests = defaultdict(list)
        
    def is_allowed(self, identifier: str) -> bool:
        now = datetime.now()
        
        # Limpar requisições antigas
        self.requests[identifier] = [
            ts for ts in self.requests[identifier]
            if now - ts < self.window
        ]
        
        # Verificar limite
        if len(self.requests[identifier]) >= self.max_requests:
            return False
            
        self.requests[identifier].append(now)
        return True
```

**Esforço:** 2-3 horas

---

## Priorização Sugerida

### Alta Prioridade
1. **Docker Containerização** - Deploy mais fácil
2. **CI/CD Pipeline** - Qualidade contínua
3. **Testes de Carga** - Validar performance

### Média Prioridade
4. **Dashboard Web** - Melhor visualização
5. **Alertas Automáticos** - Proatividade
6. **Documentação API** - Developer experience

### Baixa Prioridade
7. **Prometheus** - Se já usa stack
8. **Mais Extractors** - Conforme demanda
9. **Cache Distribuído** - Se escalar
10. **Rate Limiting** - Se público

---

## Roteiro Sugerido

### Mês 1: Produção Estável
- Monitorar métricas
- Coletar feedback
- Ajustes finos

### Mês 2: Observabilidade
- Dashboard web
- Alertas automáticos
- Testes de carga

### Mês 3: Escalabilidade
- Docker + CI/CD
- Cache distribuído (se necessário)
- Rate limiting (se necessário)

### Mês 4+: Expansão
- Novos extractors
- Integração Prometheus
- Features avançadas

---

## Conclusão

O sistema está **pronto para uso imediato**. As melhorias sugeridas são **opcionais** e devem ser implementadas conforme necessidade e prioridade do projeto.

**Recomendação:** Use o sistema em produção por algumas semanas antes de implementar melhorias, para identificar as necessidades reais.

---

**Projeto:** Data Orchestrator MCP  
**Versão Atual:** 1.0.0  
**Status:** Produção  
**Próxima Versão Sugerida:** 1.1.0 (com melhorias opcionais)
