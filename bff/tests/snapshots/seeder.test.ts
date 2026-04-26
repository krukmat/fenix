// bff-http-snapshots T2: seeder integration tests — written before implementation (TDD)
import { EventEmitter } from 'events';
import { spawnSeeder, parseSeederOutput } from '../../scripts/snapshots/seeder';

// Minimal canonical seed output matching scripts/e2e_seed_mobile_p2.go seedOutput
const CANONICAL_SEED: Record<string, unknown> = {
  credentials: { email: 'e2e@fenixcrm.test', password: 'e2eTestPass123!' },
  auth: { token: 'tok-abc', userId: 'uid-1', workspaceId: 'ws-1' },
  account: { id: 'acc-1' },
  contact: { id: 'con-1', email: 'contact@test.com' },
  lead: { id: 'lead-1' },
  deal: { id: 'deal-1' },
  pipeline: { id: 'pipe-1' },
  stage: { id: 'stage-1' },
  staleDeal: { id: 'stale-1' },
  case: { id: 'case-1', subject: 'Test case' },
  resolvedCase: { id: 'rcase-1', subject: 'Resolved case' },
  agentRuns: { completedId: 'run-1', handoffId: 'run-2', deniedByPolicyId: 'run-3' },
  inbox: { approvalId: 'apr-1', rejectApprovalId: 'apr-2', signalId: 'sig-1' },
  workflow: { id: 'wf-1' },
};

describe('parseSeederOutput', () => {
  it('parses canonical seed JSON into SeederOutput', () => {
    const result = parseSeederOutput(JSON.stringify(CANONICAL_SEED));

    expect(result.auth.token).toBe('tok-abc');
    expect(result.auth.userId).toBe('uid-1');
    expect(result.auth.workspaceId).toBe('ws-1');
    expect(result.inbox.approvalId).toBe('apr-1');
    expect(result.credentials.email).toBe('e2e@fenixcrm.test');
  });

  it('throws on invalid JSON', () => {
    expect(() => parseSeederOutput('not-json')).toThrow(/Failed to parse seeder output/);
  });

  it('throws when auth.token is missing', () => {
    const bad = { ...CANONICAL_SEED, auth: { userId: 'uid-1', workspaceId: 'ws-1' } };
    expect(() => parseSeederOutput(JSON.stringify(bad))).toThrow(/auth\.token/);
  });

  it('throws when auth.userId is missing', () => {
    const bad = { ...CANONICAL_SEED, auth: { token: 'tok', workspaceId: 'ws-1' } };
    expect(() => parseSeederOutput(JSON.stringify(bad))).toThrow(/auth\.userId/);
  });

  it('throws when auth.workspaceId is missing', () => {
    const bad = { ...CANONICAL_SEED, auth: { token: 'tok', userId: 'uid-1' } };
    expect(() => parseSeederOutput(JSON.stringify(bad))).toThrow(/auth\.workspaceId/);
  });

  it('throws when inbox.approvalId is missing', () => {
    const bad = { ...CANONICAL_SEED, inbox: { rejectApprovalId: 'apr-2', signalId: 'sig-1' } };
    expect(() => parseSeederOutput(JSON.stringify(bad))).toThrow(/inbox\.approvalId/);
  });

  it('throws when inbox.rejectApprovalId is missing', () => {
    const bad = { ...CANONICAL_SEED, inbox: { approvalId: 'apr-1', signalId: 'sig-1' } };
    expect(() => parseSeederOutput(JSON.stringify(bad))).toThrow(/inbox\.rejectApprovalId/);
  });
});

describe('spawnSeeder', () => {
  it('resolves with parsed SeederOutput when process exits 0', async () => {
    const mockChild = new EventEmitter() as NodeJS.EventEmitter & {
      stdout: EventEmitter;
      stderr: EventEmitter;
    };
    mockChild.stdout = new EventEmitter();
    mockChild.stderr = new EventEmitter();

    const promise = spawnSeeder('/fake/repo', mockChild as Parameters<typeof spawnSeeder>[1]);

    mockChild.stdout.emit('data', Buffer.from(JSON.stringify(CANONICAL_SEED)));
    mockChild.emit('close', 0);

    const result = await promise;
    expect(result.auth.token).toBe('tok-abc');
    expect(result.inbox.approvalId).toBe('apr-1');
  });

  it('rejects with stderr content when process exits non-zero', async () => {
    const mockChild = new EventEmitter() as NodeJS.EventEmitter & {
      stdout: EventEmitter;
      stderr: EventEmitter;
    };
    mockChild.stdout = new EventEmitter();
    mockChild.stderr = new EventEmitter();

    const promise = spawnSeeder('/fake/repo', mockChild as Parameters<typeof spawnSeeder>[1]);

    mockChild.stderr.emit('data', Buffer.from('go: build error'));
    mockChild.emit('close', 1);

    await expect(promise).rejects.toThrow(/Seeder exited with code 1/);
  });

  it('rejects when process emits error event', async () => {
    const mockChild = new EventEmitter() as NodeJS.EventEmitter & {
      stdout: EventEmitter;
      stderr: EventEmitter;
    };
    mockChild.stdout = new EventEmitter();
    mockChild.stderr = new EventEmitter();

    const promise = spawnSeeder('/fake/repo', mockChild as Parameters<typeof spawnSeeder>[1]);

    mockChild.emit('error', new Error('ENOENT: go not found'));

    await expect(promise).rejects.toThrow(/ENOENT/);
  });
});
