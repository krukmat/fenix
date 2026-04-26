// bff-http-snapshots T1: health gate — verifies Go backend and BFF are reachable
import type { HealthCheckResult } from './types';

export async function checkHealth(url: string, name: string): Promise<HealthCheckResult> {
  try {
    const res = await fetch(url, { signal: AbortSignal.timeout(3000) });
    if (!res.ok) {
      return { ok: false, url, error: `${name} returned HTTP ${res.status}` };
    }
    return { ok: true, url };
  } catch {
    return { ok: false, url, error: `${name} unreachable at ${url}` };
  }
}

export async function assertDependenciesHealthy(
  goUrl: string,
  bffUrl: string,
): Promise<void> {
  const [go, bff] = await Promise.all([
    checkHealth(`${goUrl}/health`, 'Go backend'),
    checkHealth(`${bffUrl}/bff/health`, 'BFF'),
  ]);

  const failed: string[] = [];
  if (!go.ok) failed.push(`  • Go backend: ${go.error}`);
  if (!bff.ok) failed.push(`  • BFF: ${bff.error}`);

  if (failed.length > 0) {
    process.stderr.write(
      `[http-snapshots] Dependency health check failed:\n${failed.join('\n')}\n` +
      `  Start Go: JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080\n` +
      `  Start BFF: cd bff && npm run dev\n`,
    );
    process.exit(1);
  }
}
