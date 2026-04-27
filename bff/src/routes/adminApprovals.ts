// BFF-ADMIN-30 / BFF-ADMIN-31: approvals queue list + decision form
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus, upstreamMessage } from './adminAuth';
import {
  PANEL,
  ApprovalDetail,
  buildProposedPayloadSection,
  buildReasoningTraceSection,
} from './adminApprovalsFragments';

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

function renderRows(rows: ApprovalRow[]): string {
  if (rows.length === 0) {
    return `<tr><td colspan="5" style="padding:24px;text-align:center;color:var(--muted)">No approvals found</td></tr>`;
  }
  return rows.map((a) => `
    <tr style="border-bottom:1px solid var(--line)">
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:12px;font-weight:600;color:var(--accent)">${esc(a.id)}</td>
      <td style="padding:10px 14px;font-size:13px;color:var(--muted)">${esc(a.requestedBy)}</td>
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:13px">${esc(a.action)}</td>
      <td style="padding:10px 14px">${statusBadge(a.status)}</td>
      <td style="padding:10px 14px;font-size:12px;color:var(--muted)">${esc(a.createdAt.slice(0, 16).replace('T', ' '))}</td>
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
        </tr>
      </thead>
      <tbody>${renderRows(rows)}</tbody>
    </table>
  </div>`;
}

function buildDecisionForm(approval: ApprovalDetail): string {
  const dt = (label: string, val: string) => `<dt style="color:var(--muted);font-weight:600">${label}</dt><dd style="margin:0;font-size:14px">${val}</dd>`;
  const mono = (v: string) => `<span style="font-family:ui-monospace,monospace;font-size:13px">${esc(v)}</span>`;
  const action = `POST /bff/admin/approvals/${esc(approval.id)}/decision`;
  const payloadSection = buildProposedPayloadSection(approval);
  const traceSection = buildReasoningTraceSection(approval);
  return `<div style="margin-bottom:20px"><a href="/bff/admin/approvals" style="font-size:13px;color:var(--muted);text-decoration:none">&larr; Approvals</a></div>
  <div style="${PANEL}">
    <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap;margin-bottom:16px">
      <h2 style="margin:0;font-size:18px">Approval Request</h2>${statusBadge(approval.status)}
    </div>
    <dl style="display:grid;grid-template-columns:140px 1fr;gap:8px 16px;margin:0">
      ${dt('ID', mono(approval.id))}${dt('Action', mono(approval.action))}${dt('Requested by', esc(approval.requestedBy))}${dt('Created', esc(approval.createdAt))}
    </dl>
  </div>
  ${payloadSection}
  ${traceSection}
  <div style="${PANEL}">
    <h3 style="margin:0 0 16px;font-size:16px;font-weight:700">Decision</h3>
    <form method="POST" action="${action}" style="display:flex;flex-direction:column;gap:14px;max-width:480px">
      <div>
        <label style="display:block;font-size:13px;font-weight:600;color:var(--muted);margin-bottom:6px">Reason (optional)</label>
        <textarea name="reason" rows="3" style="width:100%;border:1px solid var(--line);border-radius:6px;padding:10px;font-size:13px;color:var(--text);resize:vertical"></textarea>
      </div>
      <div style="display:flex;gap:10px">
        <button name="decision" value="approve" type="submit" style="padding:0 20px;height:36px;border:0;border-radius:6px;background:#065f46;color:#fff;font-size:13px;font-weight:700;cursor:pointer">approve</button>
        <button name="decision" value="reject" type="submit" style="padding:0 20px;height:36px;border:0;border-radius:6px;background:#991b1b;color:#fff;font-size:13px;font-weight:700;cursor:pointer">reject</button>
      </div>
    </form>
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

// BFF-ADMIN-31: approval detail + decision form
router.get('/:id', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;
  try {
    const client = createGoClient(token);
    const { data: approval } = await client.get<ApprovalDetail>(`/api/v1/approvals/${id}`);
    res.type('html').status(200).send(adminLayout('Approval', buildDecisionForm(approval)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

// BFF-ADMIN-31: relay decision to Go POST /api/v1/approvals/:id/decision
router.post('/:id/decision', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;
  const body = (req.body ?? {}) as { decision?: string; reason?: string };
  const { decision, reason } = body;
  try {
    const client = createGoClient(token);
    await client.post(`/api/v1/approvals/${id}/decision`, { decision, reason });
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
