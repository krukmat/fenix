// useSSE — signal context forwarding in SSE request body
// FR-200 (Copilot SSE), UC-A5: signalId/signalType included in copilot request

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

describe('useSSE — signal context in request body', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockCreateSSEClient.mockReturnValue({ close: mockClose });
  });

  it('includes signalId and signalType in SSE request body', () => {
    const { result } = renderHook(() => useSSE());

    act(() => {
      result.current.sendQuery('Explain this signal', {
        entityType: 'deal',
        entityId: 'd-1',
        signalId: 'sig-1',
        signalType: 'churn_risk',
      });
    });

    const body = mockCreateSSEClient.mock.calls[0][2] as Record<string, unknown>;
    expect(body.signalId).toBe('sig-1');
    expect(body.signalType).toBe('churn_risk');
    expect(body.entityType).toBe('deal');
    expect(body.entityId).toBe('d-1');
    expect(body.query).toBe('Explain this signal');
  });

  it('does NOT include signalId/signalType when context omits them', () => {
    const { result } = renderHook(() => useSSE());

    act(() => {
      result.current.sendQuery('Summarize', { entityType: 'account', entityId: 'a-1' });
    });

    const body = mockCreateSSEClient.mock.calls[0][2] as Record<string, unknown>;
    expect(body.signalId).toBeUndefined();
    expect(body.signalType).toBeUndefined();
    expect(body.entityType).toBe('account');
  });

  it('omits all context fields when sendQuery called without context', () => {
    const { result } = renderHook(() => useSSE());

    act(() => {
      result.current.sendQuery('hello');
    });

    const body = mockCreateSSEClient.mock.calls[0][2] as Record<string, unknown>;
    expect(body.signalId).toBeUndefined();
    expect(body.signalType).toBeUndefined();
    expect(body.entityType).toBeUndefined();
    expect(body.entityId).toBeUndefined();
    expect(body.query).toBe('hello');
  });

  it('includes only signalId when only signalId is provided', () => {
    const { result } = renderHook(() => useSSE());

    act(() => {
      result.current.sendQuery('detail', { signalId: 'sig-99' });
    });

    const body = mockCreateSSEClient.mock.calls[0][2] as Record<string, unknown>;
    expect(body.signalId).toBe('sig-99');
    expect(body.signalType).toBeUndefined();
  });
});
