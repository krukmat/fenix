// admin-screenshots Task 4: HTML report + Markdown index generator
import * as fs from 'fs';
import * as path from 'path';
import type { ShooterResult } from './shooter';

export function generateReport(results: ShooterResult[], outputDir: string): void {
  fs.mkdirSync(outputDir, { recursive: true });
  writeHtml(results, outputDir);
  writeMarkdown(results, outputDir);
}

function writeHtml(results: ShooterResult[], outputDir: string): void {
  const cards = results
    .map((r) => {
      const imgSrc = `./${r.name}.png`;
      const status = r.ok ? '✅' : '❌';
      const errorRow = r.error
        ? `<p class="error">${esc(r.error)}</p>`
        : '';
      return `
    <div class="card ${r.ok ? '' : 'card-fail'}">
      ${r.ok ? `<a href="${imgSrc}" target="_blank"><img src="${imgSrc}" alt="${esc(r.name)}" loading="lazy"></a>` : '<div class="no-img">no capture</div>'}
      <div class="meta">
        <span class="status">${status}</span>
        <strong>${esc(r.name)}</strong>
        <code>${esc(r.route)}</code>
        <time>${esc(r.capturedAt)}</time>
        ${errorRow}
      </div>
    </div>`;
    })
    .join('\n');

  const passed = results.filter((r) => r.ok).length;
  const failed = results.filter((r) => !r.ok).length;
  const generatedAt = new Date().toISOString();

  const html = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>FenixCRM BFF Admin Screenshots</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 0; background: #0f1117; color: #e2e8f0; }
  header { padding: 24px 32px; border-bottom: 1px solid #1e2533; }
  h1 { margin: 0 0 6px; font-size: 20px; }
  .summary { font-size: 13px; color: #8892a4; }
  .summary strong { color: #e2e8f0; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 20px; padding: 24px 32px; }
  .card { background: #161b27; border: 1px solid #1e2533; border-radius: 8px; overflow: hidden; }
  .card-fail { border-color: #7f1d1d; }
  .card img { width: 100%; display: block; border-bottom: 1px solid #1e2533; }
  .no-img { height: 120px; display: flex; align-items: center; justify-content: center; color: #4b5563; font-size: 13px; border-bottom: 1px solid #1e2533; }
  .meta { padding: 12px 14px; display: flex; flex-direction: column; gap: 4px; font-size: 12px; }
  .status { font-size: 16px; }
  .meta strong { font-size: 13px; color: #e2e8f0; }
  .meta code { color: #7dd3fc; word-break: break-all; }
  .meta time { color: #6b7280; }
  .error { color: #fca5a5; margin: 4px 0 0; }
</style>
</head>
<body>
<header>
  <h1>FenixCRM BFF Admin Screenshots</h1>
  <p class="summary">
    Generated: <strong>${generatedAt}</strong> &nbsp;|&nbsp;
    Captured: <strong>${passed}</strong> &nbsp;|&nbsp;
    Failed: <strong>${failed}</strong>
  </p>
</header>
<div class="grid">
${cards}
</div>
</body>
</html>`;

  fs.writeFileSync(path.join(outputDir, 'report.html'), html, 'utf8');
}

function writeMarkdown(results: ShooterResult[], outputDir: string): void {
  const generatedAt = new Date().toISOString();
  const rows = results
    .map((r) => `| ${r.ok ? '✅' : '❌'} | \`${r.name}\` | \`${r.route}\` | ${r.capturedAt} |`)
    .join('\n');

  const md = `# BFF Admin Screenshots

Generated: ${generatedAt}

| Status | Name | Route | Captured At |
|--------|------|-------|-------------|
${rows}
`;

  fs.writeFileSync(path.join(outputDir, 'index.md'), md, 'utf8');
}

function esc(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
