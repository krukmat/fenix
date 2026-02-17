// Task 4.1 — FR-301: Aggregated routes — combine multiple Go API calls into one mobile-optimized response
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';

const router = Router();

type BffRequest = Request & { bearerToken?: string; mobileHeaders?: Record<string, string> };

function getToken(req: BffRequest): string | undefined {
  return req.bearerToken;
}

// GET /bff/accounts/:id/full
// Combines: account + contacts + deals + timeline
router.get('/accounts/:id/full', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const { id } = req.params;
    const token = getToken(req);
    const client = createGoClient(token);

    const [accountRes, contactsRes, dealsRes, timelineRes] = await Promise.allSettled([
      client.get(`/api/v1/accounts/${id}`),
      client.get(`/api/v1/contacts?account_id=${id}&limit=50`),
      client.get(`/api/v1/deals?account_id=${id}&limit=50`),
      client.get(`/api/v1/accounts/${id}/timeline`),
    ]);

    res.status(200).json({
      account: accountRes.status === 'fulfilled' ? accountRes.value.data : null,
      contacts: contactsRes.status === 'fulfilled' ? contactsRes.value.data : null,
      deals: dealsRes.status === 'fulfilled' ? dealsRes.value.data : null,
      timeline: timelineRes.status === 'fulfilled' ? timelineRes.value.data : null,
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

    const [accountRes, contactRes, activitiesRes] = await Promise.allSettled([
      deal.account_id ? client.get(`/api/v1/accounts/${deal.account_id}`) : Promise.reject(new Error('no account')),
      deal.contact_id ? client.get(`/api/v1/contacts/${deal.contact_id}`) : Promise.reject(new Error('no contact')),
      client.get(`/api/v1/activities?deal_id=${id}&limit=50`),
    ]);

    res.status(200).json({
      deal,
      account: accountRes.status === 'fulfilled' ? accountRes.value.data : null,
      contact: contactRes.status === 'fulfilled' ? contactRes.value.data : null,
      activities: activitiesRes.status === 'fulfilled' ? activitiesRes.value.data : null,
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

    const subCalls: [Promise<{ data: unknown }>, Promise<{ data: unknown }>, Promise<{ data: unknown }>, Promise<{ data: unknown }>] = [
      caseData.account_id ? client.get(`/api/v1/accounts/${caseData.account_id}`) : Promise.reject(new Error('no account')),
      caseData.contact_id ? client.get(`/api/v1/contacts/${caseData.contact_id}`) : Promise.reject(new Error('no contact')),
      client.get(`/api/v1/activities?case_id=${id}&limit=50`),
      caseData.handoff_id ? client.get(`/api/v1/handoffs/${caseData.handoff_id}`) : Promise.reject(new Error('no handoff')),
    ];

    const [accountRes, contactRes, activitiesRes, handoffRes] = await Promise.allSettled(subCalls);

    res.status(200).json({
      case: caseData,
      account: accountRes.status === 'fulfilled' ? accountRes.value.data : null,
      contact: contactRes.status === 'fulfilled' ? contactRes.value.data : null,
      activities: activitiesRes.status === 'fulfilled' ? activitiesRes.value.data : null,
      handoff: handoffRes.status === 'fulfilled' ? handoffRes.value.data : null,
    });
  } catch (err) {
    next(err);
  }
});

export default router;
