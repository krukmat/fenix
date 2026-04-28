import { Router, Request, Response } from 'express';
import { createGoClient } from '../services/goClient';

type BffRequest = Request & { bearerToken?: string };

type Diagnostic = { code?: string; description?: string; message?: string; location?: string; line?: number; column?: number };
type VisualNode = { id: string; kind: string; label: string; color: string; position: { x: number; y: number } };
type VisualEdge = { from: string; to: string };
type PreviewData = {
  passed: boolean;
  diagnostics?: { violations?: Diagnostic[]; warnings?: Diagnostic[] };
  conformance?: { profile?: string; details?: Diagnostic[] };
  visual_graph?: { workflow_name?: string; nodes?: VisualNode[]; edges?: VisualEdge[] };
};
type PreviewEnvelope = { data?: PreviewData };

const router = Router();

router.post('/', async (req: BffRequest, res: Response): Promise<void> => {
  try {
    const data = await fetchPreview(req);
    res.type('html').status(200).send(renderPreview(data));
  } catch (err) {
    res.type('html').status(502).send(renderPreviewError(err));
  }
});

async function fetchPreview(req: BffRequest): Promise<PreviewData> {
  const client = createGoClient(req.bearerToken);
  const response = await client.post<PreviewEnvelope>(
    '/api/v1/workflows/preview',
    { dsl_source: formValue(req.body, 'source'), spec_source: formValue(req.body, 'spec_source') },
    { validateStatus: (status) => status < 500 },
  );
  return response.data.data ?? { passed: false };
}

function renderPreview(data: PreviewData): string {
  return [
    `<span>${data.passed ? 'Preview synced' : 'Preview has diagnostics'}</span>`,
    renderDiagnostics(data),
    renderGraph(data),
  ].join('');
}

function renderDiagnostics(data: PreviewData): string {
  const diagnostics = collectDiagnostics(data);
  const items = diagnostics.length === 0
    ? '<li class="diagnostic-empty">No validation diagnostics for current draft.</li>'
    : diagnostics.map(renderDiagnostic).join('');
  return `<ul class="diagnostics-list" id="builder-diagnostics" aria-live="polite" hx-swap-oob="true">${items}</ul>`;
}

function renderDiagnostic(item: Diagnostic): string {
  const label = item.code ?? item.location ?? 'diagnostic';
  const text = item.description ?? item.message ?? 'Validation diagnostic';
  return `<li><strong>${escapeHtml(label)}</strong>: ${escapeHtml(text)}</li>`;
}

function renderGraph(data: PreviewData): string {
  const graph = data.visual_graph;
  const nodes = graph?.nodes ?? [];
  const projectionPayload = escapeHtml(JSON.stringify({
    workflow_name: graph?.workflow_name ?? '',
    nodes,
    edges: graph?.edges ?? [],
    conformance: data.conformance,
  }));
  return `<div class="graph-shell" id="builder-graph" data-projection-source="api" data-workflow-name="${escapeHtml(graph?.workflow_name ?? '')}" data-projection-payload="${projectionPayload}" hx-swap-oob="true"><div id="builder-canvas-root" role="img" aria-label="Dynamic workflow graph canvas"></div><p class="graph-caption">Live workflow projection loaded for the bound workflow.</p>${renderInspector(data, nodes[0])}</div>`;
}

function renderInspector(data: PreviewData, node?: VisualNode): string {
  const selected = node ? `${node.kind} / ${node.label}` : 'No node selected';
  return `<aside class="inspector" id="builder-inspector" aria-labelledby="inspector-title"><div class="inspector-header"><h3 class="inspector-title" id="inspector-title">Inspector</h3></div><div class="inspector-grid"><div class="inspector-block"><span class="inspector-label">Selected node</span><p class="inspector-value">${escapeHtml(selected)}</p></div><div class="inspector-block"><span class="inspector-label">Conformance</span><p class="inspector-value">${escapeHtml(data.conformance?.profile ?? 'unknown')}</p></div><div class="inspector-block"><span class="inspector-label">Diagnostics</span><p class="inspector-value">${collectDiagnostics(data).length} current finding(s).</p></div></div></aside>`;
}

function collectDiagnostics(data: PreviewData): Diagnostic[] {
  return [
    ...(data.diagnostics?.violations ?? []),
    ...(data.diagnostics?.warnings ?? []),
    ...(data.conformance?.details ?? []),
  ];
}

function renderPreviewError(err: unknown): string {
  const message = err instanceof Error ? err.message : 'Preview request failed';
  return `<span>Preview unavailable</span><ul class="diagnostics-list" id="builder-diagnostics" aria-live="polite" hx-swap-oob="true"><li><strong>preview_error</strong>: ${escapeHtml(message)}</li></ul>`;
}

function formValue(body: unknown, key: string): string {
  if (typeof body !== 'object' || body === null) {
    return '';
  }
  const value = (body as Record<string, unknown>)[key];
  return typeof value === 'string' ? value : '';
}

function escapeHtml(value: string): string {
  return value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#39;');
}

export default router;
