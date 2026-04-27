// BFF-ADMIN-50 / BFF-ADMIN-51: governance summary + policy sets list + versions drill-down
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus } from './adminAuth';

const ADMIN_ROOT = '/bff/admin';

interface QuotaState {
  agentId: string;
  metric: string;
  used: number;
  limit: number;
  status: string;
}

interface UsageEntry {
  agentId: string;
  tokensUsed: number;
  costEuros: number;
  createdAt: string;
}

interface GovernanceSummary {
  quotaStates: QuotaState[];
  recentUsage: UsageEntry[];
}

interface PolicySet {
  id: string;
  name: string;
  version: number;
  active: boolean;
  createdAt: string;
}

interface PolicyVersion {
  id: string;
  policySetId: string;
  version: number;
  active: boolean;
  createdAt: string;
}

const QUOTA_STATUS_COLORS: Record<string, string> = {
  ok:       'background:#d1fae5;color:#065f46',
  warning:  'background:#fef3c7;color:#92400e',
  exceeded: 'background:#fee2e2;color:#991b1b',
};

const BADGE_NEUTRAL = 'background:#f3f4f6;color:#374151';

function esc(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function badge(label: string, style: string): string {
  return `<span style="display:inline-block;padding:2px 8px;border-radius:999px;font-size:12px;font-weight:600;${style}">${esc(label)}</span>`;
}

function quotaBadge(status: string): string {
  return badge(status, QUOTA_STATUS_COLORS[status] ?? BADGE_NEUTRAL);
}

function activeBadge(active: boolean): string {
  return active
    ? badge('active',   'background:#d1fae5;color:#065f46')
    : badge('inactive', BADGE_NEUTRAL);
}

// ── Quota states table ───────────────────────────────────────────────────────

function renderQuotaRows(rows: QuotaState[]): string {
  if (rows.length === 0) {
    return `<tr><td colspan="5" style="padding:24px;text-align:center;color:var(--muted)">No quota states</td></tr>`;
  }
  return rows.map((q) => `
    <tr style="border-bottom:1px solid var(--line)">
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:12px;font-weight:600;color:var(--accent)">${esc(q.agentId)}</td>
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:13px">${esc(q.metric)}</td>
      <td style="padding:10px 14px;font-size:13px;text-align:right">${esc(q.used)}</td>
      <td style="padding:10px 14px;font-size:13px;text-align:right;color:var(--muted)">${esc(q.limit)}</td>
      <td style="padding:10px 14px">${quotaBadge(q.status)}</td>
    </tr>`).join('');
}

function buildQuotaSection(states: QuotaState[]): string {
  return `
  <section style="margin-bottom:28px">
    <h2 class="page-title" style="margin-bottom:12px">Quota States</h2>
    <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
      <table style="width:100%;border-collapse:collapse;font-size:14px">
        <thead>
          <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
            <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Agent</th>
            <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Metric</th>
            <th style="padding:10px 14px;text-align:right;font-size:12px;font-weight:700;color:var(--muted)">Used</th>
            <th style="padding:10px 14px;text-align:right;font-size:12px;font-weight:700;color:var(--muted)">Limit</th>
            <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Status</th>
          </tr>
        </thead>
        <tbody>${renderQuotaRows(states)}</tbody>
      </table>
    </div>
  </section>`;
}

// ── Policy sets table ────────────────────────────────────────────────────────

function renderPolicySetRows(rows: PolicySet[]): string {
  if (rows.length === 0) {
    return `<tr><td colspan="4" style="padding:24px;text-align:center;color:var(--muted)">No policy sets</td></tr>`;
  }
  return rows.map((p) => `
    <tr style="border-bottom:1px solid var(--line)">
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:12px;font-weight:600;color:var(--accent)">
        <a href="/bff/admin/policy/${esc(p.id)}/versions" style="color:var(--accent);text-decoration:none">${esc(p.id)}</a>
      </td>
      <td style="padding:10px 14px;font-size:13px">${esc(p.name)}</td>
      <td style="padding:10px 14px;font-size:13px;text-align:center">${esc(p.version)}</td>
      <td style="padding:10px 14px">${activeBadge(p.active)}</td>
    </tr>`).join('');
}

function buildPolicySetSection(sets: PolicySet[]): string {
  return `
  <section style="margin-bottom:28px">
    <h2 class="page-title" style="margin-bottom:12px">Policy Sets</h2>
    <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
      <table style="width:100%;border-collapse:collapse;font-size:14px">
        <thead>
          <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
            <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">ID</th>
            <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Name</th>
            <th style="padding:10px 14px;text-align:center;font-size:12px;font-weight:700;color:var(--muted)">Version</th>
            <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">State</th>
          </tr>
        </thead>
        <tbody>${renderPolicySetRows(sets)}</tbody>
      </table>
    </div>
  </section>`;
}

// ── Versions drill-down ──────────────────────────────────────────────────────

function renderVersionRows(rows: PolicyVersion[]): string {
  if (rows.length === 0) {
    return `<tr><td colspan="4" style="padding:24px;text-align:center;color:var(--muted)">No versions</td></tr>`;
  }
  return rows.map((v) => `
    <tr style="border-bottom:1px solid var(--line)">
      <td style="padding:10px 14px;font-family:ui-monospace,monospace;font-size:12px;font-weight:600;color:var(--accent)">${esc(v.id)}</td>
      <td style="padding:10px 14px;font-size:13px;text-align:center">${esc(v.version)}</td>
      <td style="padding:10px 14px">${activeBadge(v.active)}</td>
      <td style="padding:10px 14px;font-size:12px;color:var(--muted)">${esc(v.createdAt.slice(0, 16).replace('T', ' '))}</td>
    </tr>`).join('');
}

function buildVersionsBody(policySetId: string, versions: PolicyVersion[]): string {
  return `
  <div style="margin-bottom:20px">
    <a href="/bff/admin/policy" style="font-size:13px;color:var(--muted);text-decoration:none">&larr; Policy</a>
  </div>
  <h2 class="page-title" style="margin-bottom:12px">Versions — ${esc(policySetId)}</h2>
  <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;overflow:hidden">
    <table style="width:100%;border-collapse:collapse;font-size:14px">
      <thead>
        <tr style="border-bottom:1px solid var(--line);background:var(--bg)">
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">ID</th>
          <th style="padding:10px 14px;text-align:center;font-size:12px;font-weight:700;color:var(--muted)">Version</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">State</th>
          <th style="padding:10px 14px;text-align:left;font-size:12px;font-weight:700;color:var(--muted)">Created</th>
        </tr>
      </thead>
      <tbody>${renderVersionRows(versions)}</tbody>
    </table>
  </div>`;
}

const router = Router();

// BFF-ADMIN-50: governance summary + policy sets overview
router.get('/', async (_req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  try {
    const client = createGoClient(token);
    const { data: summary } = await client.get<GovernanceSummary>('/api/v1/governance/summary');
    const { data: setsResp } = await client.get<{ data: PolicySet[] }>('/api/v1/policy/sets');

    const quotaStates = Array.isArray(summary?.quotaStates) ? summary.quotaStates : [];
    const sets = Array.isArray(setsResp?.data) ? setsResp.data : [];

    const body = buildQuotaSection(quotaStates) + buildPolicySetSection(sets);
    res.type('html').status(200).send(adminLayout('Policy & Governance', body));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

// BFF-ADMIN-51: policy set versions drill-down (read-only)
router.get('/:id/versions', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const id = String(req.params['id'] ?? '');
  try {
    const client = createGoClient(token);
    const { data: resp } = await client.get<{ data: PolicyVersion[] }>(`/api/v1/policy/sets/${id}/versions`);
    const versions = Array.isArray(resp?.data) ? resp.data : [];
    res.type('html').status(200).send(adminLayout('Policy Versions', buildVersionsBody(id, versions)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

export default router;
