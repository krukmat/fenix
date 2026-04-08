// Task 4.1 — FR-301: SSE proxy — relay Copilot streaming from Go to mobile client
import { Router, Request, Response, NextFunction } from 'express';
import axios from 'axios';
import { config } from '../config';
import { createGoClient } from '../services/goClient';

const router = Router();

type BffRequest = Request & { bearerToken?: string };

// POST /bff/copilot/chat → SSE relay to Go POST /api/v1/copilot/chat
router.post('/chat', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const token = req.bearerToken;

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream',
    };
    if (token) {
      headers['Authorization'] = token;
    }

    const goRes = await axios.post(`${config.backendUrl}/api/v1/copilot/chat`, req.body, {
      headers,
      responseType: 'stream',
      timeout: 60000, // SSE streams can be long
    });

    // Set SSE headers before streaming
    res.setHeader('Content-Type', 'text/event-stream');
    res.setHeader('Cache-Control', 'no-cache');
    res.setHeader('Connection', 'keep-alive');
    res.setHeader('X-Accel-Buffering', 'no'); // Disable nginx buffering
    res.flushHeaders();

    // Relay chunks from Go to mobile client
    const stream = goRes.data as import('stream').Readable;
    stream.pipe(res);

    stream.on('error', (err: Error) => {
      // Stream error after headers sent — can only end the response
      res.end();
      next(err);
    });

    req.on('close', () => {
      // Client disconnected — destroy Go stream
      stream.destroy();
    });
  } catch (err) {
    next(err);
  }
});

// POST /bff/api/v1/copilot/sales-brief → JSON relay to Go POST /api/v1/copilot/sales-brief
router.post('/sales-brief', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  try {
    const client = createGoClient(req.bearerToken);
    const goRes = await client.post('/api/v1/copilot/sales-brief', req.body);
    res.status(200).json((goRes.data as { data?: unknown }).data ?? goRes.data);
  } catch (err) {
    next(err);
  }
});

export default router;
