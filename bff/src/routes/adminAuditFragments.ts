// BFF-ADMIN-Task6: audit detail HTML fragment — extracted to stay within max-lines gate
const PANEL = 'background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:24px;margin-bottom:20px';

const OUTCOME_COLORS: Record<string, string> = {
  success: 'background:#d1fae5;color:#065f46',
  failure: 'background:#fee2e2;color:#991b1b',
  denied:  'background:#fef3c7;color:#92400e',
};
const BADGE_NEUTRAL = 'background:#f3f4f6;color:#374151';

export interface AuditDetail {
  id: string;
  actor_id: string;
  actor_type: string;
  action: string;
  entity_type: string;
  entity_id: string;
  outcome: string;
  created_at: string;
  details?: Record<string, unknown>;
  permissions_checked?: Array<{ rule: string; result: string }>;
}

export function esc(s: string): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

export function outcomeBadge(outcome: string): string {
  const style = OUTCOME_COLORS[outcome] ?? BADGE_NEUTRAL;
  return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:600;${style}">${esc(outcome)}</span>`;
}

export function buildDetailBody(e: AuditDetail): string {
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
