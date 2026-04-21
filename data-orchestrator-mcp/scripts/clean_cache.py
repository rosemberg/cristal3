#!/usr/bin/env python3
"""
Script para limpeza de cache do Data Orchestrator

Uso:
    python scripts/clean_cache.py --mode all       # Limpa todo o cache
    python scripts/clean_cache.py --mode expired   # Limpa apenas expirados
    python scripts/clean_cache.py --dry-run        # Simula sem deletar
"""

import argparse
import sys
from pathlib import Path
from datetime import datetime, timedelta
import json
import yaml

def load_config():
    """Carrega configuração do projeto"""
    config_path = Path(__file__).parent.parent / "config.yaml"
    if not config_path.exists():
        print(f"Erro: arquivo de configuração não encontrado: {config_path}")
        sys.exit(1)

    with open(config_path) as f:
        return yaml.safe_load(f)

def clean_all(cache_dir: Path, dry_run: bool = False):
    """Limpa todo o cache"""
    queries_dir = cache_dir / "queries"
    documents_dir = cache_dir / "documents"
    extracted_dir = cache_dir / "extracted"

    total_removed = 0

    for directory in [queries_dir, documents_dir, extracted_dir]:
        if not directory.exists():
            continue

        files = list(directory.glob("*"))
        print(f"\n📁 {directory.name}/")
        print(f"   Arquivos encontrados: {len(files)}")

        for file in files:
            if dry_run:
                print(f"   [DRY-RUN] Removeria: {file.name}")
            else:
                file.unlink()
                print(f"   ✓ Removido: {file.name}")

            total_removed += 1

    return total_removed

def clean_expired(cache_dir: Path, ttl_queries: int, ttl_documents: int, dry_run: bool = False):
    """Limpa apenas cache expirado"""
    queries_dir = cache_dir / "queries"
    documents_dir = cache_dir / "documents"

    total_removed = 0
    now = datetime.now()

    # Limpar queries expiradas
    if queries_dir.exists():
        print(f"\n📁 queries/")
        query_files = list(queries_dir.glob("*.json"))
        print(f"   Arquivos encontrados: {len(query_files)}")

        for file in query_files:
            try:
                data = json.loads(file.read_text())
                timestamp = datetime.fromisoformat(data['timestamp'])
                age = now - timestamp

                if age > timedelta(seconds=ttl_queries):
                    age_hours = age.total_seconds() / 3600
                    if dry_run:
                        print(f"   [DRY-RUN] Removeria: {file.name} (idade: {age_hours:.1f}h)")
                    else:
                        file.unlink()
                        print(f"   ✓ Removido: {file.name} (idade: {age_hours:.1f}h)")
                    total_removed += 1

            except Exception as e:
                print(f"   ⚠ Erro ao processar {file.name}: {e}")

    # Limpar documentos expirados
    if documents_dir.exists():
        print(f"\n📁 documents/")
        doc_files = list(documents_dir.glob("*.json"))
        print(f"   Arquivos encontrados: {len(doc_files)}")

        for file in doc_files:
            try:
                data = json.loads(file.read_text())
                timestamp = datetime.fromisoformat(data['extracted_at'])
                age = now - timestamp

                if age > timedelta(seconds=ttl_documents):
                    age_hours = age.total_seconds() / 3600
                    if dry_run:
                        print(f"   [DRY-RUN] Removeria: {file.name} (idade: {age_hours:.1f}h)")
                    else:
                        file.unlink()
                        print(f"   ✓ Removido: {file.name} (idade: {age_hours:.1f}h)")
                    total_removed += 1

            except Exception as e:
                print(f"   ⚠ Erro ao processar {file.name}: {e}")

    return total_removed

def show_cache_stats(cache_dir: Path):
    """Mostra estatísticas do cache"""
    queries_dir = cache_dir / "queries"
    documents_dir = cache_dir / "documents"
    extracted_dir = cache_dir / "extracted"

    print("\n📊 Estatísticas do Cache:")
    print("=" * 50)

    for directory, label in [
        (queries_dir, "Queries"),
        (documents_dir, "Documentos"),
        (extracted_dir, "Parquet/Extracted")
    ]:
        if directory.exists():
            files = list(directory.glob("*"))
            total_size = sum(f.stat().st_size for f in files if f.is_file())
            print(f"\n{label}:")
            print(f"  Arquivos: {len(files)}")
            print(f"  Tamanho: {total_size / 1024:.2f} KB")
        else:
            print(f"\n{label}: (diretório não existe)")

def main():
    parser = argparse.ArgumentParser(
        description="Limpa cache do Data Orchestrator MCP",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exemplos:
  %(prog)s --mode all              # Limpa todo o cache
  %(prog)s --mode expired          # Limpa apenas expirados
  %(prog)s --mode all --dry-run    # Simula limpeza completa
  %(prog)s --stats                 # Mostra estatísticas
        """
    )

    parser.add_argument(
        '--mode',
        choices=['all', 'expired'],
        help='Modo de limpeza: "all" (tudo) ou "expired" (apenas expirados)'
    )

    parser.add_argument(
        '--dry-run',
        action='store_true',
        help='Simula a limpeza sem deletar arquivos'
    )

    parser.add_argument(
        '--stats',
        action='store_true',
        help='Mostra estatísticas do cache'
    )

    args = parser.parse_args()

    # Validar argumentos
    if not args.mode and not args.stats:
        parser.error("É necessário especificar --mode ou --stats")

    # Carregar configuração
    config = load_config()
    cache_dir = Path(config['cache']['directory'])

    if not cache_dir.exists():
        print(f"⚠ Diretório de cache não existe: {cache_dir}")
        sys.exit(1)

    # Mostrar estatísticas
    if args.stats:
        show_cache_stats(cache_dir)
        if not args.mode:
            return

    # Executar limpeza
    print("\n🧹 Limpeza de Cache")
    print("=" * 50)
    print(f"Diretório: {cache_dir}")
    print(f"Modo: {args.mode}")
    print(f"Dry-run: {args.dry_run}")

    if args.dry_run:
        print("\n⚠ MODO DRY-RUN: Nenhum arquivo será deletado")

    ttl_queries = config['cache']['ttl_queries']
    ttl_documents = config['cache']['ttl_documents']

    if args.mode == 'all':
        removed = clean_all(cache_dir, args.dry_run)
    else:  # expired
        removed = clean_expired(cache_dir, ttl_queries, ttl_documents, args.dry_run)

    print("\n" + "=" * 50)
    if args.dry_run:
        print(f"✓ {removed} arquivo(s) seriam removidos")
    else:
        print(f"✓ {removed} arquivo(s) removido(s)")

    # Mostrar estatísticas finais
    if not args.dry_run and removed > 0:
        show_cache_stats(cache_dir)

if __name__ == "__main__":
    main()
