# Fixtures HTML — Amostras Reais do Portal TRE-PI

Este diretório contém páginas HTML reais do portal TRE-PI coletadas em **2026-04-20**. Servem como **ground truth** para o extrator de HTML (Task 13) e para o classificador de páginas (Task 14).

## Finalidade

Cada arquivo é um snapshot fiel de uma página real, obtida com `curl -sL` diretamente do portal. Os arquivos cobrem layouts distintos (raiz de escopo, seção, artigo, listagem de documentos, artigo profundo, página mínima) para que os testes exercitem variações reais de estrutura Plone.

## Data de Coleta

2026-04-20

## Arquivos

| Arquivo | URL de Origem | Motivo da Escolha |
|---|---|---|
| `landing_root.html` | `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas` | Raiz do escopo; layout Plone de portal com menu-pesado e pouco conteúdo textual próprio. |
| `landing_section.html` | `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/gestao-orcamentaria-e-financeira` | Seção de primeiro nível (gestão orçamentária); layout de landing de subseção com links para subpáginas. |
| `article.html` | `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/licitacoes-e-contratos/licitacoes-e-contratos` | Página com conteúdo textual substantivo sobre licitações; exercita extração de prosa e subtítulos. |
| `listing_with_docs.html` | `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/licitacoes-e-contratos/outras-contratacoes/termo-de-repasse/arquivos` | Pasta de arquivos (`template-listing_view`, `portaltype-pastaarquivos`); conteúdo principal é lista de documentos para download. |
| `deep_article.html` | `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/colegiados/comissoes-permanentes-e-tecnicas/comissao-setorial-de-risco-saof/comissao-setorial-de-risco-saof` | Página com 5 segmentos de caminho (≥ 4 exigidos); artigo de colegiado interno; exercita profundidade de breadcrumb. |
| `empty_or_minimal.html` | `https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas/gestao-orcamentaria-e-financeira/copy2_of_relatorios-do-conselho-nacional-de-justica-cnj` | Cópia de página (`copy2_of_`) com pouco conteúdo próprio; útil para testar classificação de páginas quase-vazias. |

## Observações sobre a Estrutura Plone

Todas as páginas são geradas pelo Plone e compartilham:

- Meta tag `<meta name="generator" content="Plone - https://plone.org/">`.
- Breadcrumb identificável pelos seletores `breadcrumb` e `portal-breadcrumbs`.
- Indicação de tipo de página na classe do `<body>` (ex: `portaltype-paginainterna`, `portaltype-pastaarquivos`).
- Template utilizado também na classe do `<body>` (ex: `template-view`, `template-listing_view`).

## Política de Atualização

Estes arquivos são **snapshots estáticos** — não devem ser atualizados automaticamente pelo crawler nem por testes. Para refrescar manualmente:

```bash
# Exemplo para landing_root.html:
curl -sL https://www.tre-pi.jus.br/transparencia-e-prestacao-de-contas \
  -o fixtures/html/landing_root.html

# Repita para cada arquivo usando a URL documentada na tabela acima.
# Espaçe as chamadas com ~1 segundo entre elas para não sobrecarregar o portal.
sleep 1
```

Após atualizar, faça commit separado com mensagem:
`chore(fixtures): atualiza amostras HTML do portal TRE-PI`.

## Uso em Testes

```go
const fixtureRoot    = "fixtures/html/landing_root.html"
const fixtureSection = "fixtures/html/landing_section.html"
const fixtureArticle = "fixtures/html/article.html"
const fixtureListing = "fixtures/html/listing_with_docs.html"
const fixtureDeep    = "fixtures/html/deep_article.html"
const fixtureEmpty   = "fixtures/html/empty_or_minimal.html"
```

Referenciar sempre por caminho relativo à raiz do projeto. Não duplicar os arquivos em `testdata/` de pacotes individuais.
