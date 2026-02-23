// Task 4.9 — NFR-030: BFF Prometheus-compatible metrics
import { Router, Request, Response } from 'express';

let requestCount = 0;
let errorCount = 0;
const startTime = Date.now();

export function incRequests(): void {
  requestCount++;
}

export function incErrors(): void {
  errorCount++;
}

const router = Router();

router.get('/', (_req: Request, res: Response): void => {
  const uptime = (Date.now() - startTime) / 1000;
  res.set('Content-Type', 'text/plain; version=0.0.4');
  res.send(
    `# HELP bff_requests_total Total BFF HTTP requests\n` +
      `# TYPE bff_requests_total counter\n` +
      `bff_requests_total ${requestCount}\n` +
      `# HELP bff_request_errors_total Total BFF HTTP errors\n` +
      `# TYPE bff_request_errors_total counter\n` +
      `bff_request_errors_total ${errorCount}\n` +
      `# HELP bff_uptime_seconds BFF process uptime\n` +
      `# TYPE bff_uptime_seconds gauge\n` +
      `bff_uptime_seconds ${uptime.toFixed(2)}\n`
  );
});

export default router;