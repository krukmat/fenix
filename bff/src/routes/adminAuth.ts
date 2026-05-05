// BFF-ADMIN-02: bearer-token relay middleware and 401 upstream redirect for admin HTMX surface
// BAL-01: session auth — login page render, credential relay, session type declaration
import { Request, Response, NextFunction } from 'express';
import axios from 'axios';
import { adminLoginPage } from './adminLoginLayout';
import { config } from '../config';

declare module 'express-session' {
  interface SessionData {
    adminToken?: string;
  }
}

const ADMIN_LOGIN_PATH = '/bff/admin/login';

/**
 * Redirects to the admin shell when an upstream Go response returns 401.
 * HTML surfaces must redirect rather than leak raw JSON error status to the browser.
 * Used by proxy handlers in Phase B onwards.
 */
export function redirectOnUpstream401(upstreamStatus: number, res: Response): void {
  if (upstreamStatus === 401) {
    res.redirect(ADMIN_LOGIN_PATH);
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

// BAL-02: session guard — redirects unauthenticated operators to the login page
export function requireAdminSession(req: Request, res: Response, next: NextFunction): void {
  const token = req.session.adminToken;
  if (!token) {
    res.redirect(ADMIN_LOGIN_PATH);
    return;
  }
  res.locals['adminToken'] = `Bearer ${token}`;
  next();
}

// BAL-01: render the login page (no auth required)
export function loginPage(_req: Request, res: Response): void {
  res.type('html').status(200).send(adminLoginPage());
}

function destroySession(req: Request): Promise<void> {
  return new Promise((resolve, reject) => {
    req.session.destroy((err) => {
      if (err) {
        reject(err);
        return;
      }
      resolve();
    });
  });
}

export async function invalidateAdminSession(req: Request, res: Response): Promise<void> {
  res.clearCookie('fenix.admin.sid');
  await destroySession(req);
  res.redirect(ADMIN_LOGIN_PATH);
}

// BAL-01: handle credential submission — relay to Go /auth/login, store token in session
export async function handleLogin(req: Request, res: Response): Promise<void> {
  const { email, password } = req.body as { email?: string; password?: string };

  try {
    const response = await axios.post<{ token: string }>(
      `${config.backendUrl}/auth/login`,
      { email, password },
      { timeout: 5000 },
    );

    const token = response.data.token;
    req.session.adminToken = token;

    res.redirect('/bff/admin');
  } catch (err: unknown) {
    const status = (err as { response?: { status?: number } }).response?.status;

    if (status === 401 || status === 403) {
      res.type('html').status(200).send(adminLoginPage('Invalid email or password'));
      return;
    }

    res.type('html').status(200).send(adminLoginPage('Auth service unavailable. Please try again.'));
  }
}

export async function handleLogout(req: Request, res: Response): Promise<void> {
  await invalidateAdminSession(req, res);
}
