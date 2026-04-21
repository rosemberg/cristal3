# Fixtures

Este diretório contém amostras reais extraídas do portal TRE-PI em 2026-04-20. Servem como **ground truth** para testes de extração, canonicalização e parsing — não são arquivos gerados pelo nosso crawler nem templates a copiar.

## Arquivos

### `_index.json`

Arquivo produzido pelo crawler anterior (schema v1) para a página raiz do escopo:

- **URL de origem**: `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas`
- **Schema**: v1 (formato antigo, anterior às decisões desta Fase)
- **Nome do arquivo**: `_index.json` — convenção do crawler anterior, que usa esse nome para marcar o nó raiz de cada subdiretório da árvore hierárquica
- **Uso esperado**:
  - Referência do formato anterior para migração de schema v1 → v2
  - Fixture para testes de retrocompatibilidade na leitura
  - Amostra de estrutura real de página Plone (breadcrumb, links, conteúdo)

**Nota importante**: o formato a ser produzido pelo novo crawler é o **schema v2**, definido no `BRIEF.md` na raiz do projeto. Não use este arquivo como template — use-o apenas como referência histórica e fixture de teste.

O crawler novo manterá a mesma convenção de nome (`_index.json`) dentro da árvore de saída, mas com o schema atualizado para v2.

### `sitemap.xml.gz`

Sitemap global do portal TRE-PI, comprimido conforme convenção sitemaps.org:

- **URL de origem**: `https://www.tre-pi.jus.br/sitemap.xml.gz`
- **Coleta em**: 2026-04-20
- **Conteúdo**: 2.367 URLs do portal completo; 573 URLs dentro do escopo (`/transparencia-e-prestacao-de-contas/*`)
- **Uso esperado**:
  - Fixture para testes do componente `discover` (parsing do sitemap, filtragem por prefixo, extração de `lastmod`)
  - Validação de que o crawler consegue descomprimir e parsear o formato real
  - Cálculo e verificação de estatísticas de volume (RF-01 do BRIEF)

## Política de Atualização

Estes fixtures são **snapshots temporais** — refletem o portal numa data específica. Não devem ser atualizados automaticamente pelo crawler nem por testes.

Para atualizar manualmente (ex: quando o portal sofrer reestruturação relevante):

1. Baixar novamente da fonte:
   ```
   curl -sL https://www.tre-pi.jus.br/sitemap.xml.gz -o fixtures/sitemap.xml.gz
   ```
2. Atualizar este README com a nova data de coleta.
3. Atualizar os testes que dependem dos fixtures se a estrutura tiver mudado.
4. Commit separado com mensagem clara: `chore(fixtures): atualiza amostra do portal TRE-PI`.

## Uso em Testes

Os testes do projeto devem referenciar estes arquivos por caminho relativo a partir da raiz:

```go
const fixturePage = "fixtures/_index.json"
const fixtureSitemap = "fixtures/sitemap.xml.gz"
```

Não duplicar os arquivos dentro de `testdata/` de pacotes — preferir referência única a partir de `fixtures/` para evitar drift entre cópias.

## Cuidado com Ambiguidade do Nome

Como o novo crawler também produzirá arquivos chamados `_index.json` (um por nó da árvore crawleada), há potencial de confusão entre:

- **`fixtures/_index.json`** — amostra estática, schema v1, nunca modificada
- **`data/<path>/_index.json`** — gerados pelo crawler novo, schema v2, dinâmicos

O diretório `fixtures/` é **read-only** do ponto de vista do crawler e dos testes. Nenhum código deve escrever lá.
