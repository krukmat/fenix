// Task 4.1 — FR-301: Normalize errors from Go backend or BFF internals into mobile-friendly envelope
import { Request, Response, NextFunction } from 'express';
import { AxiosError } from 'axios';

interface ErrorEnvelope {
  error: {
    code: string;
    message: string;
    details?: unknown;
  };
}

function buildErrorEnvelope(code: string, message: string, details?: unknown): ErrorEnvelope {
  return { error: { code, message, ...(details !== undefined ? { details } : {}) } };
}

// Detect Axios errors: instanceof check OR isAxiosError flag (the latter works with jest.fn mocks)
function isAxiosLikeError(err: unknown): err is AxiosError {
  return err instanceof AxiosError || (err instanceof Error && 'isAxiosError' in err && (err as { isAxiosError: unknown }).isAxiosError === true);
}

// Express 5 error handler: 4 params required
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function errorHandler(err: unknown, _req: Request, res: Response, _next: NextFunction): void {
  if (isAxiosLikeError(err)) {
    const axiosErr = err as AxiosError<{ message?: string }>;
    const status = axiosErr.response?.status ?? 502;
    const data = axiosErr.response?.data;
    const message = data?.message ?? axiosErr.message ?? 'Go backend error';
    res.status(status).json(buildErrorEnvelope('BACKEND_ERROR', message, data));
    return;
  }

  if (err instanceof Error) {
    res.status(500).json(buildErrorEnvelope('INTERNAL_ERROR', err.message));
    return;
  }

  res.status(500).json(buildErrorEnvelope('UNKNOWN_ERROR', 'An unexpected error occurred'));
}
