// Task 4.1 — FR-301: Forward mobile-specific headers to Go backend
import { Request, Response, NextFunction } from 'express';

// Headers sent by the React Native app that Go backend may use for analytics/tracing
const MOBILE_HEADERS = ['x-device-id', 'x-app-version', 'x-platform', 'x-os-version'] as const;

export function mobileHeaders(req: Request, _res: Response, next: NextFunction): void {
  // Store mobile headers on req for downstream use (proxy + aggregated routes forward them)
  const forwardedHeaders: Record<string, string> = {};
  for (const header of MOBILE_HEADERS) {
    const value = req.headers[header];
    if (typeof value === 'string') {
      forwardedHeaders[header] = value;
    }
  }
  // Attach to request for use by aggregated/copilot routes
  (req as Request & { mobileHeaders?: Record<string, string> }).mobileHeaders = forwardedHeaders;
  next();
}
