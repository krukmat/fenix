// bff-http-snapshots T7: HTML report generator tests — written before implementation (TDD)
import fs from 'fs';
import os from 'os';
import path from 'path';
import { generateReport } from '../../scripts/snapshots/report';
import type { SnapshotArtifact } from '../../scripts/snapshots/types';

// ── Fixtures ────────────────────────────────────────────────────────────────

function makeArtifact(overrides: Partial<SnapshotArtifact> & { name: string; group: string }): SnapshotArtifact & { group: string } {
  return {
    name: overrides.name,
    group: overrides.group,
    method: overrides.method ?? 'GET',
    path: overrides.path ?? '/bff/health',
    expectedStatus: overrides.expectedStatus,
    request: overrides.request ?? { headers: { 'content-type': 'application/json' } },
    response: overrides.response ?? { status: 200, headers: {}, body: { ok: true } },
    latencyMs: overrides.latencyMs ?? 42,
    capturedAt: overrides.capturedAt ?? '<timestamp>',
  };
}

const ARTIFACT_HEALTH = makeArtifact({
  name: 'health',
  group: 'root',
  path: '/bff/health',
  expectedStatus: 200,
  response: { status: 200, headers: { 'content-type': 'application/json' }, body: { ok: true } },
  latencyMs: 15,
});

const ARTIFACT_LOGIN_FAIL = makeArtifact({
  name: 'auth-login-invalid',
  group: 'auth',
  method: 'POST',
  path: '/bff/auth/login',
  expectedStatus: 401,
  response: { status: 401, headers: {}, body: { error: 'invalid credentials' } },
  latencyMs: 88,
});

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Counts opening tags (e.g. <div>) and closing tags (e.g. </div>) for a given tag name. */
function countTag(html: string, tag: string): { open: number; close: number } {
  const openRe = new RegExp(`<${tag}[\\s>]`, 'gi');
  const selfCloseRe = new RegExp(`<${tag}[^>]*/\\s*>`, 'gi');
  const closeRe = new RegExp(`</${tag}>`, 'gi');
  const opens = (html.match(openRe) ?? []).length;
  const selfClose = (html.match(selfCloseRe) ?? []).length;
  const closes = (html.match(closeRe) ?? []).length;
  return { open: opens - selfClose, close: closes };
}

// ── Setup ────────────────────────────────────────────────────────────────────

let outputDir: string;

beforeEach(() => {
  outputDir = fs.mkdtempSync(path.join(os.tmpdir(), 'fenix-report-'));
});

afterEach(() => {
  fs.rmSync(outputDir, { recursive: true, force: true });
});

// ── generateReport: basic structure ──────────────────────────────────────────

describe('generateReport — HTML structure', () => {
  it('produces report.html in outputDir', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    expect(fs.existsSync(path.join(outputDir, 'report.html'))).toBe(true);
  });

  it('HTML contains both entry names', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    expect(html).toContain('health');
    expect(html).toContain('auth-login-invalid');
  });

  it('HTML contains status codes for both entries', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    expect(html).toContain('200');
    expect(html).toContain('401');
  });

  it('HTML contains both paths', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    expect(html).toContain('/bff/health');
    expect(html).toContain('/bff/auth/login');
  });

  it('HTML has balanced <html> tags', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');
    const { open, close } = countTag(html, 'html');

    expect(open).toBe(1);
    expect(close).toBe(1);
  });

  it('HTML has balanced <body> tags', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');
    const { open, close } = countTag(html, 'body');

    expect(open).toBe(1);
    expect(close).toBe(1);
  });
});

// ── generateReport: sidebar groups ───────────────────────────────────────────

describe('generateReport — sidebar groups', () => {
  it('groups entries by group field in sidebar', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    // Both group names must appear (as section headers or labels)
    expect(html).toContain('root');
    expect(html).toContain('auth');
  });

  it('each entry appears as a clickable item in the sidebar', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    // Sidebar items use data-name attribute for JS selection
    expect(html).toContain('data-name="health"');
    expect(html).toContain('data-name="auth-login-invalid"');
  });
});

// ── generateReport: status badges ────────────────────────────────────────────

describe('generateReport — status badges', () => {
  it('2xx responses get a success badge class', () => {
    generateReport([ARTIFACT_HEALTH], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    expect(html).toContain('badge-success');
  });

  it('4xx responses get an error badge class', () => {
    generateReport([ARTIFACT_LOGIN_FAIL], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    expect(html).toContain('badge-error');
  });

  it('5xx responses get an error badge class', () => {
    const artifact = makeArtifact({
      name: 'server-error',
      group: 'test',
      response: { status: 500, headers: {}, body: { error: 'internal' } },
    });

    generateReport([artifact], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    expect(html).toContain('badge-error');
  });
});

// ── generateReport: XSS safety ───────────────────────────────────────────────

describe('generateReport — XSS safety', () => {
  it('escapes angle brackets in response body strings', () => {
    const artifact = makeArtifact({
      name: 'xss-test',
      group: 'security',
      response: {
        status: 200,
        headers: {},
        body: { message: '<script>alert("xss")</script>' },
      },
    });

    generateReport([artifact], outputDir);

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');

    // Raw script tag must NOT appear unescaped
    expect(html).not.toContain('<script>alert("xss")</script>');
    // Escaped form must appear instead
    expect(html).toContain('&lt;script&gt;');
  });
});

// ── generateReport: index.md ──────────────────────────────────────────────────

describe('generateReport — index.md', () => {
  it('produces index.md in outputDir', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    expect(fs.existsSync(path.join(outputDir, 'index.md'))).toBe(true);
  });

  it('index.md contains one row per entry', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const md = fs.readFileSync(path.join(outputDir, 'index.md'), 'utf8');

    expect(md).toContain('health');
    expect(md).toContain('auth-login-invalid');
  });

  it('index.md marks expected statuses as pass, including expected 4xx responses', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const md = fs.readFileSync(path.join(outputDir, 'index.md'), 'utf8');

    expect(md).toContain('| ✅ | GET | /bff/health | 200 | 200 | 15ms | [health](raw/root/health.json) |');
    expect(md).toContain('| ✅ | POST | /bff/auth/login | 401 | 401 | 88ms | [auth-login-invalid](raw/auth/auth-login-invalid.json) |');
  });

  it('index.md marks status mismatches as fail', () => {
    const artifact = makeArtifact({
      name: 'unexpected-error',
      group: 'test',
      expectedStatus: 200,
      response: { status: 500, headers: {}, body: { error: 'internal' } },
    });

    generateReport([artifact], outputDir);

    const md = fs.readFileSync(path.join(outputDir, 'index.md'), 'utf8');

    expect(md).toContain('| ❌ | GET | /bff/health | 200 | 500 | 42ms | [unexpected-error](raw/test/unexpected-error.json) |');
  });

  it('index.md contains latency values', () => {
    generateReport([ARTIFACT_HEALTH, ARTIFACT_LOGIN_FAIL], outputDir);

    const md = fs.readFileSync(path.join(outputDir, 'index.md'), 'utf8');

    expect(md).toContain('15');
    expect(md).toContain('88');
  });
});

// ── generateReport: empty input ───────────────────────────────────────────────

describe('generateReport — edge cases', () => {
  it('handles empty artifact list without throwing', () => {
    expect(() => generateReport([], outputDir)).not.toThrow();

    expect(fs.existsSync(path.join(outputDir, 'report.html'))).toBe(true);
    expect(fs.existsSync(path.join(outputDir, 'index.md'))).toBe(true);
  });

  it('handles SSE artifact (body is array of SSEEvent) without throwing', () => {
    const sseArtifact = makeArtifact({
      name: 'copilot-stream',
      group: 'copilot',
      method: 'GET',
      path: '/bff/copilot/events',
      response: {
        status: 200,
        headers: {},
        body: [{ data: '{"type":"chunk"}', index: 0 }, { data: '{"type":"done"}', index: 1 }],
      },
    });

    expect(() => generateReport([sseArtifact], outputDir)).not.toThrow();

    const html = fs.readFileSync(path.join(outputDir, 'report.html'), 'utf8');
    expect(html).toContain('copilot-stream');
  });
});
