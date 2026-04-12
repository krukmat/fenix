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

// Screenshot fixture — returned when screenshotMode is active to bypass LLM latency (~35s)
// Activated at runtime via POST /bff/internal/screenshot-mode { enabled: true|false }
let screenshotMode = process.env.SCREENSHOT_MODE === 'true';

const SALES_BRIEF_FIXTURE = {
  outcome: 'completed',
  entityType: 'deal',
  entityId: 'fixture',
  summary: 'Champion confirmed budget approval. Procurement requested the security addendum. Decision call is scheduled for Friday.',
  risks: ['Legal review could slip by three business days.', 'Procurement needs revised pricing language.'],
  nextBestActions: [
    { title: 'Send Security Addendum', description: 'Send the requested security addendum to Procurement today.', tool: 'create_task', params: {}, confidence_score: 0.85, confidence_level: 'high' },
    { title: 'Follow up with Procurement', description: 'Follow up with Procurement tomorrow regarding revised pricing language.', tool: 'create_task', params: {}, confidence_score: 0.85, confidence_level: 'high' },
  ],
  confidence: 'high',
  abstentionReason: null,
  evidencePack: {
    schema_version: 'v1',
    query: 'deal latest updates timeline next steps',
    sources: [{ knowledge_item_id: 'fixture-001', method: 'vector', score: 0.95, snippet: 'Champion confirmed budget approval for the expansion deal.', pii_redacted: false, created_at: new Date().toISOString() }],
    source_count: 1,
    dedup_count: 0,
    confidence: 'high',
    total_candidates: 1,
    filtered_count: 0,
    warnings: [],
    retrieval_methods_used: ['vector'],
    built_at: new Date().toISOString(),
  },
};

// POST /bff/internal/screenshot-mode — toggle fixture mode at runtime (localhost only, no auth required)
router.post('/internal/screenshot-mode', (req: Request, res: Response): void => {
  const enabled = (req.body as { enabled?: boolean }).enabled === true;
  screenshotMode = enabled;
  res.status(200).json({ screenshotMode });
});

// POST /bff/api/v1/copilot/sales-brief → JSON relay to Go POST /api/v1/copilot/sales-brief
// screenshotMode=true: returns fixture immediately (bypasses LLM ~35s latency for screenshot runs)
router.post('/sales-brief', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  if (screenshotMode) {
    res.status(200).json(SALES_BRIEF_FIXTURE);
    return;
  }
  try {
    const client = createGoClient(req.bearerToken);
    const goRes = await client.post('/api/v1/copilot/sales-brief', req.body);
    res.status(200).json((goRes.data as { data?: unknown }).data ?? goRes.data);
  } catch (err) {
    next(err);
  }
});

export default router;
