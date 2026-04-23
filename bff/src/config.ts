// Task 4.1 — FR-301: BFF configuration from environment variables
import 'dotenv/config';

function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

export const config = {
  port: parseInt(process.env['BFF_PORT'] ?? '3000', 10),
  backendUrl: process.env['BACKEND_URL'] ?? 'http://localhost:8080',
  nodeEnv: process.env['NODE_ENV'] ?? 'development',
  isProduction: process.env['NODE_ENV'] === 'production',
  corsAllowedOrigins: parseAllowedOrigins(process.env['BFF_CORS_ALLOWED_ORIGINS']),
} as const;

// Validate at startup (only BACKEND_URL is truly required)
export function validateConfig(): void {
  requireEnv('BACKEND_URL');
}

function parseAllowedOrigins(value: string | undefined): string[] {
  const configured = splitCSV(value);
  if (configured.length > 0) {
    return configured;
  }
  return uniqueStrings([
    'http://localhost:3000',
    'http://localhost:3001',
    'http://localhost:5173',
    'http://127.0.0.1:3000',
    'http://127.0.0.1:3001',
    'http://127.0.0.1:5173',
    'exp://127.0.0.1:8081',
    'http://localhost:8081',
  ]);
}

function splitCSV(value: string | undefined): string[] {
  return (value ?? '')
    .split(',')
    .map((origin) => origin.trim())
    .filter((origin) => origin.length > 0);
}

function uniqueStrings(values: string[]): string[] {
  return Array.from(new Set(values));
}
