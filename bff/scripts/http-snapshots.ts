// bff-http-snapshots T1+T9: entry point — orchestrates the full snapshot run
import path from 'path';
import { assertDependenciesHealthy } from './snapshots/health';
import { runSeeder } from './snapshots/seeder';
import { catalog } from './snapshots/catalog';
import { runEntries } from './snapshots/runner';
import { generateReport, loadArtifacts } from './snapshots/report';

const GO_URL  = process.env['FENIX_GO_URL']  ?? 'http://localhost:8080';
const BFF_URL = process.env['FENIX_BFF_URL'] ?? 'http://localhost:3000';
const SNAPSHOT_FIXTURE_MODE = process.env['FENIX_SNAPSHOTS_FIXTURE_MODE'] !== '0';

const REPO_ROOT  = path.resolve(__dirname, '..', '..');
const OUTPUT_DIR = path.join(REPO_ROOT, 'bff', 'artifacts', 'http-snapshots');
const RAW_DIR    = path.join(OUTPUT_DIR, 'raw');

async function main(): Promise<void> {
  process.stdout.write('[http-snapshots] Starting BFF HTTP snapshot run...\n');

  // 1 — Pre-flight: both services must be healthy before spending time on seeding
  await assertDependenciesHealthy(GO_URL, BFF_URL);
  process.stdout.write('[http-snapshots] Dependencies healthy.\n');

  if (SNAPSHOT_FIXTURE_MODE) {
    await setScreenshotMode(true);
  }

  try {
    // 2 — Seed: obtain real token + entity IDs from the Go seeder
    process.stdout.write('[http-snapshots] Running seeder...\n');
    const seed = await runSeeder(REPO_ROOT);
    process.stdout.write(`[http-snapshots] Seed complete — workspace: ${seed.auth.workspaceId}\n`);

    // 3 — Execute catalog entries (REST + SSE)
    process.stdout.write(`[http-snapshots] Executing ${catalog.length} catalog entries...\n`);
    const results = await runEntries(catalog, seed, BFF_URL, RAW_DIR);

    // 4 — Summary line
    const passed  = results.filter((r) => r.pass).length;
    const failed  = results.filter((r) => !r.pass).length;
    const errored = results.filter((r) => r.error).length;
    process.stdout.write(
      `[http-snapshots] Done — ${passed} passed, ${failed} failed, ${errored} errors\n`,
    );

    // 5 — Generate HTML report + index.md from the written artifacts
    process.stdout.write('[http-snapshots] Generating report...\n');
    const artifacts = loadArtifacts(RAW_DIR);
    generateReport(artifacts, OUTPUT_DIR);
    process.stdout.write(`[http-snapshots] Report → ${OUTPUT_DIR}/report.html\n`);
    process.stdout.write(`[http-snapshots] Index  → ${OUTPUT_DIR}/index.md\n`);

    // Exit non-zero if any entry failed (allows CI to detect regressions)
    if (failed > 0 || errored > 0) {
      process.exitCode = 1;
    }
  } finally {
    if (SNAPSHOT_FIXTURE_MODE) {
      await setScreenshotMode(false).catch((err: unknown) => {
        process.stderr.write(`[http-snapshots] Warning: failed to disable fixture mode: ${String(err)}\n`);
      });
    }
  }
}

async function setScreenshotMode(enabled: boolean): Promise<void> {
  const res = await fetch(`${BFF_URL}/bff/copilot/internal/screenshot-mode`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ enabled }),
  });
  if (!res.ok) {
    throw new Error(`screenshot-mode ${enabled ? 'enable' : 'disable'} failed with HTTP ${res.status}`);
  }
  process.stdout.write(`[http-snapshots] Fixture mode ${enabled ? 'enabled' : 'disabled'}.\n`);
}

main().catch((err: unknown) => {
  process.stderr.write(`[http-snapshots] Fatal: ${String(err)}\n`);
  process.exit(1);
});
