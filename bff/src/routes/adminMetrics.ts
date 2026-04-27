// BFF-ADMIN-70: metrics dashboard — proxy of Go GET /metrics (Prometheus text)
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus } from './adminAuth';

const ADMIN_ROOT = '/bff/admin';

interface MetricEntry {
  name: string;
  value: string;
}

function parsePrometheus(text: string): MetricEntry[] {
  return text
    .split('\n')
    .filter((line) => line && !line.startsWith('#'))
    .map((line) => {
      const idx = line.lastIndexOf(' ');
      return { name: line.slice(0, idx).trim(), value: line.slice(idx + 1).trim() };
    })
    .filter((e) => e.name && e.value);
}

function esc(s: unknown): string {
  return String(s ?? '')
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function buildCard(name: string, value: string): string {
  return `
  <div style="background:var(--panel);border:1px solid var(--line);border-radius:8px;padding:20px 24px;min-width:180px;flex:1">
    <div style="font-size:12px;font-weight:600;color:var(--muted);margin-bottom:8px;font-family:ui-monospace,monospace">${esc(name)}</div>
    <div style="font-size:28px;font-weight:700;color:var(--text)">${esc(value)}</div>
  </div>`;
}

function buildBody(metrics: MetricEntry[]): string {
  const cards = metrics.length > 0
    ? metrics.map((m) => buildCard(m.name, m.value)).join('')
    : `<p style="color:var(--muted);font-size:14px;margin:0">No metrics available</p>`;

  return `
  <h2 class="page-title">Metrics</h2>
  <p style="color:var(--muted);font-size:13px;margin:0 0 20px">
    Go backend counters (Prometheus text). For per-policy quota consumption see
    <a href="/bff/admin/policy" style="color:var(--accent)">Governance</a>.
  </p>
  <div style="display:flex;gap:16px;flex-wrap:wrap;margin-bottom:28px">${cards}</div>`;
}

const router = Router();

router.get('/', async (_req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  try {
    const client = createGoClient(token);
    const { data } = await client.get<string>('/metrics', { responseType: 'text' });
    const metrics = parsePrometheus(typeof data === 'string' ? data : '');
    res.type('html').status(200).send(adminLayout('Metrics', buildBody(metrics)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

export default router;
