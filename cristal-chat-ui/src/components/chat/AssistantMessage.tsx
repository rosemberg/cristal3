import React from 'react';
import ReactMarkdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import Avatar from '../ui/Avatar';
import CitationInline from './CitationInline';
import CitationsBlock from './CitationsBlock';
import { preprocessCitations } from '../../utils/citationParser';
import { formatTime } from '../../utils/formatTime';
import type { Citation } from '../../types/citation';

interface AssistantMessageProps {
  content: string;
  timestamp: Date;
  citations?: Citation[];
  className?: string;
}

/**
 * Mensagem do assistente (sem bolha)
 * Texto flutuante com markdown, avatar "IA" à esquerda
 */
const AssistantMessage: React.FC<AssistantMessageProps> = ({
  content,
  timestamp,
  citations,
  className = ''
}) => {
  // Pré-processar conteúdo para converter [texto]^N em <cite data-num="N">texto</cite>
  const processedContent = React.useMemo(
    () => preprocessCitations(content),
    [content]
  );

  return (
    <div className={`flex gap-3 mb-[22px] max-w-[880px] mx-auto ${className}`}>
      <Avatar type="assistant" />
      <div className="flex flex-col max-w-[720px]" style={{ minWidth: 0 }}>
        <div
          className="prose prose-sm max-w-none break-words"
          style={{
            color: 'var(--ink-900)',
            fontSize: '14.5px',
            lineHeight: '1.6',
          }}
        >
          <ReactMarkdown
            rehypePlugins={[rehypeRaw]}
            components={{
              a: ({ ...props }) => (
                <a
                  {...props}
                  className="transition-colors duration-150"
                  style={{
                    color: 'var(--navy-700)',
                    textDecoration: 'underline',
                    textDecorationColor: 'var(--gold-400)',
                  }}
                  target="_blank"
                  rel="noopener noreferrer"
                />
              ),
              p: ({ ...props }) => (
                <p {...props} className="mb-2.5 last:mb-0" style={{ color: 'var(--ink-900)' }} />
              ),
              ul: ({ ...props }) => (
                <ul {...props} className="list-disc list-inside mb-2.5" style={{ color: 'var(--ink-900)' }} />
              ),
              ol: ({ ...props }) => (
                <ol {...props} className="list-decimal list-inside mb-2.5" style={{ color: 'var(--ink-900)' }} />
              ),
              strong: ({ ...props }) => (
                <strong {...props} className="font-semibold" style={{ color: 'var(--ink-900)' }} />
              ),
              code: ({ ...props }) => (
                <code
                  {...props}
                  className="px-1.5 py-0.5 rounded text-sm"
                  style={{
                    backgroundColor: 'var(--navy-050)',
                    fontFamily: 'var(--font-mono)',
                    color: 'var(--green-700)',
                  }}
                />
              ),
              // Componente customizado para citações
              cite: ({ node, children }) => {
                const num = node?.properties?.dataNum as string;
                const citationNum = parseInt(num, 10);
                const citation = citations?.[citationNum - 1];

                return (
                  <CitationInline
                    number={citationNum}
                    href={citation?.url || '#'}
                  >
                    {children}
                  </CitationInline>
                );
              }
            }}
          >
            {processedContent}
          </ReactMarkdown>
        </div>

        {citations && citations.length > 0 && (
          <CitationsBlock citations={citations} className="mt-3.5" />
        )}
      </div>
    </div>
  );
};

export default React.memo(AssistantMessage);
