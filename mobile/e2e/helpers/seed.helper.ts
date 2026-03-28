// Seed helper for Mobile P2 Detox smoke suites.
// Uses a repo-local Go helper so the tests run against deterministic SQLite data.

import { execFileSync } from 'node:child_process';
import path from 'node:path';

type MobileP2Seed = {
  credentials: {
    email: string;
    password: string;
  };
  account: {
    id: string;
  };
  deal: {
    id: string;
  };
  case: {
    id: string;
  };
  workflows: {
    activeId: string;
    archivedId: string;
  };
  agentRuns: {
    rejectedId: string;
    dealRejectedId: string;
    caseRejectedId: string;
  };
};

let cachedSeed: MobileP2Seed | null = null;

export function ensureMobileP2Seed(): MobileP2Seed {
  if (cachedSeed) return cachedSeed;

  const repoRoot = path.resolve(__dirname, '../../..');
  const stdout = execFileSync('go', ['run', './scripts/e2e_seed_mobile_p2.go'], {
    cwd: repoRoot,
    encoding: 'utf8',
    env: {
      ...process.env,
      API_URL: process.env.API_URL || 'http://localhost:8080',
      DATABASE_URL: process.env.DATABASE_URL || './data/fenixcrm.db',
    },
  });

  cachedSeed = JSON.parse(stdout) as MobileP2Seed;
  return cachedSeed;
}
