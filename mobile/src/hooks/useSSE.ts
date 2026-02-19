import { useCallback, useEffect, useRef, useState } from 'react';
import { useAuthStore } from '../stores/authStore';
import { copilotApi } from '../services/api';
import { createSSEClient, type EvidenceSource, type SSEClient, type SSEMessage } from '../services/sse';
import type { SuggestedAction } from '../components/copilot/ActionButton';

export interface CopilotMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  evidenceSources?: EvidenceSource[];
  actions?: SuggestedAction[];
  isStreaming?: boolean;
}

type SendContext = {
  entityType?: string;
  entityId?: string;
};

function makeId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

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
    setMessages((prev) => {
      const idx = [...prev].reverse().findIndex((m) => m.role === 'assistant');
      if (idx === -1) return prev;
      const realIndex = prev.length - 1 - idx;
      const next = [...prev];
      next[realIndex] = fn(next[realIndex]);
      return next;
    });
  }, []);

  const onStreamMessage = useCallback(
    (msg: SSEMessage) => {
      if (msg.type === 'token') {
        updateLastAssistant((last) => ({ ...last, content: `${last.content}${msg.delta}` }));
        return;
      }

      if (msg.type === 'evidence') {
        updateLastAssistant((last) => ({ ...last, evidenceSources: msg.sources }));
        return;
      }

      if (msg.type === 'done') {
        setIsStreaming(false);
        updateLastAssistant((last) => ({ ...last, isStreaming: false }));
        return;
      }

      if (msg.type === 'error') {
        setError(msg.message);
        setIsStreaming(false);
        updateLastAssistant((last) => ({ ...last, isStreaming: false }));
      }
    },
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

      clientRef.current?.close();
      setError(null);
      setIsStreaming(true);

      const userMsg: CopilotMessage = { id: makeId('u'), role: 'user', content: trimmed };
      const assistantMsg: CopilotMessage = { id: makeId('a'), role: 'assistant', content: '', isStreaming: true };
      setMessages((prev) => [...prev, userMsg, assistantMsg]);

      const body: Record<string, unknown> = { query: trimmed };
      if (context?.entityType) body.entityType = context.entityType;
      if (context?.entityId) body.entityId = context.entityId;

      clientRef.current = createSSEClient(copilotApi.buildChatUrl(), token, body, onStreamMessage, (err) => {
        setError(err.message);
        setIsStreaming(false);
      });
    },
    [onStreamMessage],
  );

  useEffect(
    () => () => {
      clientRef.current?.close();
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
