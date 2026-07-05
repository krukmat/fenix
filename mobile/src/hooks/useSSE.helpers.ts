import type { SuggestedAction } from '../components/copilot/ActionButton';
import type {
  CopilotAbstentionReason,
  CopilotAnswerType,
  CopilotConfidence,
  CopilotEvidenceMeta,
  EvidenceSource,
  SSEClient,
  SSEMessage,
} from '../services/sse';

export interface CopilotMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  evidenceSources?: EvidenceSource[];
  evidenceMeta?: CopilotEvidenceMeta;
  confidence?: CopilotConfidence;
  warnings?: string[];
  answerType?: CopilotAnswerType;
  abstentionReason?: CopilotAbstentionReason;
  actions?: SuggestedAction[];
  isStreaming?: boolean;
}

export type SendContext = {
  entityType?: string;
  entityId?: string;
  signalId?: string;
  signalType?: string;
};

function makeId(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

export function buildRequestBody(query: string, context?: SendContext): Record<string, unknown> {
  const body: Record<string, unknown> = { query };
  if (context?.entityType) body.entityType = context.entityType;
  if (context?.entityId) body.entityId = context.entityId;
  if (context?.signalId) body.signalId = context.signalId;
  if (context?.signalType) body.signalType = context.signalType;
  return body;
}

export function makeInitialMessages(trimmed: string): [CopilotMessage, CopilotMessage] {
  return [
    { id: makeId('u'), role: 'user', content: trimmed },
    { id: makeId('a'), role: 'assistant', content: '', isStreaming: true },
  ];
}

export function updateLastAssistantMessage(
  messages: CopilotMessage[],
  fn: (message: CopilotMessage) => CopilotMessage,
): CopilotMessage[] {
  const idx = [...messages].reverse().findIndex((message) => message.role === 'assistant');
  if (idx === -1) {
    return messages;
  }

  const realIndex = messages.length - 1 - idx;
  const nextMessages = [...messages];
  nextMessages[realIndex] = fn(nextMessages[realIndex]);
  return nextMessages;
}

export function finishStreamingMessage(message: CopilotMessage): CopilotMessage {
  return { ...message, isStreaming: false };
}

export function createStreamMessageHandler({
  updateAssistant,
  setStreaming,
  setError,
}: {
  updateAssistant: (fn: (message: CopilotMessage) => CopilotMessage) => void;
  setStreaming: (value: boolean) => void;
  setError: (value: string | null) => void;
}) {
  return (message: SSEMessage) => {
    if (message.type === 'token') {
      updateAssistant((last) => ({ ...last, content: `${last.content}${message.delta}` }));
      return;
    }

    if (message.type === 'evidence') {
      updateAssistant((last) => ({
        ...last,
        evidenceSources: message.sources,
        evidenceMeta: message.meta,
        confidence: message.meta?.confidence,
        warnings: message.meta?.warnings,
      }));
      return;
    }

    if (message.type === 'done') {
      setStreaming(false);
      updateAssistant((last) => ({
        ...finishStreamingMessage(last),
        answerType: message.answerType ?? last.answerType,
        abstentionReason: message.abstentionReason ?? last.abstentionReason,
      }));
      return;
    }

    if (message.type === 'error') {
      setError(message.message);
      setStreaming(false);
      updateAssistant(finishStreamingMessage);
    }
  };
}

export function closeActiveClient(clientRef: { current: SSEClient | null }) {
  clientRef.current?.close();
}
