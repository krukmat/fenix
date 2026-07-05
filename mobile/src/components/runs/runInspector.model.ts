import type { UsageEvent } from '../../services/api.types';

export type {
  ApprovalLinkage,
  NormalizedToolCall,
  RunEvidenceItem,
  RunEvidenceMeta,
  RunInspectorDetail,
  RunInspectorProps,
} from './runInspector.types';

export {
  extractApprovalLinkage,
  extractEvidenceMeta,
  normalizeEvidenceItems,
  normalizeReasoningTrace,
  normalizeToolCalls,
} from './runInspector.normalize';

export function formatLabel(value: string): string {
  return value
    .split(/[_-]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function formatMoney(value?: number): string {
  return typeof value === 'number' ? `€${value.toFixed(4)}` : '—';
}

export function formatMs(value?: number): string {
  if (typeof value !== 'number' || Number.isNaN(value)) return '—';
  if (value < 1000) return `${value} ms`;
  return `${(value / 1000).toFixed(1)} s`;
}

export function formatTimestamp(value?: string): string {
  if (!value) return '—';
  const parsed = new Date(value);
  return Number.isNaN(parsed.getTime()) ? value : parsed.toISOString();
}

export function formatJson(value: unknown): string {
  if (value === null || value === undefined) return '—';
  if (typeof value === 'string') return value;
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function sumUsage(events: UsageEvent[] | undefined) {
  return (events ?? []).reduce(
    (acc, event) => ({
      inputUnits: acc.inputUnits + (event.inputUnits ?? 0),
      outputUnits: acc.outputUnits + (event.outputUnits ?? 0),
      estimatedCost: acc.estimatedCost + (event.estimatedCost ?? 0),
      latencyMs: acc.latencyMs + (event.latencyMs ?? 0),
    }),
    { inputUnits: 0, outputUnits: 0, estimatedCost: 0, latencyMs: 0 },
  );
}
