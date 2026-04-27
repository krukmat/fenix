// BFF-ADMIN-10 / BFF-ADMIN-11 / BFF-ADMIN-12: workflows list, detail, activation
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus, upstreamMessage } from './adminAuth';

const ADMIN_ROOT = '/bff/admin';

interface WorkflowRow {
  id: string;
  name: string;
  status: string;
  version: number;
  description?: string | null;
  created_at: string;
  updated_at: string;
}

const STATUS_COLORS: Record<string, string> = { active: 'background:#d1fae5;color:#065f46', draft: 'background:#f3f4f6;color:#374151', testing: 'background:#dbeafe;color:#1e40af', archived: 'background:#fef3c7;color:#92400e' };
function statusBadge(status: string): string {
  const style = STATUS_COLORS[status] ?? 'background:#f3f4f6;color:#374151';
  return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:600;${style}">${escHtml(status)}</span>`;
}

function escHtml(s: string): string {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function renderRows(workflows: WorkflowRow[]): string {
  if (workflows.length === 0) {
    return `<tr><td colspan="5" style="padding:24px;text-align:center;color:var(--muted)">No workflows found</td></tr>`;
  }
  return workflows
    .map(
      (wf) => `
    <tr>
      <td><a href="/bff/admin/workflows/${escHtml(wf.id)}" style="color:var(--accent);text-decoration:none;font-weight:600">${escHtml(wf.name)}</a></td>
      <td>${statusBadge(wf.status)}</td>
      <td style="color:var(--muted)">${escHtml(String(wf.version))}</td>
      <td style="color:var(--muted);font-size:13px">${escHtml(wf.description ?? '—')}</td>
      <td style="color:var(--muted);font-size:12px">${escHtml(wf.updated_at.slice(0, 10))}</td>
    </tr>`,
    )
    .join('');
}

function buildBody(workflows: WorkflowRow[], status: string, name: string): string {
  return `
  <h2 class="page-title">Workflows</h2>
  <form method="GET" action="/bff/admin/workflows" style="display:flex;gap:10px;margin-bottom:16px;flex-wrap:wrap">
    <input name="name" type="text" value="${escHtml(name)}" placeholder="Filter by name"
      style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 10px;font-size:13px;color:var(--text)">
    <select name="status"
      style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 8px;font-size:13px;color:var(--text)">
      <option value="">All statuses</option>
      ${['active','draft','testing','archived'].map(
        (s) => `<option value="${s}"${status === s ? ' selected' : ''}>${s}</option>`,
      ).join('')}
    </select>
    <button type="submit" style="height:34px;padding:0 14px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Filter</button>
    <a href="/bff/admin/workflows" style="height:34px;line-height:34px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--muted);text-decoration:none">Clear</a>
  </form>
  <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
    <table style="width:100%;border-collapse:collapse;font-size:14px">
      <thead>
        <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Name</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Status</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Version</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Description</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Updated</th>
        </tr>
      </thead>
      <tbody id="workflows-tbody">${renderRows(workflows)}</tbody>
    </table>
  </div>`;
}
function extractListParams(q: Record<string, unknown>): { statusFilter: string; nameFilter: string; params: Record<string, string> } {
  const statusFilter = typeof q['status'] === 'string' ? q['status'] : '';
  const nameFilter   = typeof q['name']   === 'string' ? q['name']   : '';
  const params: Record<string, string> = {};
  if (statusFilter) params['status'] = statusFilter;
  if (nameFilter)   params['name']   = nameFilter;
  return { statusFilter, nameFilter, params };
}
const router = Router();
router.get('/', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { statusFilter, nameFilter, params } = extractListParams(req.query);

  try {
    const client = createGoClient(token);
    // BFF-ADMIN-Task6: Go returns envelope { data: [...] } — extract the array
    const { data: body } = await client.get<{ data: WorkflowRow[] }>('/api/v1/workflows', { params });
    res.type('html').status(200).send(adminLayout('Workflows', buildBody(body.data ?? [], statusFilter, nameFilter)));
  } catch (err: unknown) {
    const status = (err as { response?: { status?: number } }).response?.status;
    if (status === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});
// BFF-ADMIN-11: workflow detail page (read-only)
interface WorkflowDetail extends WorkflowRow {
  dsl_source: string;
  spec_source?: string | null;
  workspace_id?: string;
  parent_version_id?: string | null;
}

const MONO = 'ui-monospace,SFMono-Regular,Menlo,monospace';
function detailMeta(wf: WorkflowDetail): string {
  return `
  <div style="display:flex;align-items:center;gap:12px;margin-bottom:20px;flex-wrap:wrap">
    <a href="/bff/admin/workflows" style="color:var(--muted);font-size:13px;text-decoration:none">&larr; Workflows</a>
    <h2 class="page-title" style="margin:0">${escHtml(wf.name)}</h2>
    <span style="display:inline-block;padding:2px 10px;border-radius:999px;font-size:12px;font-weight:600;background:#f3f4f6;color:#374151">${escHtml(wf.status)}</span>
    <span style="color:var(--muted);font-size:13px">v${escHtml(String(wf.version))}</span>
  </div>
  <div style="display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));gap:1px;background:var(--line);border:1px solid var(--line);border-radius:8px;overflow:hidden;margin-bottom:20px">
    <div style="background:var(--panel);padding:14px"><span style="display:block;color:var(--muted);font-size:11px;font-weight:700;margin-bottom:4px">ID</span><p style="margin:0;font-size:13px;font-family:${MONO}">${escHtml(wf.id)}</p></div>
    <div style="background:var(--panel);padding:14px"><span style="display:block;color:var(--muted);font-size:11px;font-weight:700;margin-bottom:4px">Description</span><p style="margin:0;font-size:13px">${escHtml(wf.description ?? '—')}</p></div>
    <div style="background:var(--panel);padding:14px"><span style="display:block;color:var(--muted);font-size:11px;font-weight:700;margin-bottom:4px">Created</span><p style="margin:0;font-size:13px;color:var(--muted)">${escHtml(wf.created_at.slice(0, 10))}</p></div>
    <div style="background:var(--panel);padding:14px"><span style="display:block;color:var(--muted);font-size:11px;font-weight:700;margin-bottom:4px">Updated</span><p style="margin:0;font-size:13px;color:var(--muted)">${escHtml(wf.updated_at.slice(0, 10))}</p></div>
  </div>`;
}
function detailSources(wf: WorkflowDetail): string {
  const PRE = `style="margin:0;white-space:pre-wrap;word-break:break-all;font:13px/1.6 ${MONO};color:var(--text)"`;
  const specContent = wf.spec_source
    ? `<pre ${PRE}>${escHtml(wf.spec_source)}</pre>`
    : `<p style="color:var(--muted);font-size:13px;margin:0">No spec source</p>`;
  const panel = (title: string, content: string) =>
    `<div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden"><div style="padding:10px 14px;border-bottom:1px solid var(--line);background:var(--bg)"><h3 style="margin:0;font-size:13px;font-weight:700">${title}</h3></div><div style="padding:14px;overflow:auto;max-height:320px">${content}</div></div>`;
  return `<div style="display:grid;grid-template-columns:1fr 1fr;gap:16px;margin-bottom:20px">
    ${panel('DSL Source', `<pre ${PRE}>${escHtml(wf.dsl_source)}</pre>`)}
    ${panel('Spec Source', specContent)}
  </div>`;
}

function detailActivation(wf: WorkflowDetail): string {
  return `
  <div style="display:flex;gap:12px;align-items:center;margin-bottom:20px">
    <a href="/bff/builder" style="display:inline-block;padding:8px 16px;border:1px solid var(--line);border-radius:6px;font-size:13px;font-weight:600;color:var(--accent);text-decoration:none">Open in Builder</a>
  </div>
  <div id="activation-section" style="background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:16px">
    <h3 style="margin:0 0 8px;font-size:14px;font-weight:700">Activation</h3>
    <p style="margin:0 0 12px;color:var(--muted);font-size:13px">Current status: <strong>${escHtml(wf.status)}</strong>. Activate transitions the workflow through testing → active.</p>
    <form method="POST" action="/bff/admin/workflows/${escHtml(wf.id)}/activate">
      <button type="submit" style="height:34px;padding:0 16px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Activate</button>
    </form>
  </div>`;
}

function buildDetailBody(wf: WorkflowDetail): string { return detailMeta(wf) + detailSources(wf) + detailActivation(wf); }

router.get('/:id', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;

  try {
    const client = createGoClient(token);
    // BFF-ADMIN-Task6: Go returns envelope { data: {...} } — extract the workflow object
    const { data: resp } = await client.get<{ data: WorkflowDetail }>(`/api/v1/workflows/${id}`);
    res.type('html').status(200).send(adminLayout(`Workflow: ${resp.data.name}`, buildDetailBody(resp.data)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

// BFF-ADMIN-12: activate form submission — POST-Redirect-GET on success, re-render with error on 4xx
router.post('/:id/activate', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;
  const client = createGoClient(token);

  try {
    await client.put(`/api/v1/workflows/${id}/activate`);
    res.redirect(`/bff/admin/workflows/${id}`);
  } catch (err: unknown) {
    const status = upstreamStatus(err);
    if (status === 401) { res.redirect(ADMIN_ROOT); return; }
    // 4xx from backend (e.g. 422 invalid transition): re-render detail with inline error
    if (status !== undefined && status >= 400 && status < 500) {
      try {
        const { data: wfResp } = await client.get<{ data: WorkflowDetail }>(`/api/v1/workflows/${id}`);
        const wf = wfResp.data;
        const errorBanner = `
          <div style="margin-bottom:16px;padding:12px 16px;border:1px solid #fca5a5;background:#fef2f2;border-radius:6px;color:#991b1b;font-size:13px">
            <strong>Activation failed:</strong> ${escHtml(upstreamMessage(err))}
          </div>`;
        res.type('html').status(200).send(adminLayout(`Workflow: ${wf.name}`, errorBanner + buildDetailBody(wf)));
        return;
      } catch {
        // if the re-fetch also fails, fall through to next(err)
      }
    }
    next(err);
  }
});
export default router;
