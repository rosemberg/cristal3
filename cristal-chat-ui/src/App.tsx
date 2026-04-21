import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import HostFrame from './components/layout/HostFrame';
import Header from './components/layout/Header';
import YellowDivider from './components/layout/YellowDivider';
import Footer from './components/layout/Footer';
import ChatArea from './components/chat/ChatArea';
import { Composer } from './components/composer';
import { useChatStore } from './store/chatStore';
import { useSendMessage } from './hooks/useSendMessage';
import './index.css';
import './styles/animations.css';

/**
 * Cliente TanStack Query
 * Gerencia cache e estado das requisições
 */
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

/**
 * Componente principal do chat
 * FASE 6: Integração com Backend REST API
 */
function ChatApp() {
  // Estado global do Zustand
  const messages = useChatStore((state) => state.messages);
  const isLoading = useChatStore((state) => state.isLoading);
  const error = useChatStore((state) => state.error);
  const clearChat = useChatStore((state) => state.clearChat);
  const setError = useChatStore((state) => state.setError);

  // Hook de mutation para enviar mensagens
  const { mutate: sendMessage } = useSendMessage();

  /**
   * Handler para enviar mensagem (via Composer ou sugestão)
   */
  const handleSendMessage = (content: string) => {
    // Limpar erro anterior se existir
    if (error) {
      setError(null);
    }

    // Enviar mensagem via mutation
    sendMessage(content);
  };

  /**
   * Handler para limpar o chat
   */
  const handleClearChat = () => {
    clearChat();
  };

  return (
    <HostFrame>
      <div className="flex flex-col h-full">
        {/* Chat Area (flex-1 para ocupar espaço disponível) */}
        <div className="flex-1 overflow-hidden">
          <ChatArea
            messages={messages}
            isLoading={isLoading}
            onSuggestionClick={handleSendMessage}
          />

          {/* Error Banner */}
          {error && (
            <div className="absolute top-4 left-1/2 transform -translate-x-1/2 z-50">
              <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg shadow-lg max-w-md">
                <div className="flex items-center gap-2">
                  <svg
                    className="w-5 h-5 flex-shrink-0"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                      clipRule="evenodd"
                    />
                  </svg>
                  <span className="text-sm font-medium">{error}</span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Composer (fixo no bottom) */}
        <Composer
          onSendMessage={handleSendMessage}
          onClearChat={messages.length > 0 ? handleClearChat : undefined}
          isDisabled={isLoading}
          showMetaBar={messages.length > 0}
          placeholder="Pergunte à Cristal..."
        />
      </div>
    </HostFrame>
  );
}

/**
 * App Root com QueryClientProvider
 */
function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ChatApp />
    </QueryClientProvider>
  );
}

export default App;
