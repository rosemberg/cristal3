import { create } from 'zustand';
import type { Message, Citation } from '../types/chat';

/**
 * Store global do chat usando Zustand
 * Gerencia estado de mensagens, citações, loading e erros
 */
interface ChatStore {
  // Estado
  messages: Message[];
  citations: Citation[];
  isLoading: boolean;
  error: string | null;

  // Ações
  addMessage: (message: Message) => void;
  addCitations: (citations: Citation[]) => void;
  clearChat: () => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

/**
 * Estado inicial do chat
 * IMPORTANTE: messages inicia VAZIO para mostrar WelcomeCard
 */
const initialState = {
  messages: [] as Message[],
  citations: [] as Citation[],
  isLoading: false,
  error: null as string | null,
};

/**
 * Hook do store Zustand
 */
export const useChatStore = create<ChatStore>((set) => ({
  ...initialState,

  addMessage: (message) =>
    set((state) => ({
      messages: [...state.messages, message],
    })),

  addCitations: (citations) =>
    set((state) => ({
      citations: [...state.citations, ...citations],
    })),

  clearChat: () =>
    set({
      messages: [],
      citations: [],
      error: null,
    }),

  setLoading: (loading) =>
    set({
      isLoading: loading,
    }),

  setError: (error) =>
    set({
      error,
    }),
}));
