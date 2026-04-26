// bff-http-snapshots T7: HTML report + index.md generator
import fs from 'fs';
import path from 'path';
import type { SnapshotArtifact } from './types';

export type ArtifactWithGroup = SnapshotArtifact & { group: string };

// ── Helpers ──────────────────────────────────────────────────────────────────

function escapeHtml(raw: string): string {
  return raw
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function badgeClass(status: number): string {
  if (status >= 200 && status < 300) return 'badge-success';
  if (status >= 300 && status < 400) return 'badge-redirect';
  return 'badge-error';
}

function statusIcon(status: number, expectedStatus?: number): string {
  return expectedStatus === undefined || status === expectedStatus ? '✅' : '❌';
}

/**
 * Colorizes an already-HTML-escaped JSON string with <span> elements.
 * Must be called AFTER escapeHtml to avoid double-escaping entity references.
 */
function colorizeJson(escaped: string): string {
  return escaped
    // string values (after html escaping, quotes are &quot;)
    .replace(/(&quot;)((?:[^&]|&(?!quot;))*?)(&quot;)(\s*:)/g,
      '<span class="jk">$1$2$3</span>$4')
    .replace(/(&quot;)((?:[^&]|&(?!quot;))*?)(&quot;)/g,
      '<span class="js">$1$2$3</span>')
    // numbers
    .replace(/:\s*(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g,
      ': <span class="jn">$1</span>')
    // booleans and null
    .replace(/:\s*(true|false|null)/g,
      ': <span class="jb">$1</span>');
}

function renderBody(body: unknown): string {
  const raw = JSON.stringify(body, null, 2);
  const escaped = escapeHtml(raw);
  return colorizeJson(escaped);
}

// ── Sidebar + detail HTML builders ──────────────────────────────────────────

function buildSidebarGroups(artifacts: ArtifactWithGroup[]): string {
  const groups = new Map<string, ArtifactWithGroup[]>();
  for (const a of artifacts) {
    const list = groups.get(a.group) ?? [];
    list.push(a);
    groups.set(a.group, list);
  }

  return [...groups.entries()].map(([group, items]) => `
    <div class="group">
      <div class="group-label">${escapeHtml(group)}</div>
      ${items.map((a) => `
        <div class="sidebar-item" data-name="${escapeHtml(a.name)}" onclick="showDetail('${escapeHtml(a.name)}')">
          <span class="badge ${badgeClass(a.response.status)}">${a.response.status}</span>
          <span class="item-name">${escapeHtml(a.name)}</span>
        </div>`).join('')}
    </div>`).join('');
}

function buildDetailPanels(artifacts: ArtifactWithGroup[]): string {
  return artifacts.map((a, i) => `
    <div class="detail-panel" id="panel-${escapeHtml(a.name)}" style="display:${i === 0 ? 'block' : 'none'}">
      <h2>
        <span class="badge ${badgeClass(a.response.status)}">${a.response.status}</span>
        <span class="method">${escapeHtml(a.method)}</span>
        <span class="path">${escapeHtml(a.path)}</span>
      </h2>
      <p class="meta">Latency: <strong>${a.latencyMs}ms</strong> &nbsp;·&nbsp; Captured: ${escapeHtml(a.capturedAt)}</p>

      <h3>Request headers</h3>
      <pre class="json-block">${renderBody(a.request.headers)}</pre>

      ${a.request.body !== undefined ? `
      <h3>Request body</h3>
      <pre class="json-block">${renderBody(a.request.body)}</pre>` : ''}

      <h3>Response headers</h3>
      <pre class="json-block">${renderBody(a.response.headers)}</pre>

      <h3>Response body</h3>
      <pre class="json-block">${renderBody(a.response.body)}</pre>
    </div>`).join('');
}

// ── CSS ──────────────────────────────────────────────────────────────────────

const CSS = `
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
         display: flex; height: 100vh; overflow: hidden; background: #0d1117; color: #c9d1d9; }
  #sidebar { width: 260px; min-width: 260px; overflow-y: auto; border-right: 1px solid #21262d;
             padding: 12px 0; background: #161b22; }
  #sidebar h1 { font-size: 13px; font-weight: 700; color: #58a6ff; padding: 8px 16px 16px;
                letter-spacing: .04em; text-transform: uppercase; }
  .group-label { font-size: 11px; font-weight: 600; color: #8b949e; padding: 10px 16px 4px;
                 text-transform: uppercase; letter-spacing: .06em; }
  .sidebar-item { display: flex; align-items: center; gap: 8px; padding: 5px 16px;
                  cursor: pointer; border-radius: 4px; margin: 1px 6px; }
  .sidebar-item:hover { background: #21262d; }
  .sidebar-item.active { background: #1f2937; }
  .item-name { font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  #main { flex: 1; overflow-y: auto; padding: 24px 32px; }
  .detail-panel h2 { font-size: 16px; display: flex; align-items: center; gap: 10px;
                     margin-bottom: 6px; flex-wrap: wrap; }
  .method { font-weight: 700; color: #79c0ff; }
  .path { font-family: monospace; font-size: 14px; color: #e6edf3; }
  .meta { font-size: 12px; color: #8b949e; margin-bottom: 20px; }
  h3 { font-size: 12px; font-weight: 600; color: #8b949e; text-transform: uppercase;
       letter-spacing: .05em; margin: 18px 0 6px; }
  pre.json-block { background: #161b22; border: 1px solid #21262d; border-radius: 6px;
                   padding: 14px; font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px;
                   line-height: 1.6; overflow-x: auto; white-space: pre; }
  .badge { display: inline-block; padding: 2px 7px; border-radius: 4px;
           font-size: 11px; font-weight: 700; font-family: monospace; }
  .badge-success { background: #1a4731; color: #56d364; }
  .badge-redirect { background: #2d2a1a; color: #e3b341; }
  .badge-error { background: #4a1d1d; color: #f85149; }
  /* JSON syntax highlight */
  .jk { color: #79c0ff; }
  .js { color: #a5d6ff; }
  .jn { color: #f2cc60; }
  .jb { color: #ff7b72; }
`;

// ── JS ───────────────────────────────────────────────────────────────────────

const JS = `
  function showDetail(name) {
    document.querySelectorAll('.detail-panel').forEach(function(p) {
      p.style.display = 'none';
    });
    document.querySelectorAll('.sidebar-item').forEach(function(i) {
      i.classList.remove('active');
    });
    var panel = document.getElementById('panel-' + name);
    if (panel) panel.style.display = 'block';
    var item = document.querySelector('[data-name="' + name + '"]');
    if (item) item.classList.add('active');
  }
  // Activate first item on load
  document.addEventListener('DOMContentLoaded', function() {
    var first = document.querySelector('.sidebar-item');
    if (first) first.classList.add('active');
  });
`;

// ── Public API ────────────────────────────────────────────────────────────────

/**
 * Reads all snapshot artifacts and generates report.html + index.md
 * in `outputDir`.
 *
 * Artifacts must include a `group` field (added by the runner when writing
 * to `raw/<group>/<name>.json`).
 */
export function generateReport(artifacts: ArtifactWithGroup[], outputDir: string): void {
  fs.mkdirSync(outputDir, { recursive: true });
  writeHtml(artifacts, outputDir);
  writeIndexMd(artifacts, outputDir);
}

function writeHtml(artifacts: ArtifactWithGroup[], outputDir: string): void {
  const sidebar = buildSidebarGroups(artifacts);
  const details = buildDetailPanels(artifacts);

  const html = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>BFF HTTP Snapshots</title>
  <style>${CSS}</style>
</head>
<body>
  <nav id="sidebar">
    <h1>HTTP Snapshots</h1>
    ${sidebar}
  </nav>
  <main id="main">
    ${artifacts.length === 0
      ? '<p style="color:#8b949e;padding:40px 0">No snapshots captured yet. Run <code>npm run http-snapshots</code> first.</p>'
      : details}
  </main>
  <script>${JS}</script>
</body>
</html>`;

  fs.writeFileSync(path.join(outputDir, 'report.html'), html, 'utf8');
}

function writeIndexMd(artifacts: ArtifactWithGroup[], outputDir: string): void {
  const rows = artifacts.map((a) => {
    const expected = a.expectedStatus ?? '—';
    const icon = statusIcon(a.response.status, a.expectedStatus);
    const link = `[${a.name}](raw/${a.group}/${a.name}.json)`;
    return `| ${icon} | ${a.method} | ${escapeHtml(a.path)} | ${expected} | ${a.response.status} | ${a.latencyMs}ms | ${link} |`;
  });

  const md = [
    '# BFF HTTP Snapshots — Index',
    '',
    `Generated: ${new Date().toISOString()}`,
    '',
    '| Status | Method | Path | Expected | HTTP | Latency | Artifact |',
    '|--------|--------|------|----------|------|---------|----------|',
    ...rows,
  ].join('\n');

  fs.writeFileSync(path.join(outputDir, 'index.md'), md, 'utf8');
}

// ── Loader: reads artifacts from disk ────────────────────────────────────────

/**
 * Reads all `*.json` and `*.sse.json` files under `rawDir` and returns
 * them as `ArtifactWithGroup[]`, inferring `group` from the subdirectory name.
 */
export function loadArtifacts(rawDir: string): ArtifactWithGroup[] {
  if (!fs.existsSync(rawDir)) return [];

  const artifacts: ArtifactWithGroup[] = [];

  for (const group of fs.readdirSync(rawDir)) {
    const groupDir = path.join(rawDir, group);
    if (!fs.statSync(groupDir).isDirectory()) continue;

    for (const file of fs.readdirSync(groupDir)) {
      if (!file.endsWith('.json')) continue;
      try {
        const raw = fs.readFileSync(path.join(groupDir, file), 'utf8');
        const artifact = JSON.parse(raw) as SnapshotArtifact;
        artifacts.push({ ...artifact, group });
      } catch {
        // skip malformed artifact files
      }
    }
  }

  return artifacts;
}
