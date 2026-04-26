// bff-http-snapshots T2: seeder integration — spawns Go seeder and parses output
import { spawn, ChildProcess } from 'child_process';
import path from 'path';
import type { SeederOutput } from './types';

// ChildLike allows tests to inject a mock process without spawning a real one
export type ChildLike = Pick<ChildProcess, 'stdout' | 'stderr'> & NodeJS.EventEmitter;

export function parseSeederOutput(raw: string): SeederOutput {
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    throw new Error(`Failed to parse seeder output: ${raw.slice(0, 200)}`);
  }

  const seed = parsed as Record<string, unknown>;
  const auth = seed['auth'] as Record<string, unknown> | undefined;
  const inbox = seed['inbox'] as Record<string, unknown> | undefined;

  if (!auth?.['token']) throw new Error('Seeder output missing auth.token');
  if (!auth?.['userId']) throw new Error('Seeder output missing auth.userId');
  if (!auth?.['workspaceId']) throw new Error('Seeder output missing auth.workspaceId');
  if (!inbox?.['approvalId']) throw new Error('Seeder output missing inbox.approvalId');
  if (!inbox?.['rejectApprovalId']) throw new Error('Seeder output missing inbox.rejectApprovalId');

  return parsed as SeederOutput;
}

export function spawnSeeder(repoRoot: string, child?: ChildLike): Promise<SeederOutput> {
  const proc: ChildLike = child ?? spawn(
    'go',
    ['run', './scripts/e2e_seed_mobile_p2.go'],
    { cwd: repoRoot },
  );

  return new Promise((resolve, reject) => {
    const stdoutChunks: Buffer[] = [];
    const stderrChunks: Buffer[] = [];

    proc.stdout?.on('data', (chunk: Buffer) => stdoutChunks.push(chunk));
    proc.stderr?.on('data', (chunk: Buffer) => stderrChunks.push(chunk));

    proc.on('close', (code: number | null) => {
      if (code !== 0) {
        const errText = Buffer.concat(stderrChunks).toString().trim();
        reject(new Error(`Seeder exited with code ${code ?? 'null'}. stderr: ${errText}`));
        return;
      }
      const raw = Buffer.concat(stdoutChunks).toString().trim();
      try {
        resolve(parseSeederOutput(raw));
      } catch (err) {
        reject(err);
      }
    });

    proc.on('error', (err: Error) => reject(err));
  });
}

// runSeeder spawns the real Go seeder from the repo root
export async function runSeeder(repoRoot?: string): Promise<SeederOutput> {
  const root = repoRoot ?? path.resolve(__dirname, '../../..');
  return spawnSeeder(root);
}
