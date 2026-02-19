import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { renderHook, act } from '@testing-library/react-native';
import type { createSSEClient } from '../../src/services/sse';

import { useSSE } from '../../src/hooks/useSSE';

const mockClose = jest.fn();
const mockCreateSSEClient = jest.fn<typeof createSSEClient>();

jest.mock('../../src/services/sse', () => ({
  createSSEClient: (...args: unknown[]) => mockCreateSSEClient(...args),
}));

jest.mock('../../src/services/api', () => ({
  copilotApi: {
    buildChatUrl: jest.fn(() => 'http://localhost:3000/bff/copilot/chat'),
  },
}));

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: {
    getState: jest.fn(() => ({ token: 'jwt-token' })),
  },
}));

describe('useSSE', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockCreateSSEClient.mockReturnValue({ close: mockClose });
  });

  it('sendQuery appends user message + streaming assistant', () => {
    const { result } = renderHook(() => useSSE());

    act(() => {
      result.current.sendQuery('hola', { entityType: 'case', entityId: 'c-1' });
    });

    expect(result.current.messages).toHaveLength(2);
    expect(result.current.messages[0].role).toBe('user');
    expect(result.current.messages[0].content).toBe('hola');
    expect(result.current.messages[1].role).toBe('assistant');
    expect(result.current.messages[1].isStreaming).toBe(true);
    expect(result.current.isStreaming).toBe(true);
    expect(mockCreateSSEClient).toHaveBeenCalled();
  });

  it('accumulates token deltas into last assistant message', () => {
    let onMessage: ((msg: { type: string; [key: string]: unknown }) => void) | undefined;
    mockCreateSSEClient.mockImplementation(
      (_url: string, _token: string, _body: Record<string, unknown>, cb: (msg: { type: string; [key: string]: unknown }) => void) => {
        onMessage = cb;
        return { close: mockClose };
      },
    );

    const { result } = renderHook(() => useSSE());
    act(() => {
      result.current.sendQuery('precio');
    });

    act(() => {
      onMessage?.({ type: 'token', delta: 'hola ' });
      onMessage?.({ type: 'token', delta: 'mundo' });
    });

    expect(result.current.messages[1].content).toBe('hola mundo');
  });

  it('attaches evidence sources on evidence event', () => {
    let onMessage: ((msg: { type: string; [key: string]: unknown }) => void) | undefined;
    mockCreateSSEClient.mockImplementation(
      (_url: string, _token: string, _body: Record<string, unknown>, cb: (msg: { type: string; [key: string]: unknown }) => void) => {
        onMessage = cb;
        return { close: mockClose };
      },
    );

    const { result } = renderHook(() => useSSE());
    act(() => {
      result.current.sendQuery('evidence');
    });

    act(() => {
      onMessage?.({
        type: 'evidence',
        sources: [{ id: 'e1', snippet: 'texto', score: 0.95, timestamp: '2026-01-01T00:00:00Z' }],
      });
    });

    expect(result.current.messages[1].evidenceSources).toHaveLength(1);
    expect(result.current.messages[1].evidenceSources?.[0].id).toBe('e1');
  });

  it('sets isStreaming false on done event', () => {
    let onMessage: ((msg: { type: string; [key: string]: unknown }) => void) | undefined;
    mockCreateSSEClient.mockImplementation(
      (_url: string, _token: string, _body: Record<string, unknown>, cb: (msg: { type: string; [key: string]: unknown }) => void) => {
        onMessage = cb;
        return { close: mockClose };
      },
    );

    const { result } = renderHook(() => useSSE());
    act(() => {
      result.current.sendQuery('done');
    });

    act(() => {
      onMessage?.({ type: 'done' });
    });

    expect(result.current.isStreaming).toBe(false);
    expect(result.current.messages[1].isStreaming).toBe(false);
  });

  it('calls close on unmount', () => {
    const { result, unmount } = renderHook(() => useSSE());
    act(() => {
      result.current.sendQuery('x');
    });

    unmount();
    expect(mockClose).toHaveBeenCalled();
  });

  it('sets error state on error event', () => {
    let onMessage: ((msg: { type: string; [key: string]: unknown }) => void) | undefined;
    mockCreateSSEClient.mockImplementation(
      (_url: string, _token: string, _body: Record<string, unknown>, cb: (msg: { type: string; [key: string]: unknown }) => void) => {
        onMessage = cb;
        return { close: mockClose };
      },
    );

    const { result } = renderHook(() => useSSE());
    act(() => {
      result.current.sendQuery('x');
    });

    act(() => {
      onMessage?.({ type: 'error', message: 'boom' });
    });

    expect(result.current.error).toBe('boom');
    expect(result.current.isStreaming).toBe(false);
  });
});
