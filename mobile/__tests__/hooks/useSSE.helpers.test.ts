import { describe, expect, it } from '@jest/globals';
import {
  buildRequestBody,
  createStreamMessageHandler,
  finishStreamingMessage,
  makeInitialMessages,
  updateLastAssistantMessage,
  type CopilotMessage,
} from '../../src/hooks/useSSE.helpers';

describe('useSSE helpers', () => {
  it('buildRequestBody includes only provided context fields', () => {
    expect(
      buildRequestBody('hello', { entityType: 'deal', signalId: 'sig-1' }),
    ).toEqual({
      query: 'hello',
      entityType: 'deal',
      signalId: 'sig-1',
    });
  });

  it('makeInitialMessages creates user and streaming assistant messages', () => {
    const [user, assistant] = makeInitialMessages('hola');
    expect(user.role).toBe('user');
    expect(user.content).toBe('hola');
    expect(assistant.role).toBe('assistant');
    expect(assistant.isStreaming).toBe(true);
  });

  it('updateLastAssistantMessage updates the latest assistant message only', () => {
    const messages: CopilotMessage[] = [
      { id: 'u1', role: 'user', content: 'q1' },
      { id: 'a1', role: 'assistant', content: 'old' },
      { id: 'u2', role: 'user', content: 'q2' },
      { id: 'a2', role: 'assistant', content: 'last' },
    ];

    const next = updateLastAssistantMessage(messages, (message) => ({
      ...message,
      content: `${message.content}!`,
    }));

    expect(next[1].content).toBe('old');
    expect(next[3].content).toBe('last!');
  });

  it('finishStreamingMessage clears the streaming flag', () => {
    expect(
      finishStreamingMessage({ id: 'a1', role: 'assistant', content: 'x', isStreaming: true }).isStreaming,
    ).toBe(false);
  });

  it('createStreamMessageHandler handles token, evidence, done, and error messages', () => {
    let latest: CopilotMessage = { id: 'a1', role: 'assistant', content: '', isStreaming: true };
    let streaming = true;
    let error: string | null = null;

    const handler = createStreamMessageHandler({
      updateAssistant: (fn) => {
        latest = fn(latest);
      },
      setStreaming: (value) => {
        streaming = value;
      },
      setError: (value) => {
        error = value;
      },
    });

    handler({ type: 'token', delta: 'hola' });
    handler({
      type: 'evidence',
      sources: [{ id: 'e1', snippet: 'snippet', score: 1, timestamp: '2026-01-01T00:00:00Z' }],
    });
    handler({ type: 'done' });

    expect(latest.content).toBe('hola');
    expect(latest.evidenceSources?.[0].id).toBe('e1');
    expect(streaming).toBe(false);
    expect(latest.isStreaming).toBe(false);
    expect(error).toBeNull();

    handler({ type: 'error', message: 'boom' });
    expect(error).toBe('boom');
  });
});
