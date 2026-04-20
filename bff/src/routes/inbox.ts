// W1-T3 (mobile_wedge_harmonization_plan): BFF inbox aggregation route
// GET /bff/api/v1/mobile/inbox → aggregates approvals + signals + handed-off + rejected runs
// Partial enrichment failure on handoffs does NOT fail the whole response.
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';

const router = Router();

type BffRequest = Request & { bearerToken?: string };

interface HandoffPackage {
  run_id: string;
  reason: string;
  conversation_context: string;
  evidence_count: number;
  entity_type?: string;
  entity_id?: string;
  created_at: string;
}

interface AgentRun {
  id: string;
  status: string;
  [key: string]: unknown;
}

interface AgentRunsResponse {
  data: AgentRun[];
}

type GoClient = ReturnType<typeof createGoClient>;

function buildParams(workspaceId: string | undefined): Record<string, string | number> {
  const params: Record<string, string | number> = { limit: 50 };
  if (workspaceId) params.workspace_id = workspaceId;
  return params;
}

async function enrichHandoff(client: GoClient, run: AgentRun) {
  try {
    const res = await client.get(`/api/v1/agents/runs/${run.id}/handoff`);
    return { type: 'handoff' as const, run_id: run.id, handoff: res.data as HandoffPackage };
  } catch {
    // One enrichment failure must not fail the full inbox response
    return null;
  }
}

type RawResponse = { data: unknown };

function normalizeList(res: RawResponse): unknown[] {
  const d = res.data as Record<string, unknown> | unknown[];
  if (Array.isArray(d)) return d;
  return ((d as Record<string, unknown>)?.data as unknown[]) ?? [];
}

function normalizeSignals(res: RawResponse): unknown[] {
  const d = res.data;
  if (Array.isArray(d)) return d;
  return ((d as Record<string, unknown>)?.data as unknown[]) ?? [];
}

function normalizeRuns(res: RawResponse): AgentRun[] {
  return ((res.data as AgentRunsResponse)?.data ?? []) as AgentRun[];
}

async function fetchInboxData(client: GoClient, params: Record<string, string | number>) {
  const [approvalsRes, signalsRes, handedOffRunsRes, rejectedRunsRes] = await Promise.all([
    client.get('/api/v1/approvals', { params }).catch(() => ({ data: { data: [] } })),
    client.get('/api/v1/signals', { params: { ...params, status: 'active' } }).catch(() => ({ data: [] })),
    client.get('/api/v1/agents/runs', { params: { ...params, status: 'handed_off', limit: 20 } }).catch(() => ({ data: { data: [] } })),
    client.get('/api/v1/agents/runs', { params: { ...params, status: 'denied_by_policy', limit: 20 } }).catch(() => ({ data: { data: [] } })),
  ]);
  return {
    approvals: normalizeList(approvalsRes),
    signals: normalizeSignals(signalsRes),
    handedOffRuns: normalizeRuns(handedOffRunsRes),
    rejected: normalizeRuns(rejectedRunsRes),
  };
}

// GET /bff/api/v1/mobile/inbox
router.get('/', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const client = createGoClient(req.bearerToken);
    const params = buildParams(req.query.workspace_id as string | undefined);
    const { approvals, signals, handedOffRuns, rejected } = await fetchInboxData(client, params);
    const handoffs = (await Promise.all(handedOffRuns.map((run) => enrichHandoff(client, run))))
      .filter((item): item is NonNullable<typeof item> => item !== null);
    res.json({ approvals, handoffs, signals, rejected });
  } catch (err) {
    next(err);
  }
});

export default router;
