// Task 4.1 — FR-301: Transparent proxy — all /bff/api/v1/* forwarded to Go /api/v1/*
import { Router } from 'express';
import { createProxyMiddleware } from 'http-proxy-middleware';
import { config } from '../config';

const router = Router();

// Pass-through: all methods, all paths under /bff/api/v1
// http-proxy-middleware rewrites /bff/api/v1/... → /api/v1/...
router.use(
  '/',
  createProxyMiddleware({
    target: config.backendUrl,
    changeOrigin: true,
    pathRewrite: { '^/bff/api/v1': '/api/v1' },
    on: {
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
