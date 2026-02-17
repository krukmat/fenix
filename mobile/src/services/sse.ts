// Task 4.2 — FR-300: SSE Stub
// Implementación completa en Task 4.4

export type SSEMessage =
  | { type: 'token'; delta: string }
  | { type: 'evidence'; sources: unknown[] }
  | { type: 'done' }
  | { type: 'error'; message: string };

/**
 * Creates an SSE client connection to the BFF.
 * This is a stub - full implementation in Task 4.4
 */
export function createSSEClient(_url: string, _token: string): { close: () => void } {
  // Stub - Task 4.4 will implement the full SSE connection
  return {
    close: () => undefined,
  };
}
