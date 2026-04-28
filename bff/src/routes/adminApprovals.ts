// BFF-ADMIN-30 / BFF-ADMIN-31: approvals queue with inline decision actions
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus, upstreamMessage } from './adminAuth';

const ADMIN_ROOT = '/bff/admin';

interface ApprovalRow {
  id: string;
  requestedBy: string;
  action: string;
  status: string;
  reason?: string | null;
  createdAt: string;
}

interface ApprovalsBackendResponse {
  data: ApprovalRow[];
  meta: { total: number; limit: number; offset: number };
}

const BADGE_NEUTRAL = 'background:#f3f4f6;color:#374151';

const STATUS_COLORS: Record<string, string> = {
  pending:  'background:#fef3c7;color:#92400e',
  approved: 'background:#d1fae5;color:#065f46',
  rejected: 'background:#fee2e2;color:#991b1b',
};

function esc(s: string): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function statusBadge(status: string): string {
  const style = STATUS_COLORS[status] ?? BADGE_NEUTRAL;
  return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:600;${style}">${esc(status)}</span>`;
}

function renderDecisionControls(approval: ApprovalRow): string {
  if (approval.status !== 'pending') {
    return `<span style="font-size:12px;color:var(--muted)">Already decided</span>`;
  }

  const action = `/bff/admin/approvals/${encodeURIComponent(approval.id)}/decision`;
  return `<form method="POST" action="${action}" style="display:flex;flex-direction:column;gap:8px;min-width:220px">
    <label style="display:block;font-size:12px;font-weight:600;color:var(--muted)">
      Reason
      <textarea name="reason" rows="2" style="margin-top:4px;width:100%;border:1px solid var(--line);border-radius:6px;padding:8px;font-size:12px;color:var(--text);resize:vertical"></textarea>
    </label>
    <div style="display:flex;gap:8px">
      <button name="decision" value="approve" type="submit" style="padding:0 12px;height:32px;border:0;border-radius:6px;background:#065f46;color:#fff;font-size:12px;font-weight:700;cursor:pointer">approve</button>
      <button name="decision" value="reject" type="submit" style="padding:0 12px;height:32px;border:0;border-radius:6px;background:#991b1b;color:#fff;font-size:12px;font-weight:700;cursor:pointer">reject</button>
    </div>
  </form>`;
}

function renderRows(rows: ApprovalRow[]): string {
  if (rows.length === 0) {
    return `<tr><td colspan="6" style="padding:24px;text-align:center;color:var(--muted)">No approvals found</td></tr>`;
  }
  return rows.map((a) => `
    <tr style="border-bottom:1px solid var(--line)">
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:12px;font-weight:600;color:var(--accent)">${esc(a.id)}</td>
      <td style="padding:10px 14px;font-size:13px;color:var(--muted)">${esc(a.requestedBy)}</td>
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:13px">${esc(a.action)}</td>
      <td style="padding:10px 14px">${statusBadge(a.status)}</td>
      <td style="padding:10px 14px;font-size:12px;color:var(--muted)">${esc(a.createdAt.slice(0, 16).replace('T', ' '))}</td>
      <td style="padding:10px 14px">${renderDecisionControls(a)}</td>
    </tr>`).join('');
}

function buildFilterForm(currentStatus: string): string {
  const statuses = ['pending', 'approved', 'rejected', ''];
  const opts = statuses.map((s) =>
    `<option value="${s}"${currentStatus === s ? ' selected' : ''}>${s || 'All statuses'}</option>`
  ).join('');
  return `
  <form method="GET" action="/bff/admin/approvals" style="display:flex;gap:10px;margin-bottom:16px;flex-wrap:wrap">
    <select name="status" style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 8px;font-size:13px;color:var(--text)">${opts}</select>
    <button type="submit" style="height:34px;padding:0 14px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Filter</button>
    <a href="/bff/admin/approvals" style="height:34px;line-height:34px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--muted);text-decoration:none">Reset</a>
  </form>`;
}

function buildBody(rows: ApprovalRow[], total: number, currentStatus: string): string {
  return `
  <h2 class="page-title">Approvals <span style="color:var(--muted);font-size:16px;font-weight:400">(${total})</span></h2>
  <p style="margin:0 0 16px;color:var(--muted);font-size:13px;line-height:1.5">
    The approvals contract is queue-first: operators review pending requests and decide them directly from this page.
    There is no separate approval detail route because the backend exposes only list and decide endpoints.
  </p>
  ${buildFilterForm(currentStatus)}
  <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
    <table style="width:100%;border-collapse:collapse;font-size:14px">
      <thead>
        <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">ID</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Requested By</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Action</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Status</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Created</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Decision</th>
        </tr>
      </thead>
      <tbody>${renderRows(rows)}</tbody>
    </table>
  </div>`;
}

function extractParams(q: Record<string, string>): { params: Record<string, string>; status: string } {
  // default to pending — approvals list is a work queue, not an archive
  const status = q['status'] ?? 'pending';
  const params: Record<string, string> = { status };
  if (q['limit'])  params['limit']  = q['limit'];
  if (q['offset']) params['offset'] = q['offset'];
  return { params, status };
}

const router = Router();

router.get('/', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { params, status } = extractParams(req.query as Record<string, string>);

  try {
    const client = createGoClient(token);
    const { data: body } = await client.get<ApprovalsBackendResponse>('/api/v1/approvals', { params });
    const rows = Array.isArray(body?.data) ? body.data : [];
    const total = body?.meta?.total ?? rows.length;
    res.type('html').status(200).send(adminLayout('Approvals', buildBody(rows, total, status)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

// BFF-ADMIN-31: relay queue decision to Go PUT /api/v1/approvals/:id
router.post('/:id/decision', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;
  const body = (req.body ?? {}) as { decision?: string; reason?: string };
  const { decision, reason } = body;
  try {
    const client = createGoClient(token);
    await client.put(`/api/v1/approvals/${id}`, { decision, reason });
    res.redirect('/bff/admin/approvals');
  } catch (err: unknown) {
    const st = upstreamStatus(err);
    if (st === 401) { res.redirect(ADMIN_ROOT); return; }
    if (st === 422) {
      const msg = upstreamMessage(err);
      res.type('html').status(422).send(adminLayout('Decision Error', `<p style="color:#991b1b;font-size:14px">${esc(msg)}</p>`));
      return;
    }
    next(err);
  }
});

export default router;
