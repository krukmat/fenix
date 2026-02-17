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
} as const;

// Validate at startup (only BACKEND_URL is truly required)
export function validateConfig(): void {
  requireEnv('BACKEND_URL');
}
