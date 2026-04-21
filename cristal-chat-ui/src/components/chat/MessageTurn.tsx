import React from 'react';
import UserBubble from './UserBubble';
import AssistantMessage from './AssistantMessage';
import type { Message, Citation } from '../../types/chat';

interface MessageTurnProps {
  message: Message;
  citations?: Citation[];
  className?: string;
}

/**
 * Renderiza uma mensagem (user ou assistant)
 */
const MessageTurn: React.FC<MessageTurnProps> = ({
  message,
  citations,
  className = ''
}) => {
  if (message.role === 'user') {
    return (
      <UserBubble
        content={message.content}
        timestamp={message.timestamp}
        className={className}
      />
    );
  }

  return (
    <AssistantMessage
      content={message.content}
      timestamp={message.timestamp}
      citations={citations}
      className={className}
    />
  );
};

export default React.memo(MessageTurn);
