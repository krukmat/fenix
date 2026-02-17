// Task 4.1 — FR-301: Auth relay routes — forward login/register to Go backend
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';

const router = Router();

// POST /bff/auth/login → relay to Go POST /auth/login
router.post('/login', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    const client = createGoClient();
    const { data, status } = await client.post('/auth/login', req.body);
    res.status(status).json(data);
  } catch (err) {
    next(err);
  }
});

// POST /bff/auth/register → relay to Go POST /auth/register
router.post('/register', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  try {
    const client = createGoClient();
    const { data, status } = await client.post('/auth/register', req.body);
    res.status(status).json(data);
  } catch (err) {
    next(err);
  }
});

export default router;
