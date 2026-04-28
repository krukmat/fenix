// BFF-ADMIN-Task6: workflow HTML fragments (list + detail) — extracted to meet max-lines gate

export interface WorkflowRow {
  id: string;
  name: string;
  status: string;
  version: number;
  description?: string | null;
  created_at: string;
  updated_at: string;
}

export interface WorkflowDraftForm {
  name: string;
  description: string;
  authoringMode: string;
}

export interface WorkflowCreateResponse {
  id: string;
  name: string;
}

const STATUS_COLORS: Record<string, string> = { active: 'background:#d1fae5;color:#065f46', draft: 'background:#f3f4f6;color:#374151', testing: 'background:#dbeafe;color:#1e40af', archived: 'background:#fef3c7;color:#92400e' };
const STATUS_HINTS: Record<string, string> = {
  draft: 'Draft: editable in builder and not yet ready for activation.',
  testing: 'Testing: validate behavior before promoting this workflow to active.',
  active: 'Active: currently promoted. Create or edit a draft version before changing behavior.',
  archived: 'Archived: retained for history and not part of the active operator flow.',
};

function statusBadge(status: string): string {
  const style = STATUS_COLORS[status] ?? 'background:#f3f4f6;color:#374151';
  return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:600;${style}">${escHtml(status)}</span>`;
}

export function formValue(body: Record<string, unknown>, key: string): string {
  const value = body[key];
  return typeof value === 'string' ? value.trim() : '';
}

export function draftFormState(body: Record<string, unknown> = {}): WorkflowDraftForm {
  return {
    name: formValue(body, 'name'),
    description: formValue(body, 'description'),
    authoringMode: formValue(body, 'authoring_mode') || 'visual',
  };
}

export function draftCreatePayload(form: WorkflowDraftForm): { name: string; description?: string; dsl_source: string } {
  const workflowName = form.name || 'workflow_draft';
  return {
    name: form.name,
    ...(form.description ? { description: form.description } : {}),
    dsl_source: `WORKFLOW ${workflowName}\nON case.created`,
  };
}

export function buildNewDraftBody(form: WorkflowDraftForm, errorMessage?: string): string {
  const error = errorMessage ? `
  <div style="margin-bottom:16px;padding:12px 16px;border:1px solid #fca5a5;background:#fef2f2;border-radius:6px;color:#991b1b;font-size:13px">
    <strong>Create draft failed:</strong> ${escHtml(errorMessage)}
  </div>` : '';
  return `
  <div style="display:flex;align-items:center;gap:12px;margin-bottom:20px;flex-wrap:wrap">
    <a href="/bff/admin/workflows" style="color:var(--muted);font-size:13px;text-decoration:none">&larr; Workflows</a>
    <h2 class="page-title" style="margin:0">Create workflow draft</h2>
  </div>
  ${error}
  <div style="max-width:720px;background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
    <div style="padding:12px 16px;border-bottom:1px solid var(--line);background:var(--bg)">
      <h3 style="margin:0;font-size:13px;font-weight:700">Draft scaffold</h3>
      <p style="margin:6px 0 0;color:var(--muted);font-size:13px">Creates a real draft workflow and opens the visual builder with that workflow bound.</p>
    </div>
    <form method="POST" action="/bff/admin/workflows" style="padding:16px;display:grid;gap:14px">
      <label style="display:grid;gap:6px">
        <span style="font-size:12px;font-weight:700;color:var(--muted)">Name</span>
        <input name="name" type="text" required value="${escHtml(form.name)}" placeholder="sales_followup"
          style="height:36px;border:1px solid var(--line);border-radius:6px;padding:0 10px;font-size:13px;color:var(--text)">
      </label>
      <label style="display:grid;gap:6px">
        <span style="font-size:12px;font-weight:700;color:var(--muted)">Description</span>
        <textarea name="description" rows="4" placeholder="What this workflow is for"
          style="border:1px solid var(--line);border-radius:6px;padding:10px;font-size:13px;color:var(--text);font-family:inherit;resize:vertical">${escHtml(form.description)}</textarea>
      </label>
      <label style="display:grid;gap:6px">
        <span style="font-size:12px;font-weight:700;color:var(--muted)">Authoring mode</span>
        <select name="authoring_mode"
          style="height:36px;border:1px solid var(--line);border-radius:6px;padding:0 10px;font-size:13px;color:var(--text)">
          <option value="visual"${form.authoringMode === 'visual' ? ' selected' : ''}>Visual builder</option>
        </select>
      </label>
      <div style="display:flex;gap:10px;align-items:center;flex-wrap:wrap">
        <button type="submit" style="height:36px;padding:0 16px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Create workflow</button>
        <a href="/bff/admin/workflows" style="height:36px;line-height:36px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--muted);text-decoration:none">Cancel</a>
      </div>
    </form>
  </div>`;
}

export function buildListBody(workflows: WorkflowRow[], status: string, name: string): string {
  const rows = workflows.length === 0
    ? `<tr><td colspan="5" style="padding:24px;text-align:center;color:var(--muted)">No workflows found</td></tr>`
    : workflows.map((wf) => `
    <tr>
      <td><a href="/bff/admin/workflows/${escHtml(wf.id)}" style="color:var(--accent);text-decoration:none;font-weight:600">${escHtml(wf.name)}</a></td>
      <td>${statusBadge(wf.status)}</td>
      <td style="color:var(--muted)">${escHtml(String(wf.version))}</td>
      <td style="color:var(--muted);font-size:13px">${escHtml(wf.description ?? '—')}</td>
      <td style="color:var(--muted);font-size:12px">${escHtml(wf.updated_at.slice(0, 10))}</td>
    </tr>`).join('');
  return `
  <div style="display:flex;align-items:center;justify-content:space-between;gap:16px;flex-wrap:wrap;margin-bottom:16px">
    <h2 class="page-title" style="margin:0">Workflows</h2>
    <a href="/bff/admin/workflows/new" style="display:inline-block;height:34px;line-height:34px;padding:0 14px;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;text-decoration:none">Create workflow</a>
  </div>
  <form method="GET" action="/bff/admin/workflows" style="display:flex;gap:10px;margin-bottom:16px;flex-wrap:wrap">
    <input name="name" type="text" value="${escHtml(name)}" placeholder="Filter by name"
      style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 10px;font-size:13px;color:var(--text)">
    <select name="status" style="height:34px;border:1px solid var(--line);border-radius:6px;padding:0 8px;font-size:13px;color:var(--text)">
      <option value="">All statuses</option>
      ${['active','draft','testing','archived'].map((s) => `<option value="${s}"${status === s ? ' selected' : ''}>${s}</option>`).join('')}
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
      <tbody id="workflows-tbody">${rows}</tbody>
    </table>
  </div>`;
}

export interface WorkflowDetail {
  id: string;
  name: string;
  status: string;
  version: number;
  description?: string | null;
  created_at: string;
  updated_at: string;
  dsl_source: string;
  spec_source?: string | null;
  workspace_id?: string;
  parent_version_id?: string | null;
}

const MONO = 'ui-monospace,SFMono-Regular,Menlo,monospace';

export function escHtml(s: string): string {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

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
  const statusHint = STATUS_HINTS[wf.status] ?? 'Unknown workflow state.';
  return `
  <div style="display:flex;gap:12px;align-items:center;margin-bottom:20px">
    <a href="/bff/builder?workflowId=${encodeURIComponent(wf.id)}" style="display:inline-block;padding:8px 16px;border:1px solid var(--line);border-radius:6px;font-size:13px;font-weight:600;color:var(--accent);text-decoration:none">Open in Builder</a>
    <a href="/bff/admin/workflows" style="display:inline-block;padding:8px 16px;border:1px solid var(--line);border-radius:6px;font-size:13px;font-weight:600;color:var(--muted);text-decoration:none">Back to workflow list</a>
  </div>
  <div style="margin-bottom:20px;background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:16px">
    <h3 style="margin:0 0 8px;font-size:14px;font-weight:700">Workflow status</h3>
    <p style="margin:0;color:var(--muted);font-size:13px"><strong style="color:var(--text)">${escHtml(wf.status)}</strong> — ${escHtml(statusHint)}</p>
  </div>
  <div id="activation-section" style="background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:16px">
    <h3 style="margin:0 0 8px;font-size:14px;font-weight:700">Activation</h3>
    <p style="margin:0 0 12px;color:var(--muted);font-size:13px">Current status: <strong>${escHtml(wf.status)}</strong>. Activate transitions the workflow through testing → active.</p>
    <form method="POST" action="/bff/admin/workflows/${escHtml(wf.id)}/activate">
      <button type="submit" style="height:34px;padding:0 16px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Activate</button>
    </form>
  </div>`;
}

export function buildDetailBody(wf: WorkflowDetail): string {
  return detailMeta(wf) + detailSources(wf) + detailActivation(wf);
}
