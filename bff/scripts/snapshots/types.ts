// bff-http-snapshots T1/T2: shared types for the snapshot runner

// Shape mirrors scripts/e2e_seed_mobile_p2.go seedOutput struct (verified 2026-04-26)
export type SeederOutput = {
  credentials: { email: string; password: string };
  auth: { token: string; userId: string; workspaceId: string };
  account: { id: string };
  contact: { id: string; email: string };
  lead: { id: string };
  deal: { id: string };
  pipeline: { id: string };
  stage: { id: string };
  staleDeal: { id: string };
  case: { id: string; subject: string };
  resolvedCase: { id: string; subject: string };
  agentRuns: { completedId: string; handoffId: string; deniedByPolicyId: string };
  inbox: { approvalId: string; rejectApprovalId: string; signalId: string };
  workflow: { id: string };
};

export type CatalogEntry = {
  name: string;
  group: string;
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  path: string;
  auth: boolean;
  sse?: { maxEvents: number; timeoutMs: number };
  body?: unknown | ((seed: SeederOutput) => unknown);
  pathParams?: (seed: SeederOutput) => Record<string, string>;
  expectedStatus: number;
};

export type SnapshotArtifact = {
  name: string;
  method: string;
  path: string;
  expectedStatus?: number;
  request: {
    headers: Record<string, string>;
    body?: unknown;
  };
  response: {
    status: number;
    headers: Record<string, string>;
    body?: unknown;
  };
  latencyMs: number | '<duration>';
  capturedAt: string;
};

export type HealthCheckResult = {
  ok: boolean;
  url: string;
  error?: string;
};

export type RunnerConfig = {
  bffBaseUrl: string;
  goBaseUrl: string;
  outputDir: string;
  sseTimeoutMs: number;
};
