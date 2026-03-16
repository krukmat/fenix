// Task 4.1 — FR-301: Aggregated routes — combine multiple Go API calls into one mobile-optimized response
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';

const router = Router();

type BffRequest = Request & { bearerToken?: string; mobileHeaders?: Record<string, string> };
type SettledData = PromiseSettledResult<{ data: unknown }>;
type SignalPayload = { data?: Array<{ status?: string }> } | null;
type ListPayload = { data?: Array<Record<string, unknown>>; meta?: unknown } | null;

function getToken(req: BffRequest): string | undefined {
  return req.bearerToken;
}

function settledDataOrNull(result: SettledData): unknown | null {
  return result.status === 'fulfilled' ? result.value.data : null;
}

function queryStringFromRequest(req: Request): string {
  const query = new URLSearchParams();
  Object.entries(req.query).forEach(([key, value]) => {
    if (Array.isArray(value)) {
      value.forEach((item) => {
        if (item !== undefined) {
          query.append(key, String(item));
        }
      });
      return;
    }
    if (value !== undefined) {
      query.append(key, String(value));
    }
  });
  const serialized = query.toString();
  return serialized ? `?${serialized}` : '';
}

async function enrichListWithSignalCounts(
  client: ReturnType<typeof createGoClient>,
  payload: ListPayload,
  entityType: 'account' | 'deal' | 'case'
) {
  if (!payload?.data || !Array.isArray(payload.data)) {
    return payload;
  }

  const enriched = await Promise.all(
    payload.data.map(async (item) => {
      const entityId = typeof item.id === 'string' ? item.id : '';
      if (!entityId) {
        return item;
      }
      try {
        const signalsRes = await client.get(`/api/v1/signals?entity_type=${entityType}&entity_id=${entityId}`);
        return {
          ...item,
          active_signal_count: countActiveSignals(signalsRes.data as SignalPayload),
        };
      } catch {
        return {
          ...item,
          active_signal_count: 0,
        };
      }
    })
  );

  return {
    ...payload,
    data: enriched,
  };
}

router.get('/accounts', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const token = getToken(req);
    const client = createGoClient(token);
    const query = queryStringFromRequest(req);
    const accountsRes = await client.get(`/api/v1/accounts${query}`);
    const enriched = await enrichListWithSignalCounts(client, accountsRes.data as ListPayload, 'account');
    res.status(200).json(enriched);
  } catch (err) {
    next(err);
  }
});

router.get('/deals', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const token = getToken(req);
    const client = createGoClient(token);
    const query = queryStringFromRequest(req);
    const dealsRes = await client.get(`/api/v1/deals${query}`);
    const enriched = await enrichListWithSignalCounts(client, dealsRes.data as ListPayload, 'deal');
    res.status(200).json(enriched);
  } catch (err) {
    next(err);
  }
});

router.get('/cases', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const token = getToken(req);
    const client = createGoClient(token);
    const query = queryStringFromRequest(req);
    const casesRes = await client.get(`/api/v1/cases${query}`);
    const enriched = await enrichListWithSignalCounts(client, casesRes.data as ListPayload, 'case');
    res.status(200).json(enriched);
  } catch (err) {
    next(err);
  }
});

// GET /bff/accounts/:id/full
// Combines: account + contacts + deals + timeline
router.get('/accounts/:id/full', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const { id } = req.params;
    const token = getToken(req);
    const client = createGoClient(token);

    const [accountRes, contactsRes, dealsRes, timelineRes, signalsRes] = await Promise.allSettled([
      client.get(`/api/v1/accounts/${id}`),
      client.get(`/api/v1/contacts?account_id=${id}&limit=50`),
      client.get(`/api/v1/deals?account_id=${id}&limit=50`),
      client.get(`/api/v1/timeline/account/${id}`),
      client.get(`/api/v1/signals?entity_type=account&entity_id=${id}`),
    ]);

    const signals = settledDataOrNull(signalsRes) as { data?: Array<{ status?: string }> } | null;
    res.status(200).json({
      account: settledDataOrNull(accountRes),
      contacts: settledDataOrNull(contactsRes),
      deals: settledDataOrNull(dealsRes),
      timeline: settledDataOrNull(timelineRes),
      active_signal_count: countActiveSignals(signals),
    });
  } catch (err) {
    next(err);
  }
});

// GET /bff/deals/:id/full
// Combines: deal + account + contact + activities
router.get('/deals/:id/full', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const { id } = req.params;
    const token = getToken(req);
    const client = createGoClient(token);

    const dealRes = await client.get(`/api/v1/deals/${id}`);
    const deal = dealRes.data as { account_id?: string; contact_id?: string };

    const [accountRes, contactRes, activitiesRes, signalsRes] = await Promise.allSettled([
      deal.account_id ? client.get(`/api/v1/accounts/${deal.account_id}`) : Promise.reject(new Error('no account')),
      deal.contact_id ? client.get(`/api/v1/contacts/${deal.contact_id}`) : Promise.reject(new Error('no contact')),
      client.get(`/api/v1/activities?deal_id=${id}&limit=50`),
      client.get(`/api/v1/signals?entity_type=deal&entity_id=${id}`),
    ]);

    const signals = settledDataOrNull(signalsRes) as { data?: Array<{ status?: string }> } | null;
    res.status(200).json({
      deal,
      account: settledDataOrNull(accountRes),
      contact: settledDataOrNull(contactRes),
      activities: settledDataOrNull(activitiesRes),
      active_signal_count: countActiveSignals(signals),
    });
  } catch (err) {
    next(err);
  }
});

// GET /bff/cases/:id/full
// Combines: case + account + contact + activities + handoff (if escalated)
router.get('/cases/:id/full', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const { id } = req.params;
    const token = getToken(req);
    const client = createGoClient(token);

    const caseRes = await client.get(`/api/v1/cases/${id}`);
    const caseData = caseRes.data as { account_id?: string; contact_id?: string; handoff_id?: string };

    const subCalls: [Promise<{ data: unknown }>, Promise<{ data: unknown }>, Promise<{ data: unknown }>, Promise<{ data: unknown }>, Promise<{ data: unknown }>] = [
      caseData.account_id ? client.get(`/api/v1/accounts/${caseData.account_id}`) : Promise.reject(new Error('no account')),
      caseData.contact_id ? client.get(`/api/v1/contacts/${caseData.contact_id}`) : Promise.reject(new Error('no contact')),
      client.get(`/api/v1/activities?case_id=${id}&limit=50`),
      caseData.handoff_id ? client.get(`/api/v1/handoffs/${caseData.handoff_id}`) : Promise.reject(new Error('no handoff')),
      client.get(`/api/v1/signals?entity_type=case&entity_id=${id}`),
    ];

    const [accountRes, contactRes, activitiesRes, handoffRes, signalsRes] = await Promise.allSettled(subCalls);
    const signals = settledDataOrNull(signalsRes) as { data?: Array<{ status?: string }> } | null;

    res.status(200).json({
      case: caseData,
      account: settledDataOrNull(accountRes),
      contact: settledDataOrNull(contactRes),
      activities: settledDataOrNull(activitiesRes),
      handoff: settledDataOrNull(handoffRes),
      active_signal_count: countActiveSignals(signals),
    });
  } catch (err) {
    next(err);
  }
});

function countActiveSignals(payload: SignalPayload): number {
  if (!payload || !Array.isArray(payload.data)) {
    return 0;
  }
  return payload.data.filter((signal) => signal?.status === 'active').length;
}

export default router;
