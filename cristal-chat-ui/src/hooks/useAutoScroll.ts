import { useEffect, useRef } from 'react';

/**
 * Hook para auto-scroll suave para a última mensagem
 * Utilizado para manter o chat sempre na parte mais recente
 *
 * @param dependencies - Dependências que quando mudarem, ativam o scroll
 * @returns Ref para ser anexada ao elemento de scroll
 */
export const useAutoScroll = <T extends HTMLElement>(
  dependencies: any[]
) => {
  const scrollRef = useRef<T | null>(null);

  useEffect(() => {
    const scrollElement = scrollRef.current;
    if (!scrollElement) return;

    // Scroll para o final com animação suave
    scrollElement.scrollTo({
      top: scrollElement.scrollHeight,
      behavior: 'smooth',
    });
  }, dependencies);

  return scrollRef;
};
