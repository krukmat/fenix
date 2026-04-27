// BFF-ADMIN-21b / BFF-ADMIN-21c / BFF-ADMIN-21d: detail page fragment builders
export interface TraceStep {
  step: number;
  thought: string;
}

export interface EvidenceItem {
  id: string;
  sourceId: string;
  snippet: string;
  score: number;
  confidence: string;
  timestamp: string;
}

export interface ToolCall {
  toolName: string;
  status: string;
  latencyMs?: number | null;
  idempotencyKey?: string | null;
  input?: unknown;
  output?: unknown;
}

export interface AgentRunDetail {
  id: string;
  agentDefinitionId: string;
  triggerType: string;
  status: string;
  totalTokens?: number | null;
  totalCost?: number | null;
  latencyMs?: number | null;
  startedAt: string;
  completedAt?: string | null;
  createdAt: string;
  outcome?: string | null;
  abstainReason?: string | null;
  reasoningTrace?: TraceStep[] | null;
  retrievedEvidence?: EvidenceItem[] | null;
  toolCalls?: ToolCall[] | null;
  costTokens?: number | null;
  costEuros?: number | null;
}

export const PANEL_CARD = 'background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:24px;margin-bottom:20px';
export const BADGE_NEUTRAL = 'background:#f3f4f6;color:#374151';

export const STATUS_COLORS: Record<string, string> = {
  success:   'background:#d1fae5;color:#065f46',
  failed:    'background:#fee2e2;color:#991b1b',
  running:   'background:#dbeafe;color:#1e40af',
  cancelled: 'background:#fef3c7;color:#92400e',
  abstained: BADGE_NEUTRAL,
};

const CONFIDENCE_COLORS: Record<string, string> = {
  high:   'background:#d1fae5;color:#065f46',
  medium: 'background:#fef3c7;color:#92400e',
  low:    'background:#fee2e2;color:#991b1b',
};

export const TRACE_PAGE_SIZE = 10;

export function esc(s: string): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

export function statusBadge(status: string): string {
  const style = STATUS_COLORS[status] ?? BADGE_NEUTRAL;
  return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:600;${style}">${esc(status)}</span>`;
}

// BFF-ADMIN-21c
export function buildEvidenceFragment(run: AgentRunDetail): string {
  const ev = Array.isArray(run.retrievedEvidence) ? run.retrievedEvidence : [];
  const wrap = (inner: string) => `<div style="${PANEL_CARD}">${inner}</div>`;
  const h = `<h3 style="margin:0 0 16px;font-size:16px;font-weight:700">Evidence</h3>`;
  if (ev.length === 0) return wrap(h + `<p style="color:var(--muted);font-size:14px;margin:0">No evidence</p>`);
  const items = ev.map((e) => {
    const cStyle = CONFIDENCE_COLORS[e.confidence] ?? BADGE_NEUTRAL;
    const badge = `<span style="display:inline-block;padding:1px 7px;border-radius:999px;font-size:11px;font-weight:700;${cStyle}">${esc(e.confidence)}</span>`;
    return `<li style="padding:12px 0;border-bottom:1px solid var(--line)"><div style="display:flex;gap:10px;align-items:center;margin-bottom:6px"><span style="font-family:ui-monospace,monospace;font-size:12px;color:var(--accent)">${esc(e.sourceId)}</span>${badge}<span style="margin-left:auto;font-size:12px;color:var(--muted)">${esc(e.score.toFixed(2))}</span><span style="font-size:12px;color:var(--muted)">${esc(e.timestamp.slice(0, 10))}</span></div><p style="margin:0;font-size:13px;color:var(--text);line-height:1.5">${esc(e.snippet)}</p></li>`;
  }).join('');
  return wrap(h + `<ul style="margin:0;padding:0;list-style:none">${items}</ul>`);
}

// BFF-ADMIN-21d
export function buildToolCallsFragment(run: AgentRunDetail): string {
  const calls = Array.isArray(run.toolCalls) ? run.toolCalls : [];
  const wrap = (inner: string) => `<div style="${PANEL_CARD}">${inner}</div>`;
  const h = `<h3 style="margin:0 0 16px;font-size:16px;font-weight:700">Tool Calls</h3>`;
  if (calls.length === 0) return wrap(h + `<p style="color:var(--muted);font-size:14px;margin:0">No tool calls</p>`);
  const rows = calls.map((c) => {
    const s = STATUS_COLORS[c.status] ?? BADGE_NEUTRAL;
    const badge = `<span style="display:inline-block;padding:1px 7px;border-radius:999px;font-size:11px;font-weight:700;${s}">${esc(c.status)}</span>`;
    const latency = c.latencyMs != null ? `${esc(String(c.latencyMs))}ms` : '—';
    const ikey = c.idempotencyKey ? `<span style="font-family:ui-monospace,monospace;font-size:11px;color:var(--muted)">${esc(c.idempotencyKey)}</span>` : '—';
    return `<tr style="border-bottom:1px solid var(--line)">
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:13px;font-weight:600;color:var(--accent)">${esc(c.toolName)}</td>
      <td style="padding:10px 14px">${badge}</td>
      <td style="padding:10px 14px;font-size:13px;color:var(--muted)">${latency}</td>
      <td style="padding:10px 14px;font-size:13px">${ikey}</td>
    </tr>`;
  }).join('');
  return wrap(h + `<table style="width:100%;border-collapse:collapse;font-size:14px">
    <thead><tr style="border-bottom:1px solid var(--line);background:var(--bg)">
      <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Tool</th>
      <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Status</th>
      <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Latency</th>
      <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Idempotency Key</th>
    </tr></thead>
    <tbody>${rows}</tbody>
  </table>`);
}

// BFF-ADMIN-21d
export function buildCostPanel(run: AgentRunDetail): string {
  const wrap = (inner: string) => `<div style="${PANEL_CARD}">${inner}</div>`;
  const h = `<h3 style="margin:0 0 16px;font-size:16px;font-weight:700">Cost</h3>`;
  const tokens = run.costTokens != null ? esc(String(run.costTokens)) : '—';
  const euros  = run.costEuros  != null ? `€${esc(run.costEuros.toFixed(4))}` : '—';
  const card = (label: string, val: string) =>
    `<div style="background:var(--bg);border:1px solid var(--line);border-radius:8px;padding:16px 20px;min-width:140px"><div style="font-size:12px;font-weight:600;color:var(--muted);margin-bottom:6px">${label}</div><div style="font-size:22px;font-weight:700;color:var(--text)">${val}</div></div>`;
  return wrap(h + `<div style="display:flex;gap:16px;flex-wrap:wrap">${card('Tokens', tokens)}${card('Cost (€)', euros)}</div>`);
}

// BFF-ADMIN-21b
export function buildTraceFragment(run: AgentRunDetail, traceOffset: number): string {
  const trace = Array.isArray(run.reasoningTrace) ? run.reasoningTrace : [];
  const wrap = (inner: string) => `<div style="${PANEL_CARD}">${inner}</div>`;
  const heading = (suffix = '') => `<h3 style="margin:0 0 16px;font-size:16px;font-weight:700">Reasoning Trace${suffix}</h3>`;
  if (trace.length === 0) return wrap(heading() + `<p style="color:var(--muted);font-size:14px;margin:0">No trace available</p>`);
  const page = trace.slice(traceOffset, traceOffset + TRACE_PAGE_SIZE);
  const steps = page.map((s) =>
    `<li style="padding:10px 0;border-bottom:1px solid var(--line);display:flex;gap:12px;align-items:flex-start"><span style="flex-shrink:0;min-width:52px;font-size:12px;font-weight:700;color:var(--muted)">Step ${s.step + 1}</span><span style="font-size:14px;color:var(--text);line-height:1.5">${esc(s.thought)}</span></li>`
  ).join('');
  const prev = traceOffset > 0 ? `<a href="?trace_offset=${Math.max(0, traceOffset - TRACE_PAGE_SIZE)}" style="font-size:13px;color:var(--accent);text-decoration:none">&larr; Prev</a>` : '';
  const nextOff = traceOffset + TRACE_PAGE_SIZE;
  const next = nextOff < trace.length ? `<a href="?trace_offset=${nextOff}" style="font-size:13px;color:var(--accent);text-decoration:none">Next &rarr;</a>` : '';
  const pager = (prev || next) ? `<div style="display:flex;justify-content:space-between;margin-top:12px">${prev}<span></span>${next}</div>` : '';
  return wrap(heading(` <span style="color:var(--muted);font-size:13px;font-weight:400">(${trace.length} steps)</span>`) + `<ol style="margin:0;padding:0;list-style:none">${steps}</ol>${pager}`);
}

export function buildDetailHeader(run: AgentRunDetail, traceOffset: number = 0): string {
  const dt = (label: string, val: string) => `<dt style="color:var(--muted);font-weight:600">${label}</dt><dd style="margin:0">${val}</dd>`;
  const mono = (v: string) => `<span style="font-family:ui-monospace,monospace;font-size:13px">${esc(v)}</span>`;
  const abstain = run.abstainReason ? `<div style="margin-top:12px;padding:12px 16px;background:#fef3c7;border:1px solid #fde68a;border-radius:6px;font-size:13px;color:#92400e"><strong>Abstain reason:</strong> ${esc(run.abstainReason)}</div>` : '';
  const outcome = run.outcome ? `<p style="margin:8px 0 0;font-size:14px;color:var(--text)">${esc(run.outcome)}</p>` : '';
  return `<div style="margin-bottom:20px"><a href="/bff/admin/agent-runs" style="font-size:13px;color:var(--muted);text-decoration:none">&larr; Agent Runs</a></div>
  <div style="${PANEL_CARD}">
    <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap;margin-bottom:16px"><h2 class="page-title" style="margin:0;font-size:18px">Run Detail</h2>${statusBadge(run.status)}</div>
    <dl style="display:grid;grid-template-columns:160px 1fr;gap:8px 16px;font-size:14px;margin:0">
      ${dt('Run ID', mono(run.id))}${dt('Agent', mono(run.agentDefinitionId))}${dt('Trigger', esc(run.triggerType))}
      ${dt('Started', esc(run.startedAt))}${dt('Completed', run.completedAt ? esc(run.completedAt) : '—')}${dt('Latency', run.latencyMs != null ? `${esc(String(run.latencyMs))} ms` : '—')}
    </dl>${outcome}${abstain}
  </div>
  ${buildToolCallsFragment(run)}
  ${buildCostPanel(run)}
  ${buildEvidenceFragment(run)}
  ${buildTraceFragment(run, traceOffset)}`;
}
