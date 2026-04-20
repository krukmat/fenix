// W6-T3: Wedge-first seed helper — removed workflow fixtures, added wedge-relevant
// run statuses (completed, handoff, denied_by_policy) and inbox approval.
import { execFileSync } from 'node:child_process';
import path from 'node:path';

export type WedgeSeed = {
  credentials: {
    email: string;
    password: string;
  };
  account: {
    id: string;
  };
  contact: {
    id: string;
    email: string;
  };
  deal: {
    id: string;
  };
  pipeline: {
    id: string;
  };
  stage: {
    id: string;
  };
  case: {
    id: string;
    subject: string;
  };
  agentRuns: {
    completedId: string;
    handoffId: string;
    deniedByPolicyId: string;
    rejectedId: string;
    caseRejectedId: string;
    dealRejectedId: string;
  };
  inbox: {
    approvalId: string;
  };
  workflows: {
    activeId: string;
    archivedId: string;
  };
};

let cachedSeed: WedgeSeed | null = null;

export function ensureWedgeSeed(): WedgeSeed {
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

  const parsed = JSON.parse(stdout) as Omit<WedgeSeed, 'workflows'> & {
    agentRuns: Omit<WedgeSeed['agentRuns'], 'rejectedId' | 'caseRejectedId' | 'dealRejectedId'>;
  };

  cachedSeed = {
    ...parsed,
    agentRuns: {
      ...parsed.agentRuns,
      rejectedId: parsed.agentRuns.deniedByPolicyId,
      caseRejectedId: parsed.agentRuns.deniedByPolicyId,
      dealRejectedId: parsed.agentRuns.deniedByPolicyId,
    },
    workflows: {
      activeId: '',
      archivedId: '',
    },
  };
  return cachedSeed;
}

// Backward-compatible alias while the legacy Detox suite is being retired.
export const ensureMobileP2Seed = ensureWedgeSeed;
