// bff-http-snapshots T4: deterministic redaction — stable output across runs

const UUID_RE = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi;
const BEARER_RE = /^Bearer\s+\S+$/;
const TIMESTAMP_RE = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/;
const PRIVATE_SNAPSHOT_IP_RE = /^10\.244\.\d{1,3}\.\d{1,3}$/;
const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const SEED_SUFFIX_RE = /\d{8}T\d{6}/g;
const EMBEDDED_TIMESTAMP_RE = /\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})/g;
const RFC1123_TIMESTAMP_RE = /(?:Mon|Tue|Wed|Thu|Fri|Sat|Sun), \d{2} (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) \d{4} \d{2}:\d{2}:\d{2} GMT/g;

// Keys whose values are always fully redacted regardless of format
const SENSITIVE_KEYS = new Set(['password', 'token', 'apiKey', 'api_key', 'secret', 'accessToken']);

// Already-redacted placeholder patterns — idempotency guard
const ALREADY_REDACTED_RE = /^(<REDACTED>|<uuid:\d+>|<timestamp>|Bearer <REDACTED>)$/;

type UUIDMap = Record<string, string>;

export function redactValue(value: unknown, uuidMap: UUIDMap = {}): unknown {
  if (typeof value !== 'string') return value;
  if (ALREADY_REDACTED_RE.test(value)) return value;
  if (BEARER_RE.test(value)) return 'Bearer <REDACTED>';
  if (TIMESTAMP_RE.test(value)) return '<timestamp>';
  if (PRIVATE_SNAPSHOT_IP_RE.test(value)) return '<snapshot-ip>';
  if (EMAIL_RE.test(value)) return '<email>';

  // Replace all UUIDs in the string with deterministic placeholders
  return redactSnapshotText(value).replace(UUID_RE, (match) => {
    const key = match.toLowerCase();
    if (!uuidMap[key]) {
      const next = Object.keys(uuidMap).length + 1;
      uuidMap[key] = `<uuid:${next}>`;
    }
    return uuidMap[key]!;
  });
}

export function redactObject(input: unknown, uuidMap: UUIDMap = {}): unknown {
  if (input === null || input === undefined) return input;

  if (Array.isArray(input)) {
    return input.map((item) => redactObject(item, uuidMap));
  }

  if (typeof input === 'object') {
    const result: Record<string, unknown> = {};
    for (const [key, val] of Object.entries(input as Record<string, unknown>)) {
      if (SENSITIVE_KEYS.has(key)) {
        result[key] = val === null || val === undefined ? val : '<REDACTED>';
      } else if (key === 'latency_ms') {
        result[key] = '<duration>';
      } else {
        result[key] = redactObject(val, uuidMap);
      }
    }
    return result;
  }

  return redactValue(input, uuidMap);
}

export function redactSnapshotText(raw: string): string {
  if (raw.startsWith('# HELP bff_requests_total')) {
    return raw
      .replace(/bff_requests_total \d+/g, 'bff_requests_total <count>')
      .replace(/bff_request_errors_total \d+/g, 'bff_request_errors_total <count>')
      .replace(/bff_uptime_seconds [0-9.]+/g, 'bff_uptime_seconds <duration>');
  }
  return raw
    .replace(RFC1123_TIMESTAMP_RE, '<timestamp>')
    .replace(EMBEDDED_TIMESTAMP_RE, '<timestamp>')
    .replace(SEED_SUFFIX_RE, '<seed-suffix>');
}
