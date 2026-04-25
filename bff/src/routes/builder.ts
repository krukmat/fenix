// CLSF-62-65: Web builder shell served by the BFF with HTMX loaded from CDN.
import { Router, Request, Response } from 'express';
import {
  BUILDER_SCRIPT,
  GENERATED_SOURCE_DIFF,
  GRAPH_AUTHORING_CONTROLS,
  GRAPH_AUTHORING_STYLES,
  GRAPH_CANVAS_PLACEHOLDER,
} from './builderCanvas';
import builderPreviewRouter from './builderPreview';
import builderVisualAuthoringRouter from './builderVisualAuthoring';

const HTMX_CDN = 'https://unpkg.com/htmx.org@2.0.4';

const builderHtml = `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>FenixCRM Builder</title>
    <script src="${HTMX_CDN}"></script>
    <style>
      :root {
        color-scheme: light;
        --bg: #f6f7f9;
        --panel: #ffffff;
        --text: #172033;
        --muted: #5c667a;
        --line: #d9dee8;
        --accent: #1868db;
        --accent-dark: #0f4fa8;
      }
      * { box-sizing: border-box; }
      body {
        margin: 0;
        min-height: 100vh;
        background: var(--bg);
        color: var(--text);
        font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      }
      header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 16px;
        padding: 14px 20px;
        border-bottom: 1px solid var(--line);
        background: var(--panel);
      }
      h1 {
        margin: 0;
        font-size: 20px;
        font-weight: 700;
      }
      main {
        display: grid;
        grid-template-columns: minmax(320px, 1fr) minmax(320px, 1fr);
        gap: 16px;
        padding: 16px;
        min-height: calc(100vh - 62px);
      }
      section {
        min-width: 0;
        background: var(--panel);
        border: 1px solid var(--line);
        border-radius: 8px;
        overflow: hidden;
      }
      .panel-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--line); }
      .panel-title { margin: 0; font-size: 14px; font-weight: 700; }
      textarea {
        width: 100%;
        min-height: 0;
        padding: 14px;
        resize: vertical;
        border: 0;
        outline: 0;
        color: var(--text);
        font: 14px/1.55 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
      }
      .editor-workspace { display: grid; grid-template-rows: minmax(260px, 1fr) auto; min-height: calc(100vh - 144px); }
      .diagnostics { border-top: 1px solid var(--line); padding: 12px 14px; background: #fbfcfe; }
      .diagnostics-title { margin: 0 0 8px; font-size: 13px; font-weight: 700; }
      .diagnostics-list { display: grid; gap: 8px; margin: 0; padding: 0; list-style: none; }
      .diagnostic-empty { color: var(--muted); font-size: 13px; }
      .preview-status { color: var(--muted); font-size: 12px; }
      .spec-source { border-top: 1px solid var(--line); min-height: 120px; background: #fbfcfe; }
      .graph-shell { min-height: calc(100vh - 144px); margin: 14px; overflow: auto; color: var(--muted); }
      .graph-canvas { width: 100%; min-width: 560px; min-height: 360px; border: 1px solid var(--line); border-radius: 8px; background: #fbfcfe; }
      .graph-edge { stroke: #8590a3; stroke-width: 2; marker-end: url(#arrowhead); }
      .graph-node { fill: #ffffff; stroke: #1868db; stroke-width: 2; rx: 8; }
      .graph-node.action { stroke: #2e7d32; }
      .graph-node.governance { stroke: #8a5a00; }
      .graph-label { fill: var(--text); font: 700 13px Inter, ui-sans-serif, system-ui, sans-serif; }
      .graph-meta { fill: var(--muted); font: 12px Inter, ui-sans-serif, system-ui, sans-serif; }
      .graph-caption { margin: 10px 0 0; color: var(--muted); font-size: 13px; }
      ${GRAPH_AUTHORING_STYLES}
      .inspector { margin-top: 12px; border: 1px solid var(--line); border-radius: 8px; background: #ffffff; }
      .inspector-header { padding: 10px 12px; border-bottom: 1px solid var(--line); }
      .inspector-title { margin: 0; font-size: 13px; font-weight: 700; color: var(--text); }
      .inspector-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 1px; background: var(--line); }
      .inspector-block { min-width: 0; padding: 12px; background: #ffffff; }
      .inspector-label { display: block; margin-bottom: 6px; color: var(--muted); font-size: 12px; font-weight: 700; }
      .inspector-value { margin: 0; color: var(--text); font-size: 13px; }
      .auth-bar { display: flex; align-items: center; gap: 8px; min-width: min(520px, 52vw); }
      .auth-bar input { flex: 1; min-width: 120px; height: 36px; border: 1px solid var(--line); border-radius: 6px; padding: 0 10px; color: var(--text); }
      .auth-bar button { height: 36px; border: 0; border-radius: 6px; padding: 0 12px; background: var(--accent); color: #ffffff; font-weight: 700; cursor: pointer; }
      .auth-bar button:hover { background: var(--accent-dark); }
      @media (max-width: 820px) {
        header { align-items: stretch; flex-direction: column; }
        main { grid-template-columns: 1fr; }
        .inspector-grid { grid-template-columns: 1fr; }
        .auth-bar { min-width: 0; width: 100%; }
      }
    </style>
  </head>
  <body>
    <header>
      <h1>FenixCRM Builder</h1>
      <form class="auth-bar" id="builder-auth-form">
        <input id="builder-token" type="password" autocomplete="off" placeholder="Bearer token" aria-label="Bearer token">
        <button type="submit">Use Token</button>
      </form>
    </header>
    <main>
      <section aria-labelledby="editor-title">
        <div class="panel-header">
          <h2 class="panel-title" id="editor-title">Editor</h2>
          <span class="preview-status" id="builder-preview-status">Preview idle</span>
        </div>
        <form class="editor-workspace" hx-post="/bff/builder/preview" hx-trigger="keyup changed delay:700ms from:#builder-editor, keyup changed delay:700ms from:#builder-spec-source" hx-target="#builder-preview-status" hx-swap="innerHTML">
          <textarea id="builder-editor" name="source" spellcheck="false" placeholder="WORKFLOW sales_followup&#10;ON deal.updated" aria-describedby="builder-diagnostics"></textarea>
          <textarea class="spec-source" id="builder-spec-source" name="spec_source" spellcheck="false" placeholder="CARTA sales_followup&#10;AGENT sales_assistant&#10;  PERMIT send_reply" aria-label="Carta spec source"></textarea>
          <aside class="diagnostics" aria-labelledby="diagnostics-title">
            <h3 class="diagnostics-title" id="diagnostics-title">Validation diagnostics</h3>
            <ul class="diagnostics-list" id="builder-diagnostics" aria-live="polite">
              <li class="diagnostic-empty">No diagnostics have been run for this draft.</li>
            </ul>
          </aside>
        </form>
      </section>
      <section aria-labelledby="graph-title">
        <div class="panel-header">
          <h2 class="panel-title" id="graph-title">Graph</h2>
        </div>
        ${GRAPH_AUTHORING_CONTROLS}
        <div class="graph-shell" id="builder-graph" data-projection-source="fixture">
          ${GRAPH_CANVAS_PLACEHOLDER}
          <svg class="graph-canvas" viewBox="0 0 640 380" role="img" aria-labelledby="builder-graph-title builder-graph-desc">
            <title id="builder-graph-title">Read-only workflow graph preview</title>
            <desc id="builder-graph-desc">Fixture graph with workflow, trigger, action, and governance nodes connected by read-only edges.</desc>
            <defs><marker id="arrowhead" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 Z" fill="#8590a3"></path></marker></defs>
            <line class="graph-edge" x1="190" y1="80" x2="250" y2="80"></line><line class="graph-edge" x1="410" y1="80" x2="470" y2="80"></line><line class="graph-edge" x1="320" y1="130" x2="320" y2="215"></line>
            <rect class="graph-node" x="40" y="40" width="150" height="80"></rect><text class="graph-label" x="62" y="75">Workflow</text><text class="graph-meta" x="62" y="98">sales_followup</text>
            <rect class="graph-node" x="250" y="40" width="160" height="80"></rect><text class="graph-label" x="272" y="75">Trigger</text><text class="graph-meta" x="272" y="98">deal.updated</text>
            <rect class="graph-node action" x="470" y="40" width="140" height="80"></rect><text class="graph-label" x="492" y="75">Action</text><text class="graph-meta" x="492" y="98">notify owner</text>
            <rect class="graph-node governance" x="242" y="215" width="176" height="82"></rect><text class="graph-label" x="264" y="250">Governance</text><text class="graph-meta" x="264" y="273">permit + grounds</text>
          </svg>
          <p class="graph-caption">Read-only fixture projection. Live backend refresh is reserved for CLSF-66.</p>
          <aside class="inspector" id="builder-inspector" aria-labelledby="inspector-title">
            <div class="inspector-header"><h3 class="inspector-title" id="inspector-title">Inspector</h3></div>
            <div class="inspector-grid">
              <div class="inspector-block"><span class="inspector-label">Selected node</span><p class="inspector-value">Workflow / sales_followup</p></div>
              <div class="inspector-block"><span class="inspector-label">Conformance</span><p class="inspector-value">safe fixture</p></div>
              <div class="inspector-block"><span class="inspector-label">Diagnostics</span><p class="inspector-value">No graph diagnostics for fixture projection.</p></div>
            </div>
          </aside>
        </div>
        ${GENERATED_SOURCE_DIFF}
      </section>
    </main>
    <script>
      (function () {
        var tokenInput = document.getElementById('builder-token');
        var authForm = document.getElementById('builder-auth-form');
        tokenInput.value = localStorage.getItem('fenix.builder.bearerToken') || '';
        authForm.addEventListener('submit', function (event) {
          event.preventDefault();
          localStorage.setItem('fenix.builder.bearerToken', tokenInput.value.trim());
        });
        document.body.addEventListener('htmx:configRequest', function (event) {
          var token = localStorage.getItem('fenix.builder.bearerToken');
          if (token) {
            event.detail.headers.Authorization = 'Bearer ' + token;
          }
        });
      }());
    </script>
    ${BUILDER_SCRIPT}
  </body>
</html>`;

const router = Router();

router.use('/preview', builderPreviewRouter);
router.use('/visual-authoring', builderVisualAuthoringRouter);

router.get('/', (_req: Request, res: Response): void => {
  res.type('html').status(200).send(builderHtml);
});

export default router;
