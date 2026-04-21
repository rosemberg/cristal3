import type { Message } from '../types/chat';

/**
 * Mock data para testes de UI
 * Usado para demonstrar componentes de chat com dados de exemplo
 */
export const mockMessages: Message[] = [
  {
    id: '1',
    role: 'user',
    content: 'Como contestar uma multa eleitoral?',
    timestamp: new Date('2026-04-21T10:00:00')
  },
  {
    id: '2',
    role: 'assistant',
    content: `Para contestar uma multa eleitoral, você deve seguir os seguintes passos:

1. Acesse o portal de transparência do TRE-PI
2. Navegue até a seção "Multas e Penalidades"
3. Clique em "Contestar Multa"
4. Preencha o formulário com seus dados
5. Anexe documentos comprobatórios

**Importante**: O prazo para contestação é de **30 dias corridos** a partir da notificação.`,
    timestamp: new Date('2026-04-21T10:00:05')
  },
  {
    id: '3',
    role: 'user',
    content: 'Quais diárias foram pagas em março?',
    timestamp: new Date('2026-04-21T10:05:00')
  },
  {
    id: '4',
    role: 'assistant',
    content: `Em março de 2026, foram pagas as seguintes diárias:

- **João Silva**: R$ 1.500,00 (viagem a Brasília)
- **Maria Santos**: R$ 2.300,00 (viagem a São Paulo)
- **Pedro Oliveira**: R$ 800,00 (viagem a Salvador)

**Total geral**: R$ 4.600,00

Você pode consultar o detalhamento completo no [Portal de Transparência](https://www.tre-pi.jus.br/transparencia).`,
    timestamp: new Date('2026-04-21T10:05:08')
  },
  {
    id: '5',
    role: 'user',
    content: 'Onde consultar as licitações abertas?',
    timestamp: new Date('2026-04-21T10:10:00')
  },
  {
    id: '6',
    role: 'assistant',
    content: `Atualmente, há 3 editais vigentes no TRE-PI:

1. [Pregão Eletrônico nº 001/2026]^1 - Aquisição de equipamentos de informática (prazo até 15/05/2026)
2. [Concorrência nº 002/2026]^2 - Reforma do prédio sede (prazo até 30/05/2026)
3. [Dispensa nº 003/2026]^3 - Serviços de manutenção predial (prazo até 10/05/2026)

Todos os editais estão disponíveis no [portal de transparência]^1 com documentação completa.`,
    timestamp: new Date('2026-04-21T10:10:05')
  }
];
