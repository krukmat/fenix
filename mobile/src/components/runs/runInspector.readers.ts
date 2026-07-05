export function asRecord(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : null;
}

export function readString(record: Record<string, unknown> | null, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'string' && value.trim() !== '') {
      return value.trim();
    }
  }
  return undefined;
}

export function readNumber(record: Record<string, unknown> | null, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
}

export function readStringArray(record: Record<string, unknown> | null, ...keys: string[]): string[] | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (!Array.isArray(value)) continue;
    const items = value.filter((item): item is string => typeof item === 'string' && item.trim() !== '');
    if (items.length > 0) {
      return items.map((item) => item.trim());
    }
  }
  return undefined;
}
