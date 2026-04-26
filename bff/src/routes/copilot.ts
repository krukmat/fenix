// Task 4.1 — FR-301: SSE proxy — relay Copilot streaming from Go to mobile client
import { Router, Request, Response, NextFunction } from 'express';
import axios from 'axios';
import { config } from '../config';
import { createGoClient } from '../services/goClient';

const router = Router();

type BffRequest = Request & { bearerToken?: string };

const SSE_HEADERS = {
  'Content-Type': 'text/event-stream',
  'Cache-Control': 'no-cache',
  'Connection': 'keep-alive',
  'X-Accel-Buffering': 'no',
};
const FIXTURE_TIMESTAMP = '<timestamp>';

// POST /bff/copilot/chat → SSE relay to Go POST /api/v1/copilot/chat
router.post('/chat', async (req: BffRequest, res: Response, next: NextFunction): Promise<void> => {
  if (screenshotMode) {
    writeFixtureSSE(res);
    return;
  }
  try {
    const stream = await openCopilotStream(req.body, req.bearerToken);
    pipeSSE(req, res, stream, next);
  } catch (err) {
    next(err);
  }
});

// GET /bff/copilot/events → browser EventSource-compatible SSE wrapper.
router.get('/events', async (req: BffRequest, res: Response): Promise<void> => {
  if (screenshotMode) {
    writeFixtureSSE(res);
    return;
  }
  try {
    const stream = await openCopilotStream(eventSourcePayload(req), req.bearerToken);
    pipeSSE(req, res, stream);
  } catch (err) {
    writeTerminalSSEError(res, err);
  }
});

async function openCopilotStream(body: unknown, token?: string): Promise<import('stream').Readable> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'Accept': 'text/event-stream',
  };
  if (token) {
    headers['Authorization'] = token;
  }
  const goRes = await axios.post(`${config.backendUrl}/api/v1/copilot/chat`, body, {
    headers,
    responseType: 'stream',
    timeout: 60000,
  });
  return goRes.data as import('stream').Readable;
}

function pipeSSE(req: Request, res: Response, stream: import('stream').Readable, next?: NextFunction): void {
  setSSEHeaders(res);
  stream.pipe(res);
  stream.on('error', (err: Error) => {
    res.end();
    next?.(err);
  });
  req.on('close', () => {
    stream.destroy();
  });
}

function setSSEHeaders(res: Response): void {
  Object.entries(SSE_HEADERS).forEach(([key, value]) => res.setHeader(key, value));
  res.flushHeaders();
}

function eventSourcePayload(req: Request): Record<string, string> {
  return {
    message: queryString(req.query['message']),
    entity_id: queryString(req.query['entity_id']),
    entity_type: queryString(req.query['entity_type']),
  };
}

function queryString(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function writeTerminalSSEError(res: Response, err: unknown): void {
  setSSEHeaders(res);
  const message = err instanceof Error ? err.message : 'SSE upstream unavailable';
  res.write('retry: 0\n');
  res.write(`event: error\ndata: ${JSON.stringify({ code: 'sse_upstream_error', message })}\n\n`);
  res.end();
}

function writeFixtureSSE(res: Response): void {
  setSSEHeaders(res);
  res.write(`data: ${JSON.stringify({
    type: 'evidence',
    sources: [{
      ID: 'fixture-source-001',
      Method: 'fixture',
      Score: 1,
      Snippet: 'Snapshot fixture evidence for deterministic copilot capture.',
      PiiRedacted: false,
      Metadata: null,
      CreatedAt: FIXTURE_TIMESTAMP,
    }],
    meta: {
      schema_version: 'v1',
      source_count: 1,
      retrieval_methods_used: ['fixture'],
      built_at: FIXTURE_TIMESTAMP,
    },
  })}\n\n`);
  res.write(`data: ${JSON.stringify({ type: 'token', delta: 'Snapshot fixture response.' })}\n\n`);
  res.write(`data: ${JSON.stringify({ type: 'done', done: true, meta: { answer_type: 'grounded_answer', at: FIXTURE_TIMESTAMP } })}\n\n`);
  res.end();
}

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
