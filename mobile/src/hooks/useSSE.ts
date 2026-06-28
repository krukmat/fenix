import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useAuthStore } from '../stores/authStore';
import { copilotApi } from '../services/api';
import { createSSEClient, type SSEClient } from '../services/sse';
import {
  buildRequestBody,
  closeActiveClient,
  createStreamMessageHandler,
  makeInitialMessages,
  updateLastAssistantMessage,
  type CopilotMessage,
  type SendContext,
} from './useSSE.helpers';

export type { CopilotMessage, SendContext } from './useSSE.helpers';

export function useSSE() {
  const [messages, setMessages] = useState<CopilotMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const clientRef = useRef<SSEClient | null>(null);

  const clearMessages = useCallback(() => {
    setMessages([]);
    setError(null);
    setIsStreaming(false);
  }, []);

  const updateLastAssistant = useCallback((fn: (msg: CopilotMessage) => CopilotMessage) => {
    setMessages((prev) => updateLastAssistantMessage(prev, fn));
  }, []);

  const onStreamMessage = useMemo(
    () =>
      createStreamMessageHandler({
      updateAssistant: updateLastAssistant,
      setStreaming: setIsStreaming,
      setError,
    }),
    [updateLastAssistant],
  );

  const sendQuery = useCallback(
    (query: string, context?: SendContext) => {
      const trimmed = query.trim();
      if (!trimmed) return;

      const token = useAuthStore.getState().token;
      if (!token) {
        setError('Not authenticated');
        return;
      }

      closeActiveClient(clientRef);
      setError(null);
      setIsStreaming(true);

      const [userMsg, assistantMsg] = makeInitialMessages(trimmed);
      setMessages((prev) => [...prev, userMsg, assistantMsg]);

      clientRef.current = createSSEClient(copilotApi.buildChatUrl(), token, buildRequestBody(trimmed, context), onStreamMessage, (err) => {
        setError(err.message);
        setIsStreaming(false);
      });
    },
    [onStreamMessage],
  );

  useEffect(
    () => () => {
      closeActiveClient(clientRef);
    },
    [],
  );

  return {
    messages,
    isStreaming,
    error,
    sendQuery,
    clearMessages,
  };
}
