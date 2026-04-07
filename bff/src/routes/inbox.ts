// W1-T3 (mobile_wedge_harmonization_plan): BFF inbox aggregation route
// GET /bff/api/v1/mobile/inbox → aggregates approvals + signals + handed-off runs
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

// GET /bff/api/v1/mobile/inbox
router.get('/', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const client = createGoClient(req.bearerToken);
    const workspaceId = req.query.workspace_id as string | undefined;

    const params: Record<string, string | number> = { limit: 50 };
    if (workspaceId) {
      params.workspace_id = workspaceId;
    }

    // Fetch approvals, signals, and handed-off runs in parallel
    const [approvalsRes, signalsRes, handedOffRunsRes] = await Promise.all([
      client.get('/api/v1/approvals', { params }).catch(() => ({ data: { data: [] } })),
      client.get('/api/v1/signals', { params: { ...params, status: 'active' } }).catch(() => ({ data: [] })),
      client.get('/api/v1/agents/runs', { params: { ...params, status: 'handed_off', limit: 20 } }).catch(() => ({ data: { data: [] } })),
    ]);

    const approvals = (approvalsRes.data?.data ?? approvalsRes.data ?? []) as unknown[];
    const signals = Array.isArray(signalsRes.data) ? signalsRes.data : (signalsRes.data?.data ?? []) as unknown[];
    const handedOffRuns = ((handedOffRunsRes.data as AgentRunsResponse)?.data ?? []) as AgentRun[];

    // Enrich each handed-off run with its handoff package — skip failures individually
    const handoffs = (
      await Promise.all(
        handedOffRuns.map(async (run) => {
          try {
            const handoffRes = await client.get(`/api/v1/agents/runs/${run.id}/handoff`);
            const handoff = handoffRes.data as HandoffPackage;
            return { type: 'handoff' as const, run_id: run.id, handoff };
          } catch {
            // One handoff enrichment failure must not fail the full inbox response
            return null;
          }
        })
      )
    ).filter((item): item is NonNullable<typeof item> => item !== null);

    res.json({ approvals, handoffs, signals });
  } catch (err) {
    next(err);
  }
});

export default router;
