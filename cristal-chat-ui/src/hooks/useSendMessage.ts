import { useMutation } from '@tanstack/react-query';
import { v4 as uuidv4 } from 'uuid';
import { apiClient } from '../api/client';
import { useChatStore } from '../store/chatStore';
import type { Message } from '../types/chat';

/**
 * Hook para enviar mensagens usando TanStack Query mutation
 * Gerencia estado de loading, adiciona mensagens do usuário e assistente,
 * e processa citações retornadas pela API
 */
export const useSendMessage = () => {
  const addMessage = useChatStore((state) => state.addMessage);
  const addCitations = useChatStore((state) => state.addCitations);
  const setLoading = useChatStore((state) => state.setLoading);
  const setError = useChatStore((state) => state.setError);

  return useMutation({
    mutationFn: (message: string) => apiClient.sendMessage(message),

    onMutate: async (message: string) => {
      // Adicionar mensagem do usuário imediatamente
      const userMsg: Message = {
        id: uuidv4(),
        role: 'user',
        content: message,
        timestamp: new Date(),
      };

      addMessage(userMsg);
      setLoading(true);
      setError(null);
    },

    onSuccess: (data) => {
      // Adicionar resposta da IA
      const assistantMsg: Message = {
        id: uuidv4(),
        role: 'assistant',
        content: data.response,
        timestamp: new Date(),
      };

      addMessage(assistantMsg);

      // Adicionar citações se houver
      if (data.citations && data.citations.length > 0) {
        addCitations(data.citations);
      }

      setLoading(false);
    },

    onError: (error: Error) => {
      setError(error.message || 'Erro ao enviar mensagem');
      setLoading(false);
    },
  });
};
