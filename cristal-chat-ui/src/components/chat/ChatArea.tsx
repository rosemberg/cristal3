import React, { useEffect, useRef } from 'react';
import WelcomeCard from './WelcomeCard';
import MessageTurn from './MessageTurn';
import { useChatStore } from '../../store/chatStore';
import type { Message } from '../../types/chat';

interface ChatAreaProps {
  messages: Message[];
  isLoading?: boolean;
  onSuggestionClick?: (suggestion: string) => void;
  children?: React.ReactNode;
  className?: string;
}

/**
 * Área principal de chat
 * Fundo: chat-bg, padding: p-4, overflow-y: auto, scrollbar customizado
 */
const ChatArea: React.FC<ChatAreaProps> = ({
  messages,
  isLoading = false,
  onSuggestionClick,
  children,
  className = ''
}) => {
  const chatEndRef = useRef<HTMLDivElement>(null);
  const citations = useChatStore((state) => state.citations);

  // Auto-scroll para última mensagem
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, isLoading]);

  return (
    <div
      className={`flex-1 overflow-y-auto scroll-smooth ${className}`}
      style={{
        backgroundColor: 'var(--paper)',
        padding: '22px 24px 28px',
      }}
    >
      {children || (
        <>
          {messages.length === 0 ? (
            <WelcomeCard onSuggestionClick={onSuggestionClick || (() => {})} className="animate-fade-up" />
          ) : (
            <>
              {messages.map((message) => (
                <MessageTurn
                  key={message.id}
                  message={message}
                  citations={message.role === 'assistant' ? citations : undefined}
                  className="animate-fade-up"
                />
              ))}
              <div ref={chatEndRef} />
            </>
          )}
        </>
      )}
    </div>
  );
};

export default ChatArea;
