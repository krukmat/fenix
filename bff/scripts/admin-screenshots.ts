// admin-screenshots Task 5: entry point — orchestrates the full admin screenshot run
// BFF-ADMIN-Task6: smoke-run fix — seed policy set if missing (no POST endpoint exists)
import * as path from 'path';
import { execSync } from 'child_process';
import { assertDependenciesHealthy } from './snapshots/health';
import { runSeeder } from './snapshots/seeder';
import { runShooter } from './admin-screenshots/shooter';
import { generateReport } from './admin-screenshots/report';
import type { ResolvedIds } from './admin-screenshots/catalog';

const GO_URL  = process.env['FENIX_GO_URL']  ?? 'http://localhost:8080';
const BFF_URL = process.env['FENIX_BFF_URL'] ?? 'http://localhost:3000';

const REPO_ROOT  = path.resolve(__dirname, '..', '..');
const DB_PATH    = path.join(REPO_ROOT, 'data', 'fenixcrm.db');
const OUTPUT_DIR = path.join(REPO_ROOT, 'bff', 'artifacts', 'admin-screenshots');

// Ensures at least one policy_set row exists for the workspace so the policy versions route can be screenshotted.
// policy_set has no POST API (read-only handlers), so we insert directly via sqlite3 CLI.
function ensurePolicySet(workspaceId: string): string {
  const id = `00000000-0000-0000-0000-admin-shot-ps`;
  // description and created_by must be non-NULL — Go scanner uses string (not sql.NullString)
  const sql = `INSERT OR IGNORE INTO policy_set (id, workspace_id, name, description, is_active, created_by, created_at, updated_at) VALUES ('${id}', '${workspaceId}', 'Admin Screenshot Seed', '', 1, 'admin-screenshots', datetime('now'), datetime('now'));`;
  execSync(`sqlite3 "${DB_PATH}" "${sql}"`);
  return id;
}

async function resolveIds(goUrl: string, token: string, workspaceId: string): Promise<ResolvedIds> {
  process.stdout.write('[admin-screenshots] Resolving dynamic IDs...\n');

  const headers = {
    Authorization: `Bearer ${token}`,
    'X-Workspace-ID': workspaceId,
  };

  const [auditResp, policyResp] = await Promise.all([
    fetch(`${goUrl}/api/v1/audit/events?limit=1`, { headers }),
    fetch(`${goUrl}/api/v1/policy/sets?limit=1`, { headers }),
  ]);

  if (!auditResp.ok) throw new Error(`audit/events returned HTTP ${auditResp.status}`);
  if (!policyResp.ok) throw new Error(`policy/sets returned HTTP ${policyResp.status}`);

  const auditBody  = (await auditResp.json())  as { data: { id: string }[] };
  const policyBody = (await policyResp.json()) as { data: { id: string }[] };

  if (!auditBody.data?.[0]?.id)  throw new Error('No audit events found — run seeder first');

  // policy/sets may be empty (no write API exists); seed one via sqlite3 if needed
  const firstPolicySetId = policyBody.data?.[0]?.id ?? ensurePolicySet(workspaceId);

  return {
    firstAuditEventId: auditBody.data[0].id,
    firstPolicySetId,
  };
}

async function main(): Promise<void> {
  process.stdout.write('[admin-screenshots] Starting BFF admin screenshot run...\n');

  // Phase 1 — Health check
  await assertDependenciesHealthy(GO_URL, BFF_URL);
  process.stdout.write('[admin-screenshots] Dependencies healthy.\n');

  // Phase 2 — Seed
  process.stdout.write('[admin-screenshots] Running seeder...\n');
  const seed = await runSeeder(REPO_ROOT);
  process.stdout.write(`[admin-screenshots] Seed complete — workspace: ${seed.auth.workspaceId}\n`);

  // Phase 3 — Resolve IDs not in seeder output
  const resolved = await resolveIds(GO_URL, seed.auth.token, seed.auth.workspaceId);
  process.stdout.write(`[admin-screenshots] IDs resolved — audit: ${resolved.firstAuditEventId}, policy: ${resolved.firstPolicySetId}\n`);

  // Phase 4-6 — Launch browser, capture, close
  process.stdout.write('[admin-screenshots] Launching browser...\n');
  process.stdout.write('[admin-screenshots] Capturing 12 admin routes...\n');
  const results = await runShooter(BFF_URL, seed.auth.token, seed, resolved, OUTPUT_DIR);

  // Phase 7 — Report
  const passed  = results.filter((r) => r.ok).length;
  const failed  = results.filter((r) => !r.ok).length;
  process.stdout.write(`[admin-screenshots] Done — ${passed} captured, ${failed} failed\n`);

  generateReport(results, OUTPUT_DIR);
  process.stdout.write(`[admin-screenshots] Report → ${OUTPUT_DIR}/report.html\n`);
  process.stdout.write(`[admin-screenshots] Index  → ${OUTPUT_DIR}/index.md\n`);

  if (failed > 0) {
    process.exitCode = 1;
  }
}

main().catch((err: unknown) => {
  process.stderr.write(`[admin-screenshots] Fatal: ${String(err)}\n`);
  process.exit(1);
});
