export type UnknownRecord = Record<string, unknown>;

export function asRecord(value: unknown): UnknownRecord | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as UnknownRecord)
    : null;
}

export function readString(record: UnknownRecord | null, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'string' && value.trim() !== '') {
      return value.trim();
    }
  }
  return undefined;
}

export function readNumber(record: UnknownRecord | null, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'number' && Number.isFinite(value)) {
      return value;
    }
  }
  return undefined;
}

export function readBoolean(record: UnknownRecord | null, ...keys: string[]): boolean | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'boolean') {
      return value;
    }
  }
  return undefined;
}

export function readStringArray(record: UnknownRecord | null, ...keys: string[]): string[] | undefined {
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
