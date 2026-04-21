/**
 * Formata um objeto Date para hora no formato HH:MM
 */
export function formatTime(date: Date): string {
  return new Intl.DateTimeFormat('pt-BR', {
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}
