// Task 4.1 — FR-301: Pure helpers for restreaming parsed JSON bodies to proxy requests
import type { Request } from 'express';
import type { ClientRequest } from 'http';

export function hasParsedJsonBody(req: Request): boolean {
  return req.body !== undefined && req.body !== null && Object.keys(req.body as object).length > 0;
}

export function restreamParsedJsonBody(proxyReq: ClientRequest, req: Request): void {
  if (!hasParsedJsonBody(req)) {
    return;
  }
  const body = JSON.stringify(req.body);
  proxyReq.setHeader('Content-Type', 'application/json');
  proxyReq.setHeader('Content-Length', Buffer.byteLength(body));
  proxyReq.write(body);
}
