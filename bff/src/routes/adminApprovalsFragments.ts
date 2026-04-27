// BFF-ADMIN-31: approval detail fragment builders — extracted to satisfy max-lines gate
export const PANEL = 'background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:24px;margin-bottom:20px';

export interface ToolCall {
  toolName: string;
  args?: unknown;
  result?: string;
}

export interface ApprovalDetail {
  id: string;
  requestedBy: string;
  action: string;
  status: string;
  reason?: string | null;
  createdAt: string;
  proposedPayload?: unknown;
  reasoningTrace?: string;
  toolCalls?: ToolCall[];
}

function esc(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function renderToolCallRows(calls: ToolCall[]): string {
  return calls.map((tc) => `
    <tr style="border-bottom:1px solid var(--line)">
      <td style="padding:8px 12px;font-family:ui-monospace,monospace;font-size:12px">${esc(tc.toolName)}</td>
      <td style="padding:8px 12px;font-size:12px;color:var(--muted)"><code style="background:var(--bg);padding:2px 4px;border-radius:3px">${esc(JSON.stringify(tc.args ?? {}).slice(0, 60))}</code></td>
      <td style="padding:8px 12px;font-size:12px">${tc.result ? `<span style="color:#065f46">${esc(tc.result)}</span>` : '<span style="color:var(--muted)">—</span>'}</td>
    </tr>`).join('');
}

export function buildProposedPayloadSection(approval: ApprovalDetail): string {
  if (!approval.proposedPayload) return '';
  const payloadJson = JSON.stringify(approval.proposedPayload, null, 2);
  return `
  <div style="${PANEL}">
    <h3 style="margin:0 0 12px;font-size:16px;font-weight:700">Proposed Payload</h3>
    <pre style="margin:0;background:var(--bg);padding:12px;border-radius:6px;overflow-x:auto;font-size:12px;color:var(--text)"><code>${esc(payloadJson)}</code></pre>
  </div>`;
}

export function buildReasoningTraceSection(approval: ApprovalDetail): string {
  if (!approval.reasoningTrace && (!approval.toolCalls || approval.toolCalls.length === 0)) return '';
  let html = '';
  if (approval.reasoningTrace) {
    html += `
  <div style="${PANEL}">
    <h3 style="margin:0 0 12px;font-size:16px;font-weight:700">Reasoning Trace</h3>
    <p style="margin:0;font-size:13px;line-height:1.6;color:var(--text)">${esc(approval.reasoningTrace)}</p>
  </div>`;
  }
  if (approval.toolCalls && approval.toolCalls.length > 0) {
    html += `
  <div style="${PANEL}">
    <h3 style="margin:0 0 12px;font-size:16px;font-weight:700">Tool Calls</h3>
    <table style="width:100%;border-collapse:collapse;font-size:12px">
      <thead>
        <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:var(--muted)">Tool</th>
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:var(--muted)">Args</th>
          <th style="padding:8px 12px;text-align:left;font-size:11px;font-weight:700;color:var(--muted)">Result</th>
        </tr>
      </thead>
      <tbody>${renderToolCallRows(approval.toolCalls)}</tbody>
    </table>
  </div>`;
  }
  return html;
}
