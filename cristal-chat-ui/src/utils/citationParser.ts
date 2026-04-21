/**
 * Parser de citações para detectar e processar padrão [texto]^N
 * Converte para HTML customizado que o ReactMarkdown pode processar
 */

/**
 * Verifica se o conteúdo contém citações no formato [texto]^N
 */
export function hasCitations(content: string): boolean {
  return /\[([^\]]+)\]\^(\d+)/g.test(content);
}

/**
 * Pré-processa conteúdo convertendo [texto]^N para <cite data-num="N">texto</cite>
 * Este formato é reconhecido pelo ReactMarkdown e pode ser customizado
 */
export function preprocessCitations(content: string): string {
  return content.replace(
    /\[([^\]]+)\]\^(\d+)/g,
    '<cite data-num="$2">$1</cite>'
  );
}

/**
 * Extrai informações de todas as citações encontradas no texto
 */
export interface ParsedCitation {
  text: string;
  number: number;
  startIndex: number;
  endIndex: number;
}

export function parseCitations(content: string): ParsedCitation[] {
  const regex = /\[([^\]]+)\]\^(\d+)/g;
  const citations: ParsedCitation[] = [];
  let match;

  while ((match = regex.exec(content)) !== null) {
    citations.push({
      text: match[1],
      number: parseInt(match[2], 10),
      startIndex: match.index,
      endIndex: match.index + match[0].length
    });
  }

  return citations;
}
