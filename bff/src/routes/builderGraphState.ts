// CLSF-77a: in-memory graph state model for the visual authoring canvas
export const SUPPORTED_KINDS = [
  'workflow',
  'trigger',
  'action',
  'decision',
  'grounds',
  'permit',
  'delegate',
  'invariant',
  'budget',
] as const;

export type VNodeKind = (typeof SUPPORTED_KINDS)[number];

export type VPosition = { x: number; y: number };

type VNode = { id: string; kind: VNodeKind; label: string; position: VPosition };
type VEdge = { id: string; from: string; to: string; connection_type: string };

type PayloadNode = { id: string; kind: string; label: string; position: VPosition };
type PayloadEdge = { id: string; from: string; to: string; connection_type: string };

export type VisualAuthoringPayload = {
  graph: {
    workflow_name: string;
    nodes: PayloadNode[];
    edges: PayloadEdge[];
  };
};

type ProjectionNode = { id: string; kind: string; label: string; color: string; position: VPosition };
type ProjectionEdge = { id?: string; from: string; to: string; connection_type?: string };
export type VisualProjectionInput = { nodes: ProjectionNode[]; edges: ProjectionEdge[] };

export type GraphState = {
  addNode: (kind: VNodeKind, label: string, position: VPosition) => string;
  removeNode: (id: string) => void;
  addEdge: (from: string, to: string, connectionType: string) => string;
  removeEdge: (id: string) => void;
  updateNodePosition: (id: string, position: VPosition) => void;
  getSelectedNodeId: () => string | null;
  setSelectedNodeId: (id: string | null) => void;
  loadFromProjection: (projection: VisualProjectionInput) => void;
  toPayload: (workflowName: string) => VisualAuthoringPayload;
};

function isSupportedKind(kind: string): kind is VNodeKind {
  return (SUPPORTED_KINDS as readonly string[]).includes(kind);
}

function resolveEdgeId(e: ProjectionEdge, counter: number): string {
  return e.id ?? `vedge-${counter}`;
}

function validateEdgeEndpoints(from: string, to: string, nodes: VNode[]): void {
  if (from === to) throw new Error('Self-loop edges are not allowed');
  if (!nodes.some((n) => n.id === from)) throw new Error(`Source node not found: ${from}`);
  if (!nodes.some((n) => n.id === to)) throw new Error(`Target node not found: ${to}`);
}

function nodeToPayload(n: VNode): PayloadNode {
  return { id: n.id, kind: n.kind, label: n.label, position: n.position };
}

function edgeToPayload(e: VEdge): PayloadEdge {
  return { id: e.id, from: e.from, to: e.to, connection_type: e.connection_type };
}

function removeNodeEdges(edges: VEdge[], nodeId: string): void {
  for (let i = edges.length - 1; i >= 0; i--) {
    const edge = edges[i];
    if (edge && (edge.from === nodeId || edge.to === nodeId)) edges.splice(i, 1);
  }
}

function applyProjectionNodes(nodes: VNode[], projection: VisualProjectionInput): void {
  for (const n of projection.nodes) {
    const kind = isSupportedKind(n.kind) ? n.kind : 'action';
    nodes.push({ id: n.id, kind, label: n.label, position: n.position });
  }
}

function applyProjectionEdges(edges: VEdge[], projection: VisualProjectionInput, startCounter: number): number {
  let counter = startCounter;
  for (const e of projection.edges) {
    counter += 1;
    edges.push({ id: resolveEdgeId(e, counter), from: e.from, to: e.to, connection_type: e.connection_type ?? 'execution' });
  }
  return counter;
}

export function createGraphState(): GraphState {
  const nodes: VNode[] = [];
  const edges: VEdge[] = [];
  const kindCounters = new Map<string, number>();
  let edgeCounter = 0;
  let selectedNodeId: string | null = null;

  function addNode(kind: VNodeKind, label: string, position: VPosition): string {
    if (!isSupportedKind(kind)) throw new Error(`Unsupported node kind: ${kind}`);
    const ordinal = (kindCounters.get(kind) ?? 0) + 1;
    kindCounters.set(kind, ordinal);
    const id = `vnode-${kind}-${ordinal}`;
    nodes.push({ id, kind, label, position });
    return id;
  }

  function removeNode(id: string): void {
    const index = nodes.findIndex((n) => n.id === id);
    if (index === -1) return;
    nodes.splice(index, 1);
    removeNodeEdges(edges, id);
    if (selectedNodeId === id) selectedNodeId = null;
  }

  function addEdge(from: string, to: string, connectionType: string): string {
    validateEdgeEndpoints(from, to, nodes);
    const existing = edges.find((e) => e.from === from && e.to === to);
    if (existing) return existing.id;
    edgeCounter += 1;
    const id = `vedge-${edgeCounter}`;
    edges.push({ id, from, to, connection_type: connectionType });
    return id;
  }

  function loadFromProjection(projection: VisualProjectionInput): void {
    nodes.length = 0;
    edges.length = 0;
    selectedNodeId = null;
    kindCounters.clear();
    edgeCounter = 0;
    applyProjectionNodes(nodes, projection);
    edgeCounter = applyProjectionEdges(edges, projection, edgeCounter);
  }

  function toPayload(workflowName: string): VisualAuthoringPayload {
    return {
      graph: {
        workflow_name: workflowName,
        nodes: nodes.map(nodeToPayload),
        edges: edges.map(edgeToPayload),
      },
    };
  }

  return {
    addNode, removeNode, addEdge, loadFromProjection, toPayload,
    removeEdge: (id: string) => { const i = edges.findIndex((e) => e.id === id); if (i !== -1) edges.splice(i, 1); },
    updateNodePosition: (id: string, position: VPosition) => { const n = nodes.find((x) => x.id === id); if (n) n.position = position; },
    getSelectedNodeId: () => selectedNodeId,
    setSelectedNodeId: (id: string | null) => { selectedNodeId = id; },
  };
}
