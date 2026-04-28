// admin-screenshots Task 2: route catalog — canonical admin workflow capture set
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
    name: '03_workflow_create_draft',
    url: (base) => `${base}/bff/admin/workflows/new`,
  },
  {
    name: '04_workflow_builder_bound',
    url: (base, seed) => `${base}/bff/builder?workflowId=${encodeURIComponent(seed.workflow.id)}`,
  },
  {
    name: '05_workflow_detail',
    url: (base, seed) => `${base}/bff/admin/workflows/${seed.workflow.id}`,
  },
  {
    name: '06_agent_runs_list',
    url: (base) => `${base}/bff/admin/agent-runs`,
  },
  {
    name: '07_agent_run_detail',
    url: (base, seed) => `${base}/bff/admin/agent-runs/${seed.agentRuns.completedId}`,
  },
  {
    name: '08_approvals_list',
    url: (base) => `${base}/bff/admin/approvals`,
  },
  {
    name: '09_audit_list',
    url: (base) => `${base}/bff/admin/audit`,
  },
  {
    name: '10_audit_detail',
    url: (base, _seed, resolved) => `${base}/bff/admin/audit/${resolved.firstAuditEventId}`,
  },
  {
    name: '11_policy',
    url: (base) => `${base}/bff/admin/policy`,
  },
  {
    name: '12_tools',
    url: (base) => `${base}/bff/admin/tools`,
  },
  {
    name: '13_metrics',
    url: (base) => `${base}/bff/admin/metrics`,
  },
];
