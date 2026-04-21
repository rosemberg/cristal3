import { useState } from 'react';
import { ComposerInput } from '../components/composer';

/**
 * Exemplo de uso do ComposerInput
 * CAMADA 3 - FASE 5: Textarea inteligente com auto-resize
 *
 * Demonstra:
 * - Auto-resize (1 linha inicial, cresce até 6 linhas)
 * - Enter para enviar, Shift+Enter para quebrar linha
 * - Estados disabled, focus, blur
 */
function ComposerInputExample() {
  const [value, setValue] = useState('');
  const [messages, setMessages] = useState<string[]>([]);
  const [disabled, setDisabled] = useState(false);
  const [isFocused, setIsFocused] = useState(false);

  const handleSubmit = () => {
    if (!value.trim()) return;

    setMessages([...messages, value]);
    setValue('');
  };

  return (
    <div className="min-h-screen bg-background-main p-8">
      <div className="max-w-3xl mx-auto space-y-6">
        <div>
          <h1 className="text-2xl font-bold text-text-main mb-2">
            ComposerInput - Exemplo
          </h1>
          <p className="text-text-secondary text-sm">
            Textarea inteligente com auto-resize e keyboard handling
          </p>
        </div>

        {/* Controles de teste */}
        <div className="bg-white rounded-lg border border-border-light p-4">
          <h2 className="text-sm font-semibold text-text-main mb-3">
            Controles de Teste
          </h2>
          <label className="flex items-center gap-2 text-sm text-text-main">
            <input
              type="checkbox"
              checked={disabled}
              onChange={(e) => setDisabled(e.target.checked)}
              className="w-4 h-4"
            />
            Desabilitar input
          </label>
          <div className="mt-2 text-xs text-text-secondary">
            Estado do foco: {isFocused ? 'Focado' : 'Não focado'}
          </div>
        </div>

        {/* ComposerInput em ação */}
        <div className="bg-white rounded-lg border border-border-light p-4">
          <h2 className="text-sm font-semibold text-text-main mb-3">
            ComposerInput
          </h2>
          <div className="border border-border-light rounded-lg p-3 bg-background-main">
            <ComposerInput
              value={value}
              onChange={setValue}
              onSubmit={handleSubmit}
              onFocus={() => setIsFocused(true)}
              onBlur={() => setIsFocused(false)}
              placeholder="Digite sua mensagem... (Enter envia, Shift+Enter quebra linha)"
              disabled={disabled}
              maxRows={6}
            />
          </div>
          <div className="mt-2 text-xs text-text-secondary">
            Caracteres: {value.length} | Linhas: {value.split('\n').length}
          </div>
        </div>

        {/* Mensagens enviadas */}
        {messages.length > 0 && (
          <div className="bg-white rounded-lg border border-border-light p-4">
            <h2 className="text-sm font-semibold text-text-main mb-3">
              Mensagens Enviadas ({messages.length})
            </h2>
            <div className="space-y-2">
              {messages.map((msg, index) => (
                <div
                  key={index}
                  className="bg-background-main rounded-lg p-3 text-sm text-text-main whitespace-pre-wrap"
                >
                  {msg}
                </div>
              ))}
            </div>
            <button
              onClick={() => setMessages([])}
              className="mt-3 text-xs text-primary-main hover:underline"
            >
              Limpar mensagens
            </button>
          </div>
        )}

        {/* Instruções */}
        <div className="bg-blue-50 rounded-lg border border-blue-200 p-4">
          <h2 className="text-sm font-semibold text-text-main mb-2">
            Instruções
          </h2>
          <ul className="text-xs text-text-secondary space-y-1">
            <li>• Digite texto e observe o textarea crescer automaticamente</li>
            <li>• Pressione <kbd className="px-1 py-0.5 bg-white border border-border-light rounded text-xs">Enter</kbd> para enviar</li>
            <li>• Pressione <kbd className="px-1 py-0.5 bg-white border border-border-light rounded text-xs">Shift</kbd> + <kbd className="px-1 py-0.5 bg-white border border-border-light rounded text-xs">Enter</kbd> para quebrar linha</li>
            <li>• Pressione <kbd className="px-1 py-0.5 bg-white border border-border-light rounded text-xs">Esc</kbd> para remover foco</li>
            <li>• Altura inicial: 44px (1 linha)</li>
            <li>• Altura máxima: 120px (6 linhas) - scroll aparece após isso</li>
          </ul>
        </div>
      </div>
    </div>
  );
}

export default ComposerInputExample;
