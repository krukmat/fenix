import { execFileSync } from 'node:child_process';
import { ensureMobileP2Seed } from './seed.helper';

type MobileP2Seed = ReturnType<typeof ensureMobileP2Seed>;
type BffSession = { token: string; userId: string; workspaceId: string };
type BffListResponse<T> = { data: T[] };
type SignalRecord = { id: string };
type CaseRecord = { subject?: string };

const seeded: MobileP2Seed = ensureMobileP2Seed();
let cachedSession: BffSession | null = null;
let cachedSignalId: string | null | undefined;
let cachedCaseSubject: string | null = null;

function runCurl<T>(args: string[]): T {
  return JSON.parse(execFileSync('curl', ['-fsS', ...args], { encoding: 'utf8' }).trim()) as T;
}

function ensureBffSession(): BffSession {
  if (cachedSession) return cachedSession;
  cachedSession = runCurl<BffSession>([
    '-X', 'POST', 'http://localhost:3000/bff/auth/login', '-H', 'Content-Type: application/json', '--data',
    JSON.stringify({ email: seeded.credentials.email, password: seeded.credentials.password }),
  ]);
  return cachedSession;
}

function getBff<T>(path: string): T {
  const session = ensureBffSession();
  return runCurl<T>(['-H', `Authorization: Bearer ${session.token}`, `http://localhost:3000${path}`]);
}

export function ensureFirstActiveSignalId(): string | null {
  if (cachedSignalId !== undefined) return cachedSignalId;
  try {
    const session = ensureBffSession();
    const params = new URLSearchParams({ workspace_id: session.workspaceId, status: 'active' });
    cachedSignalId = getBff<BffListResponse<SignalRecord>>(`/bff/api/v1/signals?${params.toString()}`).data[0]?.id ?? null;
  } catch {
    cachedSignalId = null;
  }
  return cachedSignalId;
}

export function ensureSeededCaseSubject(caseId: string): string {
  if (cachedCaseSubject) return cachedCaseSubject;
  cachedCaseSubject = getBff<CaseRecord>(`/bff/api/v1/cases/${caseId}`).subject ?? '';
  return cachedCaseSubject;
}
