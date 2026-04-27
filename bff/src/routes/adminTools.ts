// BFF-ADMIN-60: tools list
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus } from './adminAuth';

const ADMIN_ROOT = '/bff/admin';

interface Tool {
  id: string;
  name: string;
  description: string;
  active: boolean;
  createdAt: string;
}

interface ToolsBackendResponse {
  data: Tool[];
}

const BADGE_NEUTRAL = 'background:#f3f4f6;color:#374151';

function esc(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function badge(label: string, style: string): string {
  return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:600;${style}">${esc(label)}</span>`;
}

function activeBadge(active: boolean): string {
  return active
    ? badge('active',   'background:#d1fae5;color:#065f46')
    : badge('inactive', BADGE_NEUTRAL);
}

function renderRows(rows: Tool[]): string {
  if (rows.length === 0) {
    return `<tr><td colspan="5" style="padding:24px;text-align:center;color:var(--muted)">No tools</td></tr>`;
  }
  return rows.map((t) => `
    <tr style="border-bottom:1px solid var(--line)">
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:12px;font-weight:600;color:var(--accent)">${esc(t.id)}</td>
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:13px">${esc(t.name)}</td>
      <td style="padding:10px 14px;font-size:13px;color:var(--muted)">${esc(t.description)}</td>
      <td style="padding:10px 14px">${activeBadge(t.active)}</td>
      <td style="padding:10px 14px;font-size:12px;color:var(--muted)">${esc(t.createdAt.slice(0, 16).replace('T', ' '))}</td>
    </tr>`).join('');
}

function buildBody(rows: Tool[]): string {
  return `
  <h2 class="page-title">Tools</h2>
  <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
    <table style="width:100%;border-collapse:collapse;font-size:14px">
      <thead>
        <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">ID</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Name</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Description</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">State</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Created</th>
        </tr>
      </thead>
      <tbody>${renderRows(rows)}</tbody>
    </table>
  </div>`;
}

const router = Router();

router.get('/', async (_req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  try {
    const client = createGoClient(token);
    const { data: resp } = await client.get<ToolsBackendResponse>('/api/v1/admin/tools');
    const tools = Array.isArray(resp?.data) ? resp.data : [];
    res.type('html').status(200).send(adminLayout('Tools', buildBody(tools)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

export default router;
