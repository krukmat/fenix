// BFF-ADMIN-40 / BFF-ADMIN-41: audit trail paginated list + record detail
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus } from './adminAuth';

const ADMIN_ROOT = '/bff/admin';

// BFF-ADMIN-Task6: fields match Go response (snake_case, entity_type/entity_id, actor_id)
interface AuditRow {
  id: string;
  actor_id: string;
  actor_type: string;
  action: string;
  entity_type: string;
  entity_id: string;
  outcome: string;
  created_at: string;
}

interface AuditDetail extends AuditRow {
  details?: Record<string, unknown>;
  permissions_checked?: Array<{ rule: string; result: string }>;
}

interface AuditBackendResponse {
  data: AuditRow[];
  meta: { total?: number; nextCursor?: string };
}

const OUTCOME_COLORS: Record<string, string> = {
  success: 'background:#d1fae5;color:#065f46',
  failure: 'background:#fee2e2;color:#991b1b',
  denied:  'background:#fef3c7;color:#92400e',
};

const BADGE_NEUTRAL = 'background:#f3f4f6;color:#374151';

function esc(s: string): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function outcomeBadge(outcome: string): string {
  const style = OUTCOME_COLORS[outcome] ?? BADGE_NEUTRAL;
  return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:600;${style}">${esc(outcome)}</span>`;
}

function renderRows(rows: AuditRow[]): string {
  if (rows.length === 0) {
    return `<tr><td colspan="6" style="padding:24px;text-align:center;color:var(--muted)">No audit events found</td></tr>`;
  }
  return rows.map((e) => `
    <tr style="border-bottom:1px solid var(--line)">
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:12px;font-weight:600;color:var(--accent)">
        <a href="/bff/admin/audit/${esc(e.id)}" style="color:var(--accent);text-decoration:none">${esc(e.id)}</a>
      </td>
      <td style="padding:10px 14px;font-size:13px;color:var(--muted)">${esc(e.actor_id)}</td>
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:13px">${esc(e.action)}</td>
      <td style="padding:10px 14px;font-size:13px;color:var(--muted)">${esc(e.entity_type)}</td>
      <td style="padding:10px 14px">${outcomeBadge(e.outcome)}</td>
      <td style="padding:10px 14px;font-size:12px;color:var(--muted)">${esc((e.created_at ?? '').slice(0, 16).replace('T', ' '))}</td>
    </tr>`).join('');
}

function buildFilterForm(params: Record<string, string>): string {
  const v = (k: string) => esc(params[k] ?? '');
  return `
  <form method="GET" action="/bff/admin/audit" style="display:flex;gap:10px;margin-bottom:16px;flex-wrap:wrap;align-items:flex-end">
    <div>
      <label style="display:block;font-size:12px;font-weight:600;color:var(--muted);margin-bottom:4px">Actor</label>
      <input name="actor" value="${v('actor')}" placeholder="user-id" style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 10px;font-size:13px;color:var(--text)">
    </div>
    <div>
      <label style="display:block;font-size:12px;font-weight:600;color:var(--muted);margin-bottom:4px">Resource type</label>
      <input name="resource_type" value="${v('resource_type')}" placeholder="case, approval…" style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 10px;font-size:13px;color:var(--text)">
    </div>
    <div>
      <label style="display:block;font-size:12px;font-weight:600;color:var(--muted);margin-bottom:4px">From</label>
      <input name="date_from" type="date" value="${v('date_from')}" style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 8px;font-size:13px;color:var(--text)">
    </div>
    <div>
      <label style="display:block;font-size:12px;font-weight:600;color:var(--muted);margin-bottom:4px">To</label>
      <input name="date_to" type="date" value="${v('date_to')}" style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 8px;font-size:13px;color:var(--text)">
    </div>
    <button type="submit" style="height:34px;padding:0 14px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Filter</button>
    <a href="/bff/admin/audit" style="height:34px;line-height:34px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--muted);text-decoration:none">Reset</a>
  </form>`;
}

function buildPagination(nextCursor: string | undefined, currentParams: Record<string, string>): string {
  if (!nextCursor) return '';
  const next = new URLSearchParams({ ...currentParams, cursor: nextCursor }).toString();
  return `
  <div style="margin-top:16px;display:flex;justify-content:flex-end">
    <a href="/bff/admin/audit?${next}" style="height:34px;line-height:34px;padding:0 16px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--accent);text-decoration:none;font-weight:600">Next &rarr;</a>
  </div>`;
}

function buildListBody(rows: AuditRow[], nextCursor: string | undefined, filterParams: Record<string, string>): string {
  return `
  <h2 class="page-title">Audit Trail</h2>
  ${buildFilterForm(filterParams)}
  <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
    <table style="width:100%;border-collapse:collapse;font-size:14px">
      <thead>
        <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">ID</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Actor</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Action</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Resource</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Outcome</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Timestamp</th>
        </tr>
      </thead>
      <tbody>${renderRows(rows)}</tbody>
    </table>
  </div>
  ${buildPagination(nextCursor, filterParams)}`;
}

function buildDetailBody(e: AuditDetail): string {
  const PANEL = 'background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:24px;margin-bottom:20px';
  const dt = (label: string, val: string) =>
    `<dt style="color:var(--muted);font-weight:600;font-size:13px">${label}</dt><dd style="margin:0;font-size:14px">${val}</dd>`;
  const mono = (v: string) =>
    `<span style="font-family:ui-monospace,monospace;font-size:13px">${esc(v)}</span>`;

  const policyRows = (e.permissions_checked ?? []).map((p) =>
    `<tr style="border-bottom:1px solid var(--line)">
      <td style="padding:8px 12px;font-family:ui-monospace,monospace;font-size:12px">${esc(p.rule)}</td>
      <td style="padding:8px 12px">${outcomeBadge(p.result)}</td>
    </tr>`
  ).join('');

  const policySection = e.permissions_checked && e.permissions_checked.length > 0 ? `
  <div style="${PANEL}">
    <h3 style="margin:0 0 16px;font-size:16px;font-weight:700">Policy Checks</h3>
    <table style="width:100%;border-collapse:collapse;font-size:13px">
      <thead>
        <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
          <th style="padding:8px 12px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Rule</th>
          <th style="padding:8px 12px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Result</th>
        </tr>
      </thead>
      <tbody>${policyRows}</tbody>
    </table>
  </div>` : '';

  return `
  <div style="margin-bottom:20px">
    <a href="/bff/admin/audit" style="font-size:13px;color:var(--muted);text-decoration:none">&larr; Audit Trail</a>
  </div>
  <div style="${PANEL}">
    <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap;margin-bottom:16px">
      <h2 style="margin:0;font-size:18px">Audit Event</h2>${outcomeBadge(e.outcome)}
    </div>
    <dl style="display:grid;grid-template-columns:140px 1fr;gap:8px 16px;margin:0">
      ${dt('ID', mono(e.id))}
      ${dt('Actor', esc(e.actor_id))}
      ${dt('Action', mono(e.action))}
      ${dt('Resource type', esc(e.entity_type))}
      ${dt('Resource ID', mono(e.entity_id))}
      ${dt('Timestamp', esc(e.created_at))}
    </dl>
  </div>
  ${policySection}`;
}

function extractAuditParams(q: Record<string, string>): { params: Record<string, string>; filterParams: Record<string, string> } {
  const filterParams: Record<string, string> = {};
  if (q['actor'])         filterParams['actor']         = q['actor'];
  if (q['resource_type']) filterParams['resource_type'] = q['resource_type'];
  if (q['resource_id'])   filterParams['resource_id']   = q['resource_id'];
  if (q['date_from'])     filterParams['date_from']     = q['date_from'];
  if (q['date_to'])       filterParams['date_to']       = q['date_to'];

  const params: Record<string, string> = { ...filterParams };
  if (q['cursor']) params['cursor'] = q['cursor'];
  if (q['limit'])  params['limit']  = q['limit'];
  return { params, filterParams };
}

const router = Router();

// BFF-ADMIN-40: paginated audit list with filters
router.get('/', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { params, filterParams } = extractAuditParams(req.query as Record<string, string>);

  try {
    const client = createGoClient(token);
    // BFF-ADMIN-Task6: Go route is /audit/events, not /audit
    const { data: body } = await client.get<AuditBackendResponse>('/api/v1/audit/events', { params });
    const rows = Array.isArray(body?.data) ? body.data : [];
    const nextCursor = body?.meta?.nextCursor;
    res.type('html').status(200).send(adminLayout('Audit Trail', buildListBody(rows, nextCursor, filterParams)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

// BFF-ADMIN-41: immutable audit record detail (read-only)
router.get('/:id', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;

  try {
    const client = createGoClient(token);
    // BFF-ADMIN-Task6: Go route is /audit/events/{id}, not /audit/{id}
    const { data: event } = await client.get<AuditDetail>(`/api/v1/audit/events/${id}`);
    res.type('html').status(200).send(adminLayout('Audit Event', buildDetailBody(event)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

export default router;
