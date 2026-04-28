// BFF-ADMIN-40 / BFF-ADMIN-41: audit trail paginated list + record detail
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus } from './adminAuth';
import { esc, outcomeBadge, buildDetailBody, type AuditDetail } from './adminAuditFragments';

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

interface AuditBackendResponse {
  data: AuditRow[];
  meta: { total?: number; nextCursor?: string };
}

interface AuditDetailEnvelope {
  data?: AuditDetail;
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

function extractAuditDetail(body: AuditDetail | AuditDetailEnvelope | undefined): AuditDetail | undefined {
  if (!body) return undefined;
  if ('data' in body && body.data) return body.data;
  return body as AuditDetail;
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
    const { data: resp } = await client.get<AuditDetail | AuditDetailEnvelope>(`/api/v1/audit/events/${id}`);
    const detail = extractAuditDetail(resp);
    if (!detail) {
      throw new Error('Audit detail response missing body');
    }
    res.type('html').status(200).send(adminLayout('Audit Event', buildDetailBody(detail)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

export default router;
