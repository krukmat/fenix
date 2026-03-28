import type { Request, Response, NextFunction } from 'express';
import type { RequestHandler } from 'http-proxy-middleware';

/**
 * Creates a typed proxy stub for testing http-proxy-middleware.
 * Replaces the pattern of:
 *   jest.fn((_req: any, _res: any, next: any) => next())
 *   Object.assign(fn, { upgrade: () => {} }) as any
 *
 * Usage:
 *   const proxyStub = makeProxyStub();
 *   jest.mock('http-proxy-middleware', () => ({
 *     createProxyMiddleware: jest.fn(() => proxyStub),
 *   }));
 */
export function makeProxyStub(): RequestHandler {
  const fn = jest.fn((_req: Request, _res: Response, next: NextFunction) => next());
  return Object.assign(fn, { upgrade: () => {} }) as unknown as RequestHandler;
}
