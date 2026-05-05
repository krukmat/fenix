// admin-screenshots Task 2: route catalog — canonical admin workflow capture set
import type { SeederOutput } from '../snapshots/types';

export type ResolvedIds = {
  firstAuditEventId: string;
};

export type AdminScreenshotEntry = {
  name: string;
  requiresSession: boolean;
  url: (base: string, seed: SeederOutput, resolved: ResolvedIds) => string;
};

export const catalog: AdminScreenshotEntry[] = [
  {
    name: '00_login',
    requiresSession: false,
    url: (base) => `${base}/bff/admin/login`,
  },
  {
    name: '01_dashboard',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/`,
  },
  {
    name: '02_workflows_list',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/workflows`,
  },
  {
    name: '03_workflow_create_draft',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/workflows/new`,
  },
  {
    name: '04_workflow_builder_bound',
    requiresSession: true,
    url: (base, seed) => `${base}/bff/builder?workflowId=${encodeURIComponent(seed.workflow.id)}`,
  },
  {
    name: '05_workflow_detail',
    requiresSession: true,
    url: (base, seed) => `${base}/bff/admin/workflows/${seed.workflow.id}`,
  },
  {
    name: '06_agent_runs_list',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/agent-runs`,
  },
  {
    name: '07_agent_run_detail',
    requiresSession: true,
    url: (base, seed) => `${base}/bff/admin/agent-runs/${seed.agentRuns.completedId}`,
  },
  {
    name: '08_approvals_list',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/approvals`,
  },
  {
    name: '09_audit_list',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/audit`,
  },
  {
    name: '10_audit_detail',
    requiresSession: true,
    url: (base, _seed, resolved) => `${base}/bff/admin/audit/${resolved.firstAuditEventId}`,
  },
  {
    name: '11_policy',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/policy`,
  },
  {
    name: '12_tools',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/tools`,
  },
  {
    name: '13_metrics',
    requiresSession: true,
    url: (base) => `${base}/bff/admin/metrics`,
  },
];
