// bff-http-snapshots T4: deterministic redaction tests — written before implementation (TDD)
import { redactValue, redactObject } from '../../scripts/snapshots/redact';

describe('redactValue', () => {
  it('redacts Bearer tokens', () => {
    expect(redactValue('Bearer eyJhbGciOiJIUzI1NiJ9.abc.def')).toBe('Bearer <REDACTED>');
  });

  it('redacts UUIDs with deterministic placeholders', () => {
    const uuid1 = '550e8400-e29b-41d4-a716-446655440000';
    const uuid2 = 'f47ac10b-58cc-4372-a567-0e02b2c3d479';
    expect(redactValue(uuid1)).toBe('<uuid:1>');
    expect(redactValue(uuid2, { [uuid1]: '<uuid:1>' })).toBe('<uuid:2>');
  });

  it('redacts ISO-8601 timestamps', () => {
    expect(redactValue('2026-04-26T12:00:00.000Z')).toBe('<timestamp>');
    expect(redactValue('2026-04-26T12:00:00Z')).toBe('<timestamp>');
  });

  it('leaves non-sensitive strings unchanged', () => {
    expect(redactValue('hello world')).toBe('hello world');
    expect(redactValue('200')).toBe('200');
  });

  it('leaves numbers unchanged', () => {
    expect(redactValue(42)).toBe(42);
    expect(redactValue(3.14)).toBe(3.14);
  });

  it('leaves booleans unchanged', () => {
    expect(redactValue(true)).toBe(true);
    expect(redactValue(false)).toBe(false);
  });
});

describe('redactObject', () => {
  it('redacts nested Bearer token in headers', () => {
    const input = { headers: { authorization: 'Bearer secret-token-xyz' } };
    const result = redactObject(input) as { headers: Record<string, string> };
    expect(result.headers['authorization']).toBe('Bearer <REDACTED>');
  });

  it('replaces UUIDs consistently within the same object', () => {
    const uuid = '550e8400-e29b-41d4-a716-446655440000';
    const input = { userId: uuid, ownerId: uuid, otherId: 'f47ac10b-58cc-4372-a567-0e02b2c3d479' };
    const result = redactObject(input) as Record<string, string>;
    // Same UUID → same placeholder
    expect(result['userId']).toBe('<uuid:1>');
    expect(result['ownerId']).toBe('<uuid:1>');
    // Different UUID → next placeholder
    expect(result['otherId']).toBe('<uuid:2>');
  });

  it('redacts timestamps in nested objects', () => {
    const input = { created: '2026-01-01T00:00:00Z', nested: { updated: '2026-06-15T08:30:00.000Z' } };
    const result = redactObject(input) as Record<string, unknown>;
    expect(result['created']).toBe('<timestamp>');
    expect((result['nested'] as Record<string, string>)['updated']).toBe('<timestamp>');
  });

  it('redacts sensitive keys regardless of value format', () => {
    const input = { password: 'MySecret123!', token: 'raw-token-value', apiKey: 'sk-live-abc' };
    const result = redactObject(input) as Record<string, string>;
    expect(result['password']).toBe('<REDACTED>');
    expect(result['token']).toBe('<REDACTED>');
    expect(result['apiKey']).toBe('<REDACTED>');
  });

  it('handles arrays of objects', () => {
    const uuid = '550e8400-e29b-41d4-a716-446655440000';
    const input = [{ id: uuid }, { id: uuid }];
    const result = redactObject(input) as Array<Record<string, string>>;
    expect(result[0]!['id']).toBe('<uuid:1>');
    expect(result[1]!['id']).toBe('<uuid:1>');
  });

  it('is idempotent — running twice produces identical output', () => {
    const input = {
      token: 'Bearer abc.def.ghi',
      userId: '550e8400-e29b-41d4-a716-446655440000',
      createdAt: '2026-04-26T10:00:00Z',
    };
    const first = redactObject(input);
    const second = redactObject(first);
    expect(second).toEqual(first);
  });

  it('leaves null and undefined values unchanged', () => {
    const input = { a: null, b: undefined };
    const result = redactObject(input) as Record<string, unknown>;
    expect(result['a']).toBeNull();
    expect(result['b']).toBeUndefined();
  });
});
