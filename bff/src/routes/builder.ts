// CLSF-62-65: Web builder shell served by the BFF with HTMX loaded from CDN.
import { Router, Request, Response } from 'express';
import { createGoClient } from '../services/goClient';
import {
  BUILDER_SCRIPT,
  GENERATED_SOURCE_DIFF,
  GRAPH_AUTHORING_STYLES,
  GRAPH_CANVAS_PLACEHOLDER,
  renderGraphAuthoringControls,
} from './builderCanvas';
import builderPreviewRouter from './builderPreview';
import builderSaveRouter from './builderSave';
import builderVisualAuthoringRouter from './builderVisualAuthoring';
import { upstreamMessage, upstreamStatus } from './adminAuth';

const HTMX_CDN = 'https://unpkg.com/htmx.org@2.0.4';

const DEFAULT_WORKFLOW_ID = 'sales_followup';

interface BuilderWorkflowRecord {
  id: string;
  name: string;
  dsl_source: string;
  spec_source?: string | null;
}

interface VisualNode {
  id: string;
  kind: string;
  label: string;
  color?: string;
  position: { x: number; y: number };
}

interface VisualEdge {
  id?: string;
  from: string;
  to: string;
  connection_type?: string;
}

interface VisualProjection {
  workflow_name?: string;
  nodes?: VisualNode[];
  edges?: VisualEdge[];
  conformance?: { profile?: string; details?: unknown[] };
}

interface BuilderViewModel {
  workflowId: string;
  workflowName: string;
  source: string;
  specSource: string;
  errorMessage?: string;
  graphErrorMessage?: string;
  visualProjection?: VisualProjection;
  projectionSource: 'fixture' | 'api';
  standalone: boolean;
}

function escHtml(value: string): string {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function editorBanner(message?: string): string {
  if (!message) return '';
  return `<div style="margin:14px 14px 0;padding:12px 14px;border:1px solid #fca5a5;border-radius:8px;background:#fef2f2;color:#991b1b;font-size:13px">${escHtml(message)}</div>`;
}

function graphBanner(message?: string): string {
  if (!message) return '';
  return `<div style="margin:14px 14px 0;padding:12px 14px;border:1px solid #fcd34d;border-radius:8px;background:#fffbeb;color:#92400e;font-size:13px">${escHtml(message)}</div>`;
}

function graphNodeClass(kind: string): string {
  if (kind === 'action') return 'graph-node action';
  if (['grounds', 'permit', 'delegate', 'invariant', 'budget'].includes(kind)) return 'graph-node governance';
  return 'graph-node';
}

function renderEdge(edge: VisualEdge, index: Map<string, VisualNode>): string {
  const from = index.get(edge.from);
  const to = index.get(edge.to);
  if (!from || !to) return '';
  return `<line class="graph-edge" x1="${from.position.x + 125}" y1="${from.position.y + 36}" x2="${to.position.x + 45}" y2="${to.position.y + 36}"></line>`;
}

function renderNode(node: VisualNode): string {
  return `<rect class="${graphNodeClass(node.kind)}" x="${node.position.x}" y="${node.position.y}" width="170" height="72"></rect><text class="graph-label" x="${node.position.x + 20}" y="${node.position.y + 32}">${escHtml(node.kind)}</text><text class="graph-meta" x="${node.position.x + 20}" y="${node.position.y + 54}">${escHtml(node.label)}</text>`;
}

function renderGraphSvg(nodes: VisualNode[], edges: VisualEdge[]): string {
  const width = Math.max(...nodes.map((node) => node.position.x)) + 230;
  const height = Math.max(...nodes.map((node) => node.position.y)) + 150;
  const index = new Map(nodes.map((node) => [node.id, node]));
  return `<svg class="graph-canvas" viewBox="0 0 ${width} ${height}" role="img" aria-label="Live workflow graph"><defs><marker id="arrowhead" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 Z" fill="#8590a3"></path></marker></defs>${edges.map((edge) => renderEdge(edge, index)).join('')}${nodes.map(renderNode).join('')}</svg>`;
}

function renderEmptyGraph(): string {
  return '<div class="graph-canvas" role="img" aria-label="Empty workflow graph">No graph nodes returned for current draft.</div>';
}

function renderInspector(viewModel: BuilderViewModel): string {
  const firstNode = viewModel.visualProjection?.nodes?.[0];
  const selected = firstNode
    ? `${firstNode.kind} / ${firstNode.label}`
    : (viewModel.projectionSource === 'fixture' ? 'Workflow / sales_followup' : 'No node selected');
  const profile = viewModel.visualProjection?.conformance?.profile ?? (viewModel.projectionSource === 'fixture' ? 'safe fixture' : 'unknown');
  const diagnostics = viewModel.projectionSource === 'fixture'
    ? 'No graph diagnostics for fixture projection.'
    : 'Projection loaded from workflow context.';
  return `<aside class="inspector" id="builder-inspector" aria-labelledby="inspector-title">
            <div class="inspector-header"><h3 class="inspector-title" id="inspector-title">Inspector</h3></div>
            <div class="inspector-grid">
              <div class="inspector-block"><span class="inspector-label">Selected node</span><p class="inspector-value">${escHtml(selected)}</p></div>
              <div class="inspector-block"><span class="inspector-label">Conformance</span><p class="inspector-value">${escHtml(profile)}</p></div>
              <div class="inspector-block"><span class="inspector-label">Diagnostics</span><p class="inspector-value">${escHtml(diagnostics)}</p></div>
            </div>
          </aside>`;
}

function renderGraphContent(viewModel: BuilderViewModel): string {
  if (viewModel.projectionSource === 'api') {
    const nodes = viewModel.visualProjection?.nodes ?? [];
    const edges = viewModel.visualProjection?.edges ?? [];
    return nodes.length === 0 ? renderEmptyGraph() : renderGraphSvg(nodes, edges);
  }
  return `<svg class="graph-canvas" viewBox="0 0 640 380" role="img" aria-labelledby="builder-graph-title builder-graph-desc">
            <title id="builder-graph-title">Read-only workflow graph preview</title>
            <desc id="builder-graph-desc">Fixture graph with workflow, trigger, action, and governance nodes connected by read-only edges.</desc>
            <defs><marker id="arrowhead" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 Z" fill="#8590a3"></path></marker></defs>
            <line class="graph-edge" x1="190" y1="80" x2="250" y2="80"></line><line class="graph-edge" x1="410" y1="80" x2="470" y2="80"></line><line class="graph-edge" x1="320" y1="130" x2="320" y2="215"></line>
            <rect class="graph-node" x="40" y="40" width="150" height="80"></rect><text class="graph-label" x="62" y="75">Workflow</text><text class="graph-meta" x="62" y="98">sales_followup</text>
            <rect class="graph-node" x="250" y="40" width="160" height="80"></rect><text class="graph-label" x="272" y="75">Trigger</text><text class="graph-meta" x="272" y="98">deal.updated</text>
            <rect class="graph-node action" x="470" y="40" width="140" height="80"></rect><text class="graph-label" x="492" y="75">Action</text><text class="graph-meta" x="492" y="98">notify owner</text>
            <rect class="graph-node governance" x="242" y="215" width="176" height="82"></rect><text class="graph-label" x="264" y="250">Governance</text><text class="graph-meta" x="264" y="273">permit + grounds</text>
          </svg>`;
}

function graphCaption(viewModel: BuilderViewModel): string {
  if (viewModel.projectionSource === 'api') {
    return 'Live workflow projection loaded for the bound workflow.';
  }
  return 'Read-only fixture projection. Live backend refresh is reserved for CLSF-66.';
}

function renderShellGuidance(viewModel: BuilderViewModel): string {
  if (viewModel.standalone) {
    return `<div style="margin-bottom:16px;padding:12px 14px;border:1px solid var(--line);border-radius:8px;background:#fbfcfe;color:var(--muted);font-size:13px">
              Builder opened without a bound admin workflow. Use this mode for previewing the shell, or return to admin workflows and open a real draft.
            </div>`;
  }
  if (viewModel.errorMessage) {
    return `<div style="margin-bottom:16px;padding:12px 14px;border:1px solid var(--line);border-radius:8px;background:#fbfcfe;color:var(--muted);font-size:13px">
              This builder session is still bound to workflow <strong style="color:var(--text)">${escHtml(viewModel.workflowId)}</strong>. Fix the load issue or return to the admin detail to retry from the workflow record.
            </div>`;
  }
  return `<div style="margin-bottom:16px;padding:12px 14px;border:1px solid var(--line);border-radius:8px;background:#fbfcfe;color:var(--muted);font-size:13px">
            Editing workflow <strong style="color:var(--text)">${escHtml(viewModel.workflowName)}</strong> in the admin authoring flow. Save text or graph changes here, then return to workflow detail to review status and activate when ready.
          </div>`;
}

function renderBuilderNav(viewModel: BuilderViewModel): string {
  const detailLink = viewModel.standalone
    ? ''
    : `<a href="/bff/admin/workflows/${encodeURIComponent(viewModel.workflowId)}" style="display:inline-block;height:34px;line-height:34px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--text);text-decoration:none">Back to workflow detail</a>`;
  return `<div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-bottom:16px">
            <a href="/bff/admin/workflows" style="display:inline-block;height:34px;line-height:34px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--muted);text-decoration:none">Back to workflows</a>
            ${detailLink}
          </div>`;
}

function renderEditorActions(viewModel: BuilderViewModel): string {
  if (viewModel.standalone) {
    return `<span class="preview-status" id="builder-save-status">Text save unavailable in standalone mode</span>`;
  }
  return `<div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap">
            <button type="button"
              hx-post="/bff/builder/save/${encodeURIComponent(viewModel.workflowId)}"
              hx-include="#builder-editor-form"
              hx-target="#builder-save-status"
              hx-swap="outerHTML"
              style="height:32px;padding:0 12px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Save draft</button>
            <span class="preview-status" id="builder-save-status">Unsaved text changes</span>
          </div>`;
}

function builderHtml(viewModel: BuilderViewModel): string {
  const workflowLabel = viewModel.workflowId || DEFAULT_WORKFLOW_ID;
  const graphControls = renderGraphAuthoringControls(workflowLabel);
  const projectionPayload = viewModel.visualProjection ? escHtml(JSON.stringify(viewModel.visualProjection)) : '';
  const initialProjection = viewModel.visualProjection
    ? `<script id="builder-initial-projection" type="application/json">${projectionPayload}</script>`
    : '';
  return `<!doctype html>
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
      .builder-context { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
      .context-chip { display: inline-flex; align-items: center; height: 26px; padding: 0 10px; border: 1px solid var(--line); border-radius: 999px; color: var(--muted); font: 600 12px/1 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; background: #fbfcfe; }
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
      <div class="builder-context">
        <h1>FenixCRM Builder</h1>
        <span class="context-chip" id="builder-workflow-id">workflowId: ${escHtml(workflowLabel)}</span>
      </div>
      <form class="auth-bar" id="builder-auth-form">
        <input id="builder-token" type="password" autocomplete="off" placeholder="Bearer token" aria-label="Bearer token">
        <button type="submit">Use Token</button>
      </form>
    </header>
    <main>
      <div style="grid-column:1 / -1">
        ${renderBuilderNav(viewModel)}
        ${renderShellGuidance(viewModel)}
      </div>
      <section aria-labelledby="editor-title">
        <div class="panel-header">
          <h2 class="panel-title" id="editor-title">Editor: ${escHtml(viewModel.workflowName)}</h2>
          <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap">
            <span class="preview-status" id="builder-preview-status">Preview idle</span>
            ${renderEditorActions(viewModel)}
          </div>
        </div>
        ${editorBanner(viewModel.errorMessage)}
        <form id="builder-editor-form" class="editor-workspace" hx-post="/bff/builder/preview" hx-trigger="keyup changed delay:700ms from:#builder-editor, keyup changed delay:700ms from:#builder-spec-source" hx-target="#builder-preview-status" hx-swap="innerHTML">
          <textarea id="builder-editor" name="source" spellcheck="false" placeholder="WORKFLOW sales_followup&#10;ON deal.updated" aria-describedby="builder-diagnostics">${escHtml(viewModel.source)}</textarea>
          <textarea class="spec-source" id="builder-spec-source" name="spec_source" spellcheck="false" placeholder="CARTA sales_followup&#10;AGENT sales_assistant&#10;  PERMIT send_reply" aria-label="Carta spec source">${escHtml(viewModel.specSource)}</textarea>
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
          <span class="preview-status" id="builder-bound-workflow">Bound workflow: ${escHtml(workflowLabel)}</span>
        </div>
        ${graphBanner(viewModel.graphErrorMessage)}
        ${graphControls}
        <div class="graph-shell" id="builder-graph" data-projection-source="${viewModel.projectionSource}" data-workflow-id="${escHtml(workflowLabel)}" data-workflow-name="${escHtml(viewModel.workflowName)}"${projectionPayload ? ` data-projection-payload="${projectionPayload}"` : ''}>
          ${GRAPH_CANVAS_PLACEHOLDER}
          ${renderGraphContent(viewModel)}
          <p class="graph-caption">${escHtml(graphCaption(viewModel))}</p>
          ${renderInspector(viewModel)}
        </div>
        ${initialProjection}
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
}

function baseViewModel(workflowId: string): BuilderViewModel {
  return {
    workflowId,
    workflowName: workflowId,
    source: '',
    specSource: '',
    projectionSource: 'fixture',
    standalone: workflowId === DEFAULT_WORKFLOW_ID,
  };
}

function fromWorkflowRecord(workflow: BuilderWorkflowRecord): BuilderViewModel {
  return {
    workflowId: workflow.id,
    workflowName: workflow.name,
    source: workflow.dsl_source,
    specSource: workflow.spec_source ?? '',
    projectionSource: 'fixture',
    standalone: false,
  };
}

function loadErrorMessage(workflowId: string, err: unknown): string {
  const status = upstreamStatus(err);
  if (status === 401) return `Workflow ${workflowId} could not be loaded because the bearer token is unauthorized. Update the token and retry.`;
  if (status === 404) return `Workflow ${workflowId} was not found. Check the admin link or create the draft again.`;
  return `Workflow ${workflowId} could not be loaded: ${upstreamMessage(err)}`;
}

async function fetchVisualProjection(
  client: ReturnType<typeof createGoClient>,
  workflowId: string,
): Promise<{ projection?: VisualProjection; errorMessage?: string }> {
  try {
    const { data: resp } = await client.get<{ data: { visual_graph?: VisualProjection } }>(
      `/api/v1/workflows/${workflowId}/graph`,
      { params: { format: 'visual' } },
    );
    return { projection: resp.data.visual_graph ?? { workflow_name: workflowId, nodes: [], edges: [] } };
  } catch (err: unknown) {
    return {
      projection: { workflow_name: workflowId, nodes: [], edges: [] },
      errorMessage: `Visual projection for ${workflowId} is unavailable: ${upstreamMessage(err)}`,
    };
  }
}

const router = Router();

router.use('/preview', builderPreviewRouter);
router.use('/save', builderSaveRouter);
router.use('/visual-authoring', builderVisualAuthoringRouter);

router.get('/', async (req: Request, res: Response): Promise<void> => {
  const requestedWorkflowId = typeof req.query['workflowId'] === 'string' && req.query['workflowId'].trim()
    ? req.query['workflowId'].trim()
    : '';
  if (!requestedWorkflowId) {
    res.type('html').status(200).send(builderHtml(baseViewModel(DEFAULT_WORKFLOW_ID)));
    return;
  }

  const bearerToken = (req as Request & { bearerToken?: string }).bearerToken;
  const client = createGoClient(bearerToken);
  try {
    const { data: resp } = await client.get<{ data: BuilderWorkflowRecord }>(`/api/v1/workflows/${requestedWorkflowId}`);
    const viewModel = fromWorkflowRecord(resp.data);
    const { projection, errorMessage } = await fetchVisualProjection(client, requestedWorkflowId);
    if (projection) {
      viewModel.visualProjection = projection;
      viewModel.projectionSource = 'api';
    }
    if (errorMessage) {
      viewModel.graphErrorMessage = errorMessage;
    }
    res.type('html').status(200).send(builderHtml(viewModel));
  } catch (err: unknown) {
    const fallback = baseViewModel(requestedWorkflowId);
    fallback.errorMessage = loadErrorMessage(requestedWorkflowId, err);
    res.type('html').status(200).send(builderHtml(fallback));
  }
});

export default router;
