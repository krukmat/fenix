// CLSF-76a/76b/76c: BFF relay + HTMX fragments for visual authoring
import { Router, Request, Response } from 'express';
import { createGoClient } from '../services/goClient';

type BffRequest = Request & { bearerToken?: string };

type VisualNode = { id: string; kind: string; label: string; color: string; position: { x: number; y: number } };
type VisualEdge = { id?: string; from: string; to: string; connection_type?: string };
type VisualProjection = { workflow_name?: string; nodes?: VisualNode[]; edges?: VisualEdge[]; conformance?: { profile?: string; details?: unknown[] } };
type GraphEnvelopeData = { workflow_id?: string; conformance?: { profile?: string; details?: unknown[] }; visual_graph?: VisualProjection };
type Diagnostic = { code?: string; description?: string; message?: string; severity?: string };
type ValidateData = { diagnostics?: { violations?: Diagnostic[]; warnings?: Diagnostic[] }; conformance?: { profile?: string; details?: Diagnostic[] } };

const router = Router({ mergeParams: true });

router.post('/:id', async (req: BffRequest, res: Response): Promise<void> => {
  const raw = req.params['id'];
  const id = Array.isArray(raw) ? raw[0] : raw;
  const client = createGoClient(req.bearerToken);
  try {
    const upstream = await client.post(
      `/api/v1/workflows/${id}/visual-authoring`,
      req.body,
      { validateStatus: (s) => s < 500 },
    );
    await sendUpstreamResponse(req, res, client, id, upstream);
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Visual authoring request failed';
    res.status(502).json({ error: message });
  }
});

async function sendUpstreamResponse(
  req: Request,
  res: Response,
  client: ReturnType<typeof createGoClient>,
  id: string,
  upstream: { status: number; data?: { data?: unknown } },
): Promise<void> {
  if (upstream.status === 200) {
    const projection = await fetchVisualGraph(client, id);
    sendSaved(req, res, id, projection);
    return;
  }
  if (upstream.status === 422) {
    sendValidationError(req, res, (upstream.data?.data ?? {}) as ValidateData);
    return;
  }
  res.status(upstream.status).json(upstream.data);
}

function sendSaved(req: Request, res: Response, id: string, projection: VisualProjection): void {
  if (wantsJson(req)) res.status(200).json({ status: 'saved', projection });
  else res.type('html').status(200).send(renderSaved(id, projection));
}

function sendValidationError(req: Request, res: Response, payload: ValidateData): void {
  if (wantsJson(req)) res.status(422).json({ status: 'validation_error', diagnostics: collectDiagnostics(payload) });
  else res.type('html').status(422).send(renderErrors(payload));
}

function wantsJson(req: Request): boolean {
  return String(req.headers.accept ?? '').includes('application/json');
}

async function fetchVisualGraph(
  client: ReturnType<typeof createGoClient>,
  id: string,
): Promise<VisualProjection> {
  try {
    const r = await client.get<{ data: GraphEnvelopeData }>(`/api/v1/workflows/${id}/graph`, { params: { format: 'visual' } });
    return r.data?.data?.visual_graph ?? {};
  } catch {
    return {};
  }
}

function renderSaved(id: string, projection: VisualProjection): string {
  const nodes = projection.nodes ?? [];
  const edges = projection.edges ?? [];
  const profile = projection.conformance?.profile ?? 'unknown';
  return [
    `<span id="builder-preview-status" hx-swap-oob="true">Graph saved</span>`,
    renderGraphFragment(nodes, edges),
    renderInspectorFragment(id, nodes[0], profile),
  ].join('');
}

function renderGraphFragment(nodes: VisualNode[], edges: VisualEdge[]): string {
  const content = nodes.length === 0 ? renderEmptyGraph() : renderGraphSvg(nodes, edges);
  return `<div class="graph-shell" id="builder-graph" data-projection-source="api" hx-swap-oob="true">${content}</div>`;
}

function renderGraphSvg(nodes: VisualNode[], edges: VisualEdge[]): string {
  const width = Math.max(...nodes.map((n) => n.position.x)) + 230;
  const height = Math.max(...nodes.map((n) => n.position.y)) + 150;
  const index = new Map(nodes.map((n) => [n.id, n]));
  const defs = `<defs><marker id="arrowhead" markerWidth="10" markerHeight="8" refX="9" refY="4" orient="auto"><path d="M0,0 L10,4 L0,8 Z" fill="#8590a3"></path></marker></defs>`;
  return `<svg class="graph-canvas" viewBox="0 0 ${width} ${height}" role="img" aria-label="Live workflow graph">${defs}${edges.map((e) => renderEdge(e, index)).join('')}${nodes.map(renderNode).join('')}</svg>`;
}

function renderEdge(edge: VisualEdge, index: Map<string, VisualNode>): string {
  const from = index.get(edge.from);
  const to = index.get(edge.to);
  if (!from || !to) return '';
  return `<line class="graph-edge" x1="${from.position.x + 140}" y1="${from.position.y + 66}" x2="${to.position.x + 30}" y2="${to.position.y + 66}"></line>`;
}

function renderNode(node: VisualNode): string {
  const kindClass = graphNodeClass(node.kind);
  return `<rect class="${kindClass}" x="${node.position.x + 30}" y="${node.position.y + 30}" width="170" height="72" style="stroke:${escapeHtml(node.color)}"></rect><text class="graph-label" x="${node.position.x + 50}" y="${node.position.y + 62}">${escapeHtml(node.label)}</text><text class="graph-meta" x="${node.position.x + 50}" y="${node.position.y + 84}">${escapeHtml(node.kind)}</text>`;
}

function renderInspectorFragment(id: string, node: VisualNode | undefined, profile: string): string {
  const selected = node ? `${node.kind} / ${node.label}` : 'No node selected';
  return `<aside class="inspector" id="builder-inspector" aria-labelledby="inspector-title" hx-swap-oob="true"><div class="inspector-header"><h3 class="inspector-title" id="inspector-title">Inspector — ${escapeHtml(id)}</h3></div><div class="inspector-grid"><div class="inspector-block"><span class="inspector-label">Selected node</span><p class="inspector-value">${escapeHtml(selected)}</p></div><div class="inspector-block"><span class="inspector-label">Conformance</span><p class="inspector-value">${escapeHtml(profile)}</p></div><div class="inspector-block"><span class="inspector-label">Status</span><p class="inspector-value">Saved</p></div></div></aside>`;
}

function renderErrors(data: ValidateData): string {
  const items = collectDiagnostics(data);
  return [
    `<span id="builder-preview-status" hx-swap-oob="true">Save failed — diagnostics</span>`,
    renderDiagnosticsFragment(items),
  ].join('');
}

function collectDiagnostics(data: ValidateData): Diagnostic[] {
  return [
    ...(data.diagnostics?.violations ?? []),
    ...(data.diagnostics?.warnings ?? []),
    ...(data.conformance?.details ?? []),
  ];
}

function renderDiagnosticsFragment(items: Diagnostic[]): string {
  const inner = items.length === 0
    ? '<li class="diagnostic-empty">No validation diagnostics for current draft.</li>'
    : items.map(renderDiagnosticItem).join('');
  return `<ul class="diagnostics-list" id="builder-diagnostics" aria-live="polite" hx-swap-oob="true">${inner}</ul>`;
}

function renderDiagnosticItem(item: Diagnostic): string {
  const label = item.code ?? 'diagnostic';
  const text = item.description ?? item.message ?? 'Validation diagnostic';
  return `<li><strong>${escapeHtml(label)}</strong>: ${escapeHtml(text)}</li>`;
}

function renderEmptyGraph(): string {
  return '<div class="graph-canvas" role="img" aria-label="Empty workflow graph">No graph nodes returned for current draft.</div>';
}

function graphNodeClass(kind: string): string {
  if (kind === 'action') return 'graph-node action';
  if (['grounds', 'permit', 'delegate', 'invariant', 'budget'].includes(kind)) return 'graph-node governance';
  return 'graph-node';
}

function escapeHtml(value: string): string {
  return value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#39;');
}

export default router;
