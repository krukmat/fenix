// admin-screenshots Task 2: route catalog — 11 canonical admin pages to capture
import type { SeederOutput } from '../snapshots/types';

export type ResolvedIds = {
  firstAuditEventId: string;
};

export type AdminScreenshotEntry = {
  name: string;
  url: (base: string, seed: SeederOutput, resolved: ResolvedIds) => string;
};

export const catalog: AdminScreenshotEntry[] = [
  {
    name: '01_dashboard',
    url: (base) => `${base}/bff/admin/`,
  },
  {
    name: '02_workflows_list',
    url: (base) => `${base}/bff/admin/workflows`,
  },
  {
    name: '03_workflow_detail',
    url: (base, seed) => `${base}/bff/admin/workflows/${seed.workflow.id}`,
  },
  {
    name: '04_agent_runs_list',
    url: (base) => `${base}/bff/admin/agent-runs`,
  },
  {
    name: '05_agent_run_detail',
    url: (base, seed) => `${base}/bff/admin/agent-runs/${seed.agentRuns.completedId}`,
  },
  {
    name: '06_approvals_list',
    url: (base) => `${base}/bff/admin/approvals`,
  },
  {
    name: '07_audit_list',
    url: (base) => `${base}/bff/admin/audit`,
  },
  {
    name: '08_audit_detail',
    url: (base, _seed, resolved) => `${base}/bff/admin/audit/${resolved.firstAuditEventId}`,
  },
  {
    name: '09_policy',
    url: (base) => `${base}/bff/admin/policy`,
  },
  {
    name: '10_tools',
    url: (base) => `${base}/bff/admin/tools`,
  },
  {
    name: '11_metrics',
    url: (base) => `${base}/bff/admin/metrics`,
  },
];
