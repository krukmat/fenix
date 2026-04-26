// bff-http-snapshots T3: declarative catalog of BFF endpoints to snapshot
// Add a new endpoint = add one entry here, no other changes needed.
import type { CatalogEntry, SeederOutput } from './types';

const SSE_TIMEOUT_MS = parseInt(process.env['FENIX_SNAPSHOTS_SSE_TIMEOUT_MS'] ?? '5000', 10);

export const catalog: CatalogEntry[] = [
  // ── Health & Metrics ────────────────────────────────────────────────────────
  {
    name: 'health',
    group: 'health',
    method: 'GET',
    path: '/bff/health',
    auth: false,
    expectedStatus: 200,
  },
  {
    name: 'metrics',
    group: 'metrics',
    method: 'GET',
    path: '/bff/metrics',
    auth: false,
    expectedStatus: 200,
  },

  // ── Auth ────────────────────────────────────────────────────────────────────
  {
    name: 'auth-login-success',
    group: 'auth',
    method: 'POST',
    path: '/bff/auth/login',
    auth: false,
    body: (seed: SeederOutput) => ({
      email: seed.credentials.email,
      password: seed.credentials.password,
    }),
    expectedStatus: 200,
  },
  {
    name: 'auth-login-invalid',
    group: 'auth',
    method: 'POST',
    path: '/bff/auth/login',
    auth: false,
    body: (seed: SeederOutput) => ({
      email: seed.credentials.email,
      password: 'wrong-password',
    }),
    expectedStatus: 401,
  },
  {
    name: 'auth-register-success',
    group: 'auth',
    method: 'POST',
    path: '/bff/auth/register',
    auth: false,
    body: () => ({
      email: `snapshot-${Date.now()}@fenix.test`,
      password: 'SnapshotPass123!',
      displayName: 'Snapshot User',
      workspaceName: 'Snapshot Workspace',
    }),
    expectedStatus: 201,
  },

  // ── Builder ─────────────────────────────────────────────────────────────────
  {
    name: 'builder-list',
    group: 'builder',
    method: 'GET',
    path: '/bff/builder',
    auth: true,
    expectedStatus: 200,
  },

  // ── Copilot ─────────────────────────────────────────────────────────────────
  {
    name: 'copilot-chat-grounded',
    group: 'copilot',
    method: 'POST',
    path: '/bff/api/v1/copilot/chat',
    auth: true,
    body: (seed: SeederOutput) => ({
      query: 'Summarize the latest case',
      entityId: seed.case.id,
      entityType: 'case',
    }),
    expectedStatus: 200,
  },
  {
    name: 'copilot-chat-abstain',
    group: 'copilot',
    method: 'POST',
    path: '/bff/api/v1/copilot/chat',
    auth: true,
    body: () => ({
      query: 'Summarize entity that does not exist',
      entityId: '00000000-0000-0000-0000-000000000000',
      entityType: 'case',
    }),
    expectedStatus: 200,
  },
  {
    name: 'copilot-stream',
    group: 'copilot',
    method: 'GET',
    path: '/bff/copilot/events',
    auth: true,
    sse: { maxEvents: 10, timeoutMs: SSE_TIMEOUT_MS },
    expectedStatus: 200,
  },
  {
    name: 'copilot-sales-brief',
    group: 'copilot',
    method: 'POST',
    path: '/bff/copilot/sales-brief',
    auth: true,
    body: (seed: SeederOutput) => ({
      entityType: 'deal',
      entityId: seed.deal.id,
    }),
    expectedStatus: 200,
  },

  // ── Inbox ───────────────────────────────────────────────────────────────────
  {
    name: 'inbox-list-with-items',
    group: 'inbox',
    method: 'GET',
    path: '/bff/api/v1/mobile/inbox',
    auth: true,
    expectedStatus: 200,
  },

  // ── Approvals ───────────────────────────────────────────────────────────────
  {
    name: 'approvals-approve',
    group: 'approvals',
    method: 'POST',
    path: '/bff/api/v1/approvals/:id/approve',
    auth: true,
    pathParams: (seed: SeederOutput) => ({ id: seed.inbox.approvalId }),
    body: () => ({ comment: 'Approved by snapshot runner' }),
    expectedStatus: 204,
  },
  {
    name: 'approvals-reject',
    group: 'approvals',
    method: 'POST',
    path: '/bff/api/v1/approvals/:id/reject',
    auth: true,
    pathParams: (seed: SeederOutput) => ({ id: seed.inbox.rejectApprovalId }),
    body: () => ({ comment: 'Rejected by snapshot runner' }),
    expectedStatus: 204,
  },

  // ── Transparent proxy passthrough ────────────────────────────────────────────
  {
    name: 'proxy-passthrough-deals',
    group: 'proxy',
    method: 'GET',
    path: '/bff/api/v1/deals',
    auth: true,
    expectedStatus: 200,
  },
  {
    name: 'proxy-passthrough-accounts',
    group: 'proxy',
    method: 'GET',
    path: '/bff/api/v1/accounts',
    auth: true,
    expectedStatus: 200,
  },
  {
    name: 'proxy-passthrough-cases',
    group: 'proxy',
    method: 'GET',
    path: '/bff/api/v1/cases',
    auth: true,
    expectedStatus: 200,
  },
];
