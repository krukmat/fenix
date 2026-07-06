export type RecordLike = Record<string, unknown>;

export function readString(record: RecordLike | null | undefined, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record?.[key];
    if (typeof value === 'string' && value.length > 0) {
      return value;
    }
  }
  return undefined;
}
