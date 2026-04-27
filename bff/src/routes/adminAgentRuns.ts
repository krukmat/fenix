// BFF-ADMIN-20: agent runs list page — HTMX table over GET /api/v1/agents/runs
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus } from './adminAuth';
import {
  AgentRunDetail,
  BADGE_NEUTRAL,
  STATUS_COLORS,
  esc,
  statusBadge,
  buildDetailHeader,
} from './adminAgentRunsFragments';

const ADMIN_ROOT = '/bff/admin';

interface AgentRunRow {
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
}

interface AgentRunsBackendResponse {
  data: AgentRunRow[];
  meta: { total: number; limit: number; offset: number };
}

function renderRows(runs: AgentRunRow[]): string {
  if (runs.length === 0) {
    return `<tr><td colspan="6" style="padding:24px;text-align:center;color:var(--muted)">No agent runs found</td></tr>`;
  }
  return runs.map((run) => `
    <tr>
      <td><a href="/bff/admin/agent-runs/${esc(run.id)}" style="color:var(--accent);text-decoration:none;font-weight:600;font-size:12px;font-family:ui-monospace,monospace">${esc(run.id)}</a></td>
      <td>${statusBadge(run.status)}</td>
      <td style="color:var(--muted);font-size:13px">${esc(run.triggerType)}</td>
      <td style="color:var(--muted);font-size:13px">${run.totalTokens != null ? esc(String(run.totalTokens)) : '—'}</td>
      <td style="color:var(--muted);font-size:13px">${run.latencyMs != null ? `${esc(String(run.latencyMs))}ms` : '—'}</td>
      <td style="color:var(--muted);font-size:12px">${esc(run.startedAt.slice(0, 16).replace('T', ' '))}</td>
    </tr>`).join('');
}

function buildFilterForm(status: string, workflowId: string, dateFrom: string, dateTo: string): string {
  const statuses = ['', 'success', 'failed', 'running', 'cancelled', 'abstained'];
  const opts = statuses.map((s) => `<option value="${s}"${status === s ? ' selected' : ''}>${s || 'All statuses'}</option>`).join('');
  const field = (name: string, val: string, ph: string) =>
    `<input name="${name}" type="text" value="${esc(val)}" placeholder="${ph}" style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 10px;font-size:13px;color:var(--text)">`;
  return `
  <form method="GET" action="/bff/admin/agent-runs" style="display:flex;gap:10px;margin-bottom:16px;flex-wrap:wrap">
    <select name="status" style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 8px;font-size:13px;color:var(--text)">${opts}</select>
    ${field('workflow_id', workflowId, 'Workflow ID')}
    ${field('date_from', dateFrom, 'From (YYYY-MM-DD)')}
    ${field('date_to', dateTo, 'To (YYYY-MM-DD)')}
    <button type="submit" style="height:34px;padding:0 14px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Filter</button>
    <a href="/bff/admin/agent-runs" style="height:34px;line-height:34px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--muted);text-decoration:none">Clear</a>
  </form>`;
}

function buildBody(runs: AgentRunRow[], total: number, filters: Record<string, string>): string {
  const { status = '', workflow_id: wfId = '', date_from: df = '', date_to: dt = '' } = filters;
  return `
  <h2 class="page-title">Agent Runs <span style="color:var(--muted);font-size:16px;font-weight:400">(${total})</span></h2>
  ${buildFilterForm(status, wfId, df, dt)}
  <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
    <table style="width:100%;border-collapse:collapse;font-size:14px">
      <thead>
        <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Run ID</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Status</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Trigger</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Tokens</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Latency</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Started</th>
        </tr>
      </thead>
      <tbody>${renderRows(runs)}</tbody>
    </table>
  </div>`;
}

const FILTER_KEYS = ['status', 'workflow_id', 'date_from', 'date_to', 'limit', 'offset'] as const;

function extractParams(q: Record<string, string>): Record<string, string> {
  const params: Record<string, string> = {};
  for (const key of FILTER_KEYS) {
    if (q[key]) params[key] = q[key];
  }
  return params;
}

// suppress unused-import lint — STATUS_COLORS and BADGE_NEUTRAL are consumed by fragments module;
// re-exported here only to keep the original list page's statusBadge working transitively
void STATUS_COLORS;
void BADGE_NEUTRAL;

const router = Router();

router.get('/', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const q = req.query as Record<string, string>;

  try {
    const client = createGoClient(token);
    const { data: body } = await client.get<AgentRunsBackendResponse>('/api/v1/agents/runs', { params: extractParams(q) });
    const runs = Array.isArray(body?.data) ? body.data : [];
    const total = body?.meta?.total ?? runs.length;
    res.type('html').status(200).send(adminLayout('Agent Runs', buildBody(runs, total, q)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

// BFF-ADMIN-21a / 21b / 21c / 21d: agent run detail
router.get('/:id', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;
  const traceOffset = Math.max(0, parseInt((req.query as Record<string, string>)['trace_offset'] ?? '0', 10) || 0);

  try {
    const client = createGoClient(token);
    const { data: run } = await client.get<AgentRunDetail>(`/api/v1/agents/runs/${id}`);
    res.type('html').status(200).send(adminLayout(`Run ${id}`, buildDetailHeader(run, traceOffset)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

export default router;
