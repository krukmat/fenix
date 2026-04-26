// bff-http-snapshots T5+T6: REST runner + SSE routing
import fs from 'fs';
import path from 'path';
import type { CatalogEntry, SeederOutput, SnapshotArtifact } from './types';
import { redactObject } from './redact';
import { captureSSE } from './sse-capture';

export type EntryResult = {
  name: string;
  pass: boolean;
  status: number;
  latencyMs: number;
  error?: string;
};

const SNAPSHOT_IP = `10.244.${process.pid % 250}.${Date.now() % 250}`;

function resolveBody(entry: CatalogEntry, seed: SeederOutput): unknown {
  if (typeof entry.body === 'function') return entry.body(seed);
  return entry.body;
}

function resolvePath(entry: CatalogEntry, seed: SeederOutput): string {
  if (!entry.pathParams) return entry.path;
  const params = entry.pathParams(seed);
  return entry.path.replace(/:([a-zA-Z]+)/g, (_, key: string) => params[key] ?? `:${key}`);
}

function pickResponseHeaders(headers: Headers): Record<string, string> {
  const keep = ['content-type', 'x-request-id', 'cache-control'];
  const result: Record<string, string> = {};
  for (const key of keep) {
    const val = headers.get(key);
    if (val) result[key] = val;
  }
  return result;
}

async function executeEntry(
  entry: CatalogEntry,
  seed: SeederOutput,
  baseUrl: string,
): Promise<{ artifact: SnapshotArtifact; pass: boolean; latencyMs: number }> {
  const resolvedPath = resolvePath(entry, seed);
  const url = `${baseUrl}${resolvedPath}`;
  const body = resolveBody(entry, seed);

  const requestHeaders: Record<string, string> = {
    'content-type': 'application/json',
    'x-mobile-platform': 'snapshot-runner',
    'x-real-ip': SNAPSHOT_IP,
  };
  if (entry.auth) {
    requestHeaders['authorization'] = `Bearer ${seed.auth.token}`;
  }

  const fetchInit: RequestInit = {
    method: entry.method,
    headers: requestHeaders,
  };
  if (body !== undefined && entry.method !== 'GET') {
    fetchInit.body = JSON.stringify(body);
  }

  const start = performance.now();
  const res = await fetch(url, fetchInit);
  const latencyMs = Math.round(performance.now() - start);

  let responseBody: unknown;
  const contentType = res.headers.get('content-type') ?? '';
  if (contentType.includes('application/json')) {
    try { responseBody = await res.json(); } catch { responseBody = null; }
  } else {
    responseBody = await res.text();
  }

  const artifact: SnapshotArtifact = redactObject({
    name: entry.name,
    method: entry.method,
    path: resolvedPath,
    expectedStatus: entry.expectedStatus,
    request: {
      headers: requestHeaders,
      body: body !== undefined ? body : undefined,
    },
    response: {
      status: res.status,
      headers: pickResponseHeaders(res.headers),
      body: responseBody,
    },
    latencyMs: '<duration>',
    capturedAt: new Date().toISOString(),
  }) as SnapshotArtifact;

  return { artifact, pass: res.status === entry.expectedStatus, latencyMs };
}

async function executeSSEEntry(
  entry: CatalogEntry,
  seed: SeederOutput,
  baseUrl: string,
  groupDir: string,
): Promise<EntryResult> {
  const resolvedPath = resolvePath(entry, seed);
  const url = `${baseUrl}${resolvedPath}`;
  const sseOpts = entry.sse!;

  const requestHeaders: Record<string, string> = {};
  if (entry.auth) {
    requestHeaders['authorization'] = `Bearer ${seed.auth.token}`;
  }

  const start = performance.now();
  const sseEvents = await captureSSE(url, requestHeaders, sseOpts);
  const latencyMs = Math.round(performance.now() - start);

  const artifact = redactObject({
    name: entry.name,
    method: entry.method,
    path: resolvedPath,
    expectedStatus: entry.expectedStatus,
    request: { headers: requestHeaders },
    response: {
      status: sseEvents.length > 0 ? 200 : 0,
      headers: {},
      body: sseEvents,
    },
    latencyMs: '<duration>',
    capturedAt: new Date().toISOString(),
  }) as SnapshotArtifact;

  const filePath = `${groupDir}/${entry.name}.sse.json`;
  fs.writeFileSync(filePath, JSON.stringify(artifact, null, 2), 'utf8');

  const pass = sseEvents.length > 0;
  const line = pass ? '✓' : '✗';
  process.stdout.write(
    `[http-snapshots] ${line} ${entry.name.padEnd(40)} SSE  ${sseEvents.length} events  ${latencyMs}ms\n`,
  );

  return { name: entry.name, pass, status: pass ? 200 : 0, latencyMs };
}

export async function runEntries(
  entries: CatalogEntry[],
  seed: SeederOutput,
  baseUrl: string,
  outputDir: string,
): Promise<EntryResult[]> {
  const results: EntryResult[] = [];

  for (const entry of entries) {
    const groupDir = path.join(outputDir, entry.group);
    fs.mkdirSync(groupDir, { recursive: true });

    let result: EntryResult;

    if (entry.sse) {
      result = await executeSSEEntry(entry, seed, baseUrl, groupDir);
    } else {
      try {
        const { artifact, pass, latencyMs } = await executeEntry(entry, seed, baseUrl);
        const filePath = path.join(groupDir, `${entry.name}.json`);
        fs.writeFileSync(filePath, JSON.stringify(artifact, null, 2), 'utf8');

        const line = pass ? '✓' : '✗';
        process.stdout.write(
          `[http-snapshots] ${line} ${entry.name.padEnd(40)} ${artifact.response.status}  ${latencyMs}ms\n`,
        );

        result = { name: entry.name, pass, status: artifact.response.status, latencyMs };
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        process.stderr.write(`[http-snapshots] ✗ ${entry.name} — ERROR: ${message}\n`);
        result = { name: entry.name, pass: false, status: 0, latencyMs: 0, error: message };
      }
    }

    results.push(result);
  }

  return results;
}
