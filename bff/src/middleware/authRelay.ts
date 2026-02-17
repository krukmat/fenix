// Task 4.1 — FR-301: Extract and relay Authorization header to Go backend
import { Request, Response, NextFunction } from 'express';

// Attach the bearer token to the request for use by aggregated/copilot routes
export function authRelay(req: Request, _res: Response, next: NextFunction): void {
  const authHeader = req.headers['authorization'];
  // Store token for downstream use (aggregated/copilot routes need it for Go calls)
  if (typeof authHeader === 'string' && authHeader.startsWith('Bearer ')) {
    (req as Request & { bearerToken?: string }).bearerToken = authHeader;
  }
  next();
}
