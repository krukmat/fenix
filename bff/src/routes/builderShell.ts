// CLSF-62-65: Builder HTML shell template — extracted to meet max-lines + max-lines-per-function gates
import { BUILDER_SCRIPT, GENERATED_SOURCE_DIFF, GRAPH_AUTHORING_STYLES, GRAPH_CANVAS_PLACEHOLDER, renderGraphAuthoringControls } from './builderCanvas';

const HTMX_CDN = 'https://unpkg.com/htmx.org@2.0.4';

export interface VisualNode { id: string; kind: string; label: string; color?: string; position: { x: number; y: number } }
export interface VisualEdge { id?: string; from: string; to: string; connection_type?: string }
export interface VisualProjection { workflow_name?: string; nodes?: VisualNode[]; edges?: VisualEdge[]; conformance?: { profile?: string; details?: unknown[] } }
export interface BuilderViewModel {
  workflowId: string; workflowName: string; source: string; specSource: string;
  errorMessage?: string; graphErrorMessage?: string; visualProjection?: VisualProjection;
  projectionSource: 'fixture' | 'api'; standalone: boolean;
}

export function escHtml(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

const BUILDER_CSS = `
      :root { color-scheme: light; --bg: #f6f7f9; --panel: #ffffff; --text: #172033; --muted: #5c667a; --line: #d9dee8; --accent: #1868db; --accent-dark: #0f4fa8; }
      * { box-sizing: border-box; }
      body { margin: 0; min-height: 100vh; background: var(--bg); color: var(--text); font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
      header { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 14px 20px; border-bottom: 1px solid var(--line); background: var(--panel); }
      h1 { margin: 0; font-size: 20px; font-weight: 700; }
      main { display: grid; grid-template-columns: minmax(320px, 1fr) minmax(320px, 1fr); gap: 16px; padding: 16px; min-height: calc(100vh - 62px); }
      section { min-width: 0; background: var(--panel); border: 1px solid var(--line); border-radius: 8px; overflow: hidden; }
      .panel-header { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 14px; border-bottom: 1px solid var(--line); }
      .panel-title { margin: 0; font-size: 14px; font-weight: 700; }
      textarea { width: 100%; min-height: 0; padding: 14px; resize: vertical; border: 0; outline: 0; color: var(--text); font: 14px/1.55 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
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
      .graph-node { fill: #ffffff; stroke: #1868db; stroke-width: 2; rx: 8; } .graph-node.action { stroke: #2e7d32; } .graph-node.governance { stroke: #8a5a00; }
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
      .builder-context { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
      .context-chip { display: inline-flex; align-items: center; height: 26px; padding: 0 10px; border: 1px solid var(--line); border-radius: 999px; color: var(--muted); font: 600 12px/1 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; background: #fbfcfe; }
      @media (max-width: 820px) { header { align-items: stretch; flex-direction: column; } main { grid-template-columns: 1fr; } .inspector-grid { grid-template-columns: 1fr; } .auth-bar { min-width: 0; width: 100%; } }`;

const AUTH_SCRIPT = `<script>
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
          if (token) { event.detail.headers.Authorization = 'Bearer ' + token; }
        });
      }());
    </script>`;

interface ShellSections {
  workflowLabel: string; graphControls: string; projectionPayload: string; initialProjection: string;
  navHtml: string; guidanceHtml: string; editorTitle: string; previewActionsHtml: string;
  editorBannerHtml: string; editorSource: string; specSource: string; graphBannerHtml: string;
  projectionSource: string; workflowName: string; graphContentHtml: string; graphCaptionText: string; inspectorHtml: string;
}

function renderEditorSection(s: ShellSections): string {
  return `<section aria-labelledby="editor-title">
        <div class="panel-header">
          <h2 class="panel-title" id="editor-title">${s.editorTitle}</h2>
          <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap"><span class="preview-status" id="builder-preview-status">Preview idle</span>${s.previewActionsHtml}</div>
        </div>
        ${s.editorBannerHtml}
        <form id="builder-editor-form" class="editor-workspace" hx-post="/bff/builder/preview" hx-trigger="keyup changed delay:700ms from:#builder-editor, keyup changed delay:700ms from:#builder-spec-source" hx-target="#builder-preview-status" hx-swap="innerHTML">
          <textarea id="builder-editor" name="source" spellcheck="false" placeholder="WORKFLOW sales_followup&#10;ON deal.updated" aria-describedby="builder-diagnostics">${s.editorSource}</textarea>
          <textarea class="spec-source" id="builder-spec-source" name="spec_source" spellcheck="false" placeholder="CARTA sales_followup&#10;AGENT sales_assistant&#10;  PERMIT send_reply" aria-label="Carta spec source">${s.specSource}</textarea>
          <aside class="diagnostics" aria-labelledby="diagnostics-title">
            <h3 class="diagnostics-title" id="diagnostics-title">Validation diagnostics</h3>
            <ul class="diagnostics-list" id="builder-diagnostics" aria-live="polite"><li class="diagnostic-empty">No diagnostics have been run for this draft.</li></ul>
          </aside>
        </form>
      </section>`;
}

function renderGraphSection(s: ShellSections): string {
  const projAttr = s.projectionPayload ? ` data-projection-payload="${s.projectionPayload}"` : '';
  return `<section aria-labelledby="graph-title">
        <div class="panel-header">
          <h2 class="panel-title" id="graph-title">Graph</h2>
          <span class="preview-status" id="builder-bound-workflow">Bound workflow: ${escHtml(s.workflowLabel)}</span>
        </div>
        ${s.graphBannerHtml}${s.graphControls}
        <div class="graph-shell" id="builder-graph" data-projection-source="${s.projectionSource}" data-workflow-id="${escHtml(s.workflowLabel)}" data-workflow-name="${escHtml(s.workflowName)}"${projAttr}>
          ${GRAPH_CANVAS_PLACEHOLDER}${s.graphContentHtml}
          <p class="graph-caption">${escHtml(s.graphCaptionText)}</p>
          ${s.inspectorHtml}
        </div>
        ${s.initialProjection}${GENERATED_SOURCE_DIFF}
      </section>`;
}

export function renderBuilderShell(
  workflowLabel: string, graphControls: string, projectionPayload: string, initialProjection: string,
  navHtml: string, guidanceHtml: string, editorTitle: string, previewActionsHtml: string,
  editorBannerHtml: string, editorSource: string, specSource: string, graphBannerHtml: string,
  projectionSource: string, workflowName: string, graphContentHtml: string, graphCaptionText: string, inspectorHtml: string,
): string {
  const s: ShellSections = { workflowLabel, graphControls, projectionPayload, initialProjection, navHtml, guidanceHtml, editorTitle, previewActionsHtml, editorBannerHtml, editorSource, specSource, graphBannerHtml, projectionSource, workflowName, graphContentHtml, graphCaptionText, inspectorHtml };
  return `<!doctype html>
<html lang="en">
  <head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>FenixCRM Builder</title><script src="${HTMX_CDN}"></script><style>${BUILDER_CSS}</style></head>
  <body>
    <header>
      <div class="builder-context"><h1>FenixCRM Builder</h1><span class="context-chip" id="builder-workflow-id">workflowId: ${escHtml(workflowLabel)}</span></div>
      <form class="auth-bar" id="builder-auth-form"><input id="builder-token" type="password" autocomplete="off" placeholder="Bearer token" aria-label="Bearer token"><button type="submit">Use Token</button></form>
    </header>
    <main>
      <div style="grid-column:1 / -1">${navHtml}${guidanceHtml}</div>
      ${renderEditorSection(s)}
      ${renderGraphSection(s)}
    </main>
    ${AUTH_SCRIPT}${BUILDER_SCRIPT}
  </body>
</html>`;
}

export function renderGraphAuthoringControlsShell(workflowLabel: string): string {
  return renderGraphAuthoringControls(workflowLabel);
}
