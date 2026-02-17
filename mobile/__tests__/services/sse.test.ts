import { describe, it, expect } from '@jest/globals';
import { createSSEClient } from '../../src/services/sse';

describe('createSSEClient (stub)', () => {
  it('should return a client with close method', () => {
    const client = createSSEClient('http://localhost:3000/sse', 'token');

    expect(client).toBeDefined();
    expect(typeof client.close).toBe('function');
  });

  it('close should be callable without throwing', () => {
    const client = createSSEClient('http://localhost:3000/sse', 'token');

    expect(() => client.close()).not.toThrow();
  });
});
