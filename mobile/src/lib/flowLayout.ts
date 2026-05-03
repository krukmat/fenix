// CLSF-81a/81b: dependency-free contract and algorithm for the mobile read-only workflow graph layout.
import { buildColumnMap, buildRowMap, FLOW_EDGE_TYPES, sortNodes } from './flowLayoutTopology';

export const FLOW_LAYOUT_SPACING = {
  canvasPadding: 24,
  columnGap: 64,
  rowGap: 40,
  nodeWidth: 200,
  nodeHeight: 72,
} as const;

export type FlowNodeKind =
  | 'workflow'
  | 'trigger'
  | 'action'
  | 'decision'
  | 'grounds'
  | 'permit'
  | 'delegate'
  | 'invariant'
  | 'budget';

export type FlowPoint = Readonly<{
  x: number;
  y: number;
}>;

export type FlowSize = Readonly<{
  width: number;
  height: number;
}>;

export type FlowVisualNode = Readonly<{
  id: string;
  kind: FlowNodeKind | string;
  label: string;
}>;

export type FlowVisualEdge = Readonly<{
  id?: string;
  from: string;
  to: string;
  connection_type?: string;
}>;

export type FlowLayoutInput = Readonly<{
  nodes: readonly FlowVisualNode[];
  edges: readonly FlowVisualEdge[];
  nodeSize?: FlowSize;
  spacing?: Partial<typeof FLOW_LAYOUT_SPACING>;
}>;

export type FlowNodeBox = Readonly<{
  id: string;
  kind: FlowNodeKind | string;
  label: string;
  x: number;
  y: number;
  width: number;
  height: number;
}>;

export type FlowConnectorSegment = Readonly<{
  id: string;
  from: string;
  to: string;
  start: FlowPoint;
  end: FlowPoint;
  connectionType: string;
}>;

export type FlowLayoutBounds = Readonly<{
  width: number;
  height: number;
}>;

export type FlowLayoutResult = Readonly<{
  nodes: readonly FlowNodeBox[];
  connectors: readonly FlowConnectorSegment[];
  bounds: FlowLayoutBounds;
}>;

type ResolvedSpacing = typeof FLOW_LAYOUT_SPACING;

export function layoutWorkflowGraph(input: FlowLayoutInput): FlowLayoutResult {
  const spacing = resolveSpacing(input.spacing);
  const nodeSize = input.nodeSize ?? {
    width: spacing.nodeWidth,
    height: spacing.nodeHeight,
  };
  const nodes = buildNodeBoxes(input.nodes, input.edges, nodeSize, spacing);
  return {
    nodes,
    connectors: buildConnectors(input.edges, nodes),
    bounds: buildBounds(nodes, spacing),
  };
}

function resolveSpacing(spacing?: Partial<typeof FLOW_LAYOUT_SPACING>): ResolvedSpacing {
  return {
    ...FLOW_LAYOUT_SPACING,
    ...spacing,
  };
}

function buildNodeBoxes(
  nodes: readonly FlowVisualNode[],
  edges: readonly FlowVisualEdge[],
  nodeSize: FlowSize,
  spacing: ResolvedSpacing,
): FlowNodeBox[] {
  const columnMap = buildColumnMap(nodes, edges);
  const rowMap = buildRowMap(nodes, edges, columnMap);
  return sortNodes(nodes).map((node, index) => {
    const column = columnMap.get(node.id) ?? 0;
    const row = rowMap.get(node.id) ?? index;
    return {
      id: node.id,
      kind: node.kind,
      label: node.label,
      x: spacing.canvasPadding + column * (nodeSize.width + spacing.columnGap),
      y: spacing.canvasPadding + row * (nodeSize.height + spacing.rowGap),
      width: nodeSize.width,
      height: nodeSize.height,
    };
  });
}

function buildConnectors(edges: readonly FlowVisualEdge[], nodes: readonly FlowNodeBox[]): FlowConnectorSegment[] {
  const nodeMap = new Map(nodes.map((node) => [node.id, node]));
  return edges
    .filter((edge) => FLOW_EDGE_TYPES.has(edge.connection_type ?? 'execution'))
    .flatMap((edge, index) => {
    const from = nodeMap.get(edge.from);
    const to = nodeMap.get(edge.to);
    if (!from || !to) return [];
    return [connectorFromEdge(edge, index, from, to)];
  });
}

function connectorFromEdge(
  edge: FlowVisualEdge,
  index: number,
  from: FlowNodeBox,
  to: FlowNodeBox,
): FlowConnectorSegment {
  return {
    id: edge.id ?? `edge-${index + 1}`,
    from: edge.from,
    to: edge.to,
    start: rightCenter(from),
    end: leftCenter(to),
    connectionType: edge.connection_type ?? 'execution',
  };
}

function rightCenter(node: FlowNodeBox): FlowPoint {
  return {
    x: node.x + node.width,
    y: node.y + node.height / 2,
  };
}

function leftCenter(node: FlowNodeBox): FlowPoint {
  return {
    x: node.x,
    y: node.y + node.height / 2,
  };
}

function buildBounds(nodes: readonly FlowNodeBox[], spacing: ResolvedSpacing): FlowLayoutBounds {
  if (nodes.length === 0) {
    return {
      width: spacing.canvasPadding * 2,
      height: spacing.canvasPadding * 2,
    };
  }
  return {
    width: Math.max(...nodes.map((node) => node.x + node.width)) + spacing.canvasPadding,
    height: Math.max(...nodes.map((node) => node.y + node.height)) + spacing.canvasPadding,
  };
}
