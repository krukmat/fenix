// W1-T2 (mobile_wedge_harmonization_plan): BFF approval alias routes
// Translates POST /approve and /reject to the existing PUT /api/v1/approvals/{id} backend handler.
// The mobile client sends 'approve' or 'reject' — 'deny' is legacy and not accepted here.
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';

const router = Router();

type BffRequest = Request & { bearerToken?: string };

// POST /bff/api/v1/approvals/:id/approve
router.post('/:id/approve', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const client = createGoClient(req.bearerToken);
    await client.put(`/api/v1/approvals/${req.params.id}`, {
      decision: 'approve',
      reason: req.body?.reason,
    });
    res.status(204).end();
  } catch (err) {
    next(err);
  }
});

// POST /bff/api/v1/approvals/:id/reject
router.post('/:id/reject', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const client = createGoClient(req.bearerToken);
    await client.put(`/api/v1/approvals/${req.params.id}`, {
      decision: 'reject',
      reason: req.body?.reason,
    });
    res.status(204).end();
  } catch (err) {
    next(err);
  }
});

export default router;
