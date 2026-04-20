// Task 4.1 — FR-301: Transparent proxy — all /bff/api/v1/* forwarded to Go /api/v1/*
import { Router } from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import type { Request } from 'express';
import { config } from '../config';

const router = Router();

export function hasParsedJsonBody(req: Request): boolean {
  return req.body !== undefined && req.body !== null && Object.keys(req.body as object).length > 0;
}

export function restreamParsedJsonBody(proxyReq: import('http').ClientRequest, req: Request): void {
  if (!hasParsedJsonBody(req)) {
    return;
  }
  const body = JSON.stringify(req.body);
  proxyReq.setHeader('Content-Type', 'application/json');
  proxyReq.setHeader('Content-Length', Buffer.byteLength(body));
  proxyReq.write(body);
}

// Pass-through: all methods, all paths under /bff/api/v1
// http-proxy-middleware rewrites /bff/api/v1/... → /api/v1/...
router.use(
  '/',
  createProxyMiddleware({
    target: config.backendUrl,
    changeOrigin: true,
    // Router is mounted at /bff/api/v1 in app.ts; at this point req.url is usually /<resource>.
    // Prefix all forwarded paths with /api/v1 to match Go backend routes.
    pathRewrite: (path) => `/api/v1${path}`,
    on: {
      proxyReq: restreamParsedJsonBody,
      // istanbul ignore next — proxy error handler only reachable with live network failure
      error: (err, _req, res) => {
        /* istanbul ignore next */
        if ('status' in res && typeof res.status === 'function') {
          (res as import('express').Response).status(502).json({
            error: {
              code: 'PROXY_ERROR',
              message: 'Go backend unreachable',
              details: (err as Error).message,
            },
          });
        }
      },
    },
  }),
);

export default router;
