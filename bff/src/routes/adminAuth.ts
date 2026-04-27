// BFF-ADMIN-02: bearer-token relay middleware and 401 upstream redirect for admin HTMX surface
import { Request, Response, NextFunction } from 'express';

/**
 * Redirects to the admin shell when an upstream Go response returns 401.
 * HTML surfaces must redirect rather than leak raw JSON error status to the browser.
 * Used by proxy handlers in Phase B onwards.
 */
export function redirectOnUpstream401(upstreamStatus: number, res: Response): void {
  if (upstreamStatus === 401) {
    res.redirect('/bff/admin');
  }
}

/**
 * Middleware: attaches the bearer token from the Authorization header to
 * res.locals so admin proxy handlers (Phase B+) can forward it without
 * re-reading the header in each route.
 */
export function adminBearerRelay(req: Request, res: Response, next: NextFunction): void {
  const auth = req.headers['authorization'];
  if (typeof auth === 'string' && auth.startsWith('Bearer ')) {
    res.locals['adminToken'] = auth;
  }
  next();
}

/** Returns the HTTP status from an Axios-like upstream error, or undefined. */
export function upstreamStatus(err: unknown): number | undefined {
  return (err as { response?: { status?: number } }).response?.status;
}

/** Extracts the message string from an upstream error response body. */
export function upstreamMessage(err: unknown): string {
  const data = (err as { response?: { data?: { message?: string } } }).response?.data;
  return data?.message ?? (err instanceof Error ? err.message : 'Unknown error');
}
