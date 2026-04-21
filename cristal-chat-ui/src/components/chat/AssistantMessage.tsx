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
    <div className={`flex gap-3 mb-4 ${className}`}>
      <Avatar type="assistant" />
      <div className="flex flex-col max-w-[80%] sm:max-w-[80%]">
        <div className="prose prose-sm max-w-none break-words">
          <ReactMarkdown
            rehypePlugins={[rehypeRaw]}
            components={{
              a: ({ ...props }) => (
                <a
                  {...props}
                  className="text-primary-blue underline hover:text-dark-blue transition-colors duration-200"
                  target="_blank"
                  rel="noopener noreferrer"
                />
              ),
              p: ({ ...props }) => (
                <p {...props} className="mb-2 last:mb-0 text-text-main" />
              ),
              ul: ({ ...props }) => (
                <ul {...props} className="list-disc list-inside mb-2 text-text-main" />
              ),
              ol: ({ ...props }) => (
                <ol {...props} className="list-decimal list-inside mb-2 text-text-main" />
              ),
              strong: ({ ...props }) => (
                <strong {...props} className="font-semibold text-text-main" />
              ),
              code: ({ ...props }) => (
                <code {...props} className="bg-pale-blue-bg px-1 py-0.5 rounded text-sm font-mono text-urn-green" />
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
        <span className="text-xs text-text-secondary mt-1">
          {formatTime(timestamp)}
        </span>

        {citations && citations.length > 0 && (
          <CitationsBlock citations={citations} className="mt-4" />
        )}
      </div>
    </div>
  );
};

export default React.memo(AssistantMessage);
