// Task 4.1 — FR-301: BFF health check — reports BFF status + Go backend reachability
import { Router, Request, Response } from 'express';
import { pingGoBackend } from '../services/goClient';

const router = Router();

router.get('/', async (_req: Request, res: Response): Promise<void> => {
  const { reachable, latencyMs } = await pingGoBackend();

  if (reachable) {
    res.status(200).json({
      status: 'ok',
      backend: 'reachable',
      latency_ms: latencyMs,
    });
  } else {
    res.status(503).json({
      status: 'degraded',
      backend: 'unreachable',
    });
  }
});

export default router;
