import type { ChatResponse } from '../types/chat';

/**
 * Cliente HTTP para comunicação com o backend REST API
 * Base URL: http://localhost:8080
 */
class ApiClient {
  private baseUrl: string;
  private timeout: number;

  constructor(baseUrl: string = import.meta.env.VITE_API_BASE_URL || '') {
    // Em desenvolvimento, usa URL relativa para passar pelo proxy do Vite
    // Em produção, usa VITE_API_BASE_URL do .env
    this.baseUrl = baseUrl;
    this.timeout = 60000; // 60 segundos (aumentado para consultas complexas)
  }

  /**
   * Envia uma mensagem para o chatbot
   * POST /chat
   */
  async sendMessage(message: string): Promise<ChatResponse> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(`${this.baseUrl}/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ message }),
        signal: controller.signal,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Erro desconhecido' }));
        throw new Error(errorData.error || `Erro HTTP: ${response.status}`);
      }

      const data = await response.json();
      return data;
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          throw new Error('Tempo limite de resposta excedido. Tente novamente.');
        }
        throw error;
      }

      throw new Error('Erro ao enviar mensagem');
    }
  }

  /**
   * Verifica o status de saúde da API
   * GET /health
   */
  async checkHealth(): Promise<{ status: string }> {
    try {
      const response = await fetch(`${this.baseUrl}/health`, {
        method: 'GET',
      });

      if (!response.ok) {
        throw new Error(`Health check falhou: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Erro ao verificar saúde da API');
    }
  }
}

export const apiClient = new ApiClient();
export default apiClient;
