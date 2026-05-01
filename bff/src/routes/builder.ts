// CLSF-62-65: Web builder shell served by the BFF with HTMX loaded from CDN.
import { Router, Request, Response } from 'express';
import { createGoClient } from '../services/goClient';
import builderPreviewRouter from './builderPreview';
import builderSaveRouter from './builderSave';
import builderVisualAuthoringRouter from './builderVisualAuthoring';
import { upstreamMessage, upstreamStatus } from './adminAuth';
import {
  escHtml,
  renderBuilderShell,
  renderGraphAuthoringControlsShell,
  type BuilderViewModel,
  type VisualNode,
  type VisualEdge,
  type VisualProjection,
} from './builderShell';

const DEFAULT_WORKFLOW_ID = 'sales_followup';

interface BuilderWorkflowRecord {
  id: string; name: string; dsl_source: string; spec_source?: string | null;
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
  const width = Math.max(...nodes.map((n) => n.position.x)) + 230;
  const height = Math.max(...nodes.map((n) => n.position.y)) + 150;
  const index = new Map(nodes.map((n) => [n.id, n]));
  return `<svg class="graph-canvas" viewBox="0 0 ${width} ${height}" role="img" aria-label="Live workflow graph"><defs><marker id="arrowhead" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 Z" fill="#8590a3"></path></marker></defs>${edges.map((e) => renderEdge(e, index)).join('')}${nodes.map(renderNode).join('')}</svg>`;
}

const FIXTURE_GRAPH = `<svg class="graph-canvas" viewBox="0 0 640 380" role="img" aria-labelledby="builder-graph-title builder-graph-desc"><title id="builder-graph-title">Read-only workflow graph preview</title><desc id="builder-graph-desc">Fixture graph with workflow, trigger, action, and governance nodes connected by read-only edges.</desc><defs><marker id="arrowhead" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 Z" fill="#8590a3"></path></marker></defs><line class="graph-edge" x1="190" y1="80" x2="250" y2="80"></line><line class="graph-edge" x1="410" y1="80" x2="470" y2="80"></line><line class="graph-edge" x1="320" y1="130" x2="320" y2="215"></line><rect class="graph-node" x="40" y="40" width="150" height="80"></rect><text class="graph-label" x="62" y="75">Workflow</text><text class="graph-meta" x="62" y="98">sales_followup</text><rect class="graph-node" x="250" y="40" width="160" height="80"></rect><text class="graph-label" x="272" y="75">Trigger</text><text class="graph-meta" x="272" y="98">deal.updated</text><rect class="graph-node action" x="470" y="40" width="140" height="80"></rect><text class="graph-label" x="492" y="75">Action</text><text class="graph-meta" x="492" y="98">notify owner</text><rect class="graph-node governance" x="242" y="215" width="176" height="82"></rect><text class="graph-label" x="264" y="250">Governance</text><text class="graph-meta" x="264" y="273">permit + grounds</text></svg>`;

function renderGraphContent(viewModel: BuilderViewModel): string {
  if (viewModel.projectionSource !== 'api') return FIXTURE_GRAPH;
  const nodes = viewModel.visualProjection?.nodes ?? [];
  const edges = viewModel.visualProjection?.edges ?? [];
  return nodes.length === 0
    ? '<div class="graph-canvas" role="img" aria-label="Empty workflow graph">No graph nodes returned for current draft.</div>'
    : renderGraphSvg(nodes, edges);
}

function graphCaption(viewModel: BuilderViewModel): string {
  return viewModel.projectionSource === 'api'
    ? 'Live workflow projection loaded for the bound workflow.'
    : 'Read-only fixture projection. Live backend refresh is reserved for CLSF-66.';
}

function inspectorSelected(viewModel: BuilderViewModel): string {
  const firstNode = viewModel.visualProjection?.nodes?.[0];
  if (firstNode) return `${firstNode.kind} / ${firstNode.label}`;
  return viewModel.projectionSource === 'fixture' ? 'Workflow / sales_followup' : 'No node selected';
}

function inspectorProfile(viewModel: BuilderViewModel): string {
  return viewModel.visualProjection?.conformance?.profile ?? (viewModel.projectionSource === 'fixture' ? 'safe fixture' : 'unknown');
}

function inspectorValues(viewModel: BuilderViewModel): { selected: string; profile: string; diagnostics: string } {
  return {
    selected: inspectorSelected(viewModel),
    profile: inspectorProfile(viewModel),
    diagnostics: viewModel.projectionSource === 'fixture' ? 'No graph diagnostics for fixture projection.' : 'Projection loaded from workflow context.',
  };
}

function renderInspector(viewModel: BuilderViewModel): string {
  const { selected, profile, diagnostics } = inspectorValues(viewModel);
  return `<aside class="inspector" id="builder-inspector" aria-labelledby="inspector-title"><div class="inspector-header"><h3 class="inspector-title" id="inspector-title">Inspector</h3></div><div class="inspector-grid"><div class="inspector-block"><span class="inspector-label">Selected node</span><p class="inspector-value">${escHtml(selected)}</p></div><div class="inspector-block"><span class="inspector-label">Conformance</span><p class="inspector-value">${escHtml(profile)}</p></div><div class="inspector-block"><span class="inspector-label">Diagnostics</span><p class="inspector-value">${escHtml(diagnostics)}</p></div></div></aside>`;
}

function renderShellGuidance(viewModel: BuilderViewModel): string {
  const style = 'margin-bottom:16px;padding:12px 14px;border:1px solid var(--line);border-radius:8px;background:#fbfcfe;color:var(--muted);font-size:13px';
  if (viewModel.standalone) return `<div style="${style}">Builder opened without a bound admin workflow. Use this mode for previewing the shell, or return to admin workflows and open a real draft.</div>`;
  if (viewModel.errorMessage) return `<div style="${style}">This builder session is still bound to workflow <strong style="color:var(--text)">${escHtml(viewModel.workflowId)}</strong>. Fix the load issue or return to the admin detail to retry from the workflow record.</div>`;
  return `<div style="${style}">Editing workflow <strong style="color:var(--text)">${escHtml(viewModel.workflowName)}</strong> in the admin authoring flow. Save text or graph changes here, then return to workflow detail to review status and activate when ready.</div>`;
}

function renderBuilderNav(viewModel: BuilderViewModel): string {
  const detailLink = viewModel.standalone ? '' : `<a href="/bff/admin/workflows/${encodeURIComponent(viewModel.workflowId)}" style="display:inline-block;height:34px;line-height:34px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--text);text-decoration:none">Back to workflow detail</a>`;
  return `<div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap;margin-bottom:16px"><a href="/bff/admin/workflows" style="display:inline-block;height:34px;line-height:34px;padding:0 12px;border:1px solid var(--line);border-radius:6px;font-size:13px;color:var(--muted);text-decoration:none">Back to workflows</a>${detailLink}</div>`;
}

function renderEditorActions(viewModel: BuilderViewModel): string {
  if (viewModel.standalone) return `<span class="preview-status" id="builder-save-status">Text save unavailable in standalone mode</span>`;
  return `<div style="display:flex;align-items:center;gap:10px;flex-wrap:wrap"><button type="button" hx-post="/bff/builder/save/${encodeURIComponent(viewModel.workflowId)}" hx-include="#builder-editor-form" hx-target="#builder-save-status" hx-swap="outerHTML" style="height:32px;padding:0 12px;border:0;border-radius:6px;background:var(--accent);color:#fff;font-size:13px;font-weight:700;cursor:pointer">Save draft</button><span class="preview-status" id="builder-save-status">Unsaved text changes</span></div>`;
}

function builderHtml(viewModel: BuilderViewModel): string {
  const workflowLabel = viewModel.workflowId || DEFAULT_WORKFLOW_ID;
  const projectionPayload = viewModel.visualProjection ? escHtml(JSON.stringify(viewModel.visualProjection)) : '';
  const initialProjection = viewModel.visualProjection ? `<script id="builder-initial-projection" type="application/json">${projectionPayload}</script>` : '';
  return renderBuilderShell(
    workflowLabel, renderGraphAuthoringControlsShell(workflowLabel), projectionPayload, initialProjection,
    renderBuilderNav(viewModel), renderShellGuidance(viewModel),
    `Editor: ${escHtml(viewModel.workflowName)}`, renderEditorActions(viewModel), editorBanner(viewModel.errorMessage),
    escHtml(viewModel.source), escHtml(viewModel.specSource),
    graphBanner(viewModel.graphErrorMessage), viewModel.projectionSource, viewModel.workflowName,
    renderGraphContent(viewModel), graphCaption(viewModel), renderInspector(viewModel),
  );
}

function baseViewModel(workflowId: string): BuilderViewModel {
  return { workflowId, workflowName: workflowId, source: '', specSource: '', projectionSource: 'fixture', standalone: workflowId === DEFAULT_WORKFLOW_ID };
}

function fromWorkflowRecord(workflow: BuilderWorkflowRecord): BuilderViewModel {
  return { workflowId: workflow.id, workflowName: workflow.name, source: workflow.dsl_source, specSource: workflow.spec_source ?? '', projectionSource: 'fixture', standalone: false };
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
    const { data: resp } = await client.get<{ data: { visual_graph?: VisualProjection } }>(`/api/v1/workflows/${workflowId}/graph`, { params: { format: 'visual' } });
    return { projection: resp.data.visual_graph ?? { workflow_name: workflowId, nodes: [], edges: [] } };
  } catch (err: unknown) {
    return { projection: { workflow_name: workflowId, nodes: [], edges: [] }, errorMessage: `Visual projection for ${workflowId} is unavailable: ${upstreamMessage(err)}` };
  }
}

const router = Router();
router.use('/preview', builderPreviewRouter);
router.use('/save', builderSaveRouter);
router.use('/visual-authoring', builderVisualAuthoringRouter);

router.get('/', async (req: Request, res: Response): Promise<void> => {
  const requestedWorkflowId = typeof req.query['workflowId'] === 'string' && req.query['workflowId'].trim() ? req.query['workflowId'].trim() : '';
  if (!requestedWorkflowId) { res.type('html').status(200).send(builderHtml(baseViewModel(DEFAULT_WORKFLOW_ID))); return; }
  const bearerToken = (req as Request & { bearerToken?: string }).bearerToken;
  const client = createGoClient(bearerToken);
  try {
    const { data: resp } = await client.get<{ data: BuilderWorkflowRecord }>(`/api/v1/workflows/${requestedWorkflowId}`);
    const viewModel = fromWorkflowRecord(resp.data);
    const { projection, errorMessage } = await fetchVisualProjection(client, requestedWorkflowId);
    if (projection) { viewModel.visualProjection = projection; viewModel.projectionSource = 'api'; }
    if (errorMessage) { viewModel.graphErrorMessage = errorMessage; }
    res.type('html').status(200).send(builderHtml(viewModel));
  } catch (err: unknown) {
    const fallback = baseViewModel(requestedWorkflowId);
    fallback.errorMessage = loadErrorMessage(requestedWorkflowId, err);
    res.type('html').status(200).send(builderHtml(fallback));
  }
});

export default router;
