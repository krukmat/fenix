import type { FlowVisualEdge, FlowVisualNode } from './flowLayout';

export const FLOW_EDGE_TYPES = new Set(['next', 'execution']);

const KIND_RANK: Record<string, number> = {
  workflow: 0,
  trigger: 1,
  action: 2,
  decision: 2,
  grounds: 3,
  permit: 3,
  delegate: 3,
  invariant: 3,
  budget: 3,
};

type FlowGraph = Readonly<{
  nodeMap: ReadonlyMap<string, FlowVisualNode>;
  predecessors: ReadonlyMap<string, ReadonlySet<string>>;
  successors: ReadonlyMap<string, ReadonlySet<string>>;
}>;

export function buildColumnMap(nodes: readonly FlowVisualNode[], edges: readonly FlowVisualEdge[]): Map<string, number> {
  const flowGraph = buildFlowGraph(nodes, edges);
  const indegree = new Map<string, number>();
  const columns = new Map<string, number>();
  const { nodeMap, predecessors, successors } = flowGraph;

  nodes.forEach((node) => {
    indegree.set(node.id, predecessors.get(node.id)?.size ?? 0);
  });

  const sortedNodes = sortNodes(nodes);
  const queued = new Set<string>();
  const queue = sortedNodes
    .filter((node) => (indegree.get(node.id) ?? 0) === 0)
    .map((node) => {
      queued.add(node.id);
      return node.id;
    });

  while (queue.length > 0) {
    const nodeId = queue.shift();
    if (!nodeId) continue;

    const nodePredecessors = predecessors.get(nodeId) ?? new Set<string>();
    const nodeSuccessors = successors.get(nodeId) ?? new Set<string>();
    const predecessorColumns = [...nodePredecessors].map((predecessorId) => columns.get(predecessorId) ?? 0);
    const fallbackColumn = nodePredecessors.size === 0 && nodeSuccessors.size === 0 ? kindRank(nodeMap.get(nodeId)?.kind ?? '') : 0;
    columns.set(nodeId, predecessorColumns.length > 0 ? Math.max(...predecessorColumns) + 1 : fallbackColumn);

    orderSuccessors(nodeSuccessors, nodeMap).forEach((successor) => {
      const nextIndegree = (indegree.get(successor.id) ?? 0) - 1;
      indegree.set(successor.id, nextIndegree);
      if (nextIndegree === 0 && !queued.has(successor.id)) {
        queue.push(successor.id);
        queued.add(successor.id);
      }
    });
  }

  sortedNodes.forEach((node) => {
    if (!columns.has(node.id)) columns.set(node.id, 0);
  });

  return columns;
}

export function buildRowMap(
  nodes: readonly FlowVisualNode[],
  edges: readonly FlowVisualEdge[],
  columnMap: ReadonlyMap<string, number>,
): Map<string, number> {
  const flowGraph = buildFlowGraph(nodes, edges);
  const rowMap = new Map<string, number>();
  const usedRowsByColumn = new Map<number, Set<number>>();
  const { predecessors, successors } = flowGraph;
  const nodeMap = new Map(nodes.map((node) => [node.id, node]));
  const sortedNodes = [...sortNodes(nodes)].sort((a, b) => {
    const columnDelta = (columnMap.get(a.id) ?? 0) - (columnMap.get(b.id) ?? 0);
    if (columnDelta !== 0) return columnDelta;
    const rankDelta = kindRank(a.kind) - kindRank(b.kind);
    if (rankDelta !== 0) return rankDelta;
    return a.id.localeCompare(b.id);
  });

  sortedNodes.forEach((node) => {
    const column = columnMap.get(node.id) ?? 0;
    const nodePredecessors = [...(predecessors.get(node.id) ?? new Set<string>())].sort();
    const nodeSuccessors = successors.get(node.id) ?? new Set<string>();
    let preferredRow = 0;

    if (nodePredecessors.length > 0) {
      if (nodePredecessors.length === 1) {
        const predecessorId = nodePredecessors[0];
        const predecessorRow = rowMap.get(predecessorId) ?? 0;
        const siblingIds = orderSuccessorIds(successors.get(predecessorId) ?? new Set<string>(), nodeMap);
        const siblingIndex = Math.max(siblingIds.indexOf(node.id), 0);
        preferredRow = predecessorRow + siblingIndex;
      } else {
        preferredRow = Math.min(...nodePredecessors.map((predecessorId) => rowMap.get(predecessorId) ?? 0));
      }
    } else if (nodeSuccessors.size === 0) {
      preferredRow = nextUnusedRow(usedRowsByColumn.get(column), 0);
    }

    rowMap.set(node.id, reserveRow(usedRowsByColumn, column, preferredRow));
  });

  return rowMap;
}

export function sortNodes(nodes: readonly FlowVisualNode[]): FlowVisualNode[] {
  return [...nodes].sort((a, b) => {
    const rankDelta = kindRank(a.kind) - kindRank(b.kind);
    if (rankDelta !== 0) return rankDelta;
    return a.id.localeCompare(b.id);
  });
}

function buildFlowGraph(nodes: readonly FlowVisualNode[], edges: readonly FlowVisualEdge[]): FlowGraph {
  const nodeMap = new Map(nodes.map((node) => [node.id, node]));
  const predecessors = new Map<string, Set<string>>();
  const successors = new Map<string, Set<string>>();

  nodes.forEach((node) => {
    predecessors.set(node.id, new Set());
    successors.set(node.id, new Set());
  });

  edges
    .filter((edge) => FLOW_EDGE_TYPES.has(edge.connection_type ?? 'execution'))
    .forEach((edge) => {
      if (!nodeMap.has(edge.from) || !nodeMap.has(edge.to)) return;
      const edgePredecessors = predecessors.get(edge.to);
      const edgeSuccessors = successors.get(edge.from);
      if (!edgePredecessors || !edgeSuccessors || edgePredecessors.has(edge.from)) return;
      edgePredecessors.add(edge.from);
      edgeSuccessors.add(edge.to);
    });

  return { nodeMap, predecessors, successors };
}

function kindRank(kind: string): number {
  return KIND_RANK[kind] ?? 4;
}

function orderSuccessors(successors: ReadonlySet<string>, nodeMap: ReadonlyMap<string, FlowVisualNode>): FlowVisualNode[] {
  return [...successors]
    .map((successorId) => nodeMap.get(successorId))
    .filter((node): node is FlowVisualNode => Boolean(node))
    .sort((a, b) => {
      const rankDelta = kindRank(a.kind) - kindRank(b.kind);
      if (rankDelta !== 0) return rankDelta;
      return a.id.localeCompare(b.id);
    });
}

function orderSuccessorIds(successors: ReadonlySet<string>, nodeMap: ReadonlyMap<string, FlowVisualNode>): string[] {
  return orderSuccessors(successors, nodeMap).map((node) => node.id);
}

function reserveRow(usedRowsByColumn: Map<number, Set<number>>, column: number, preferredRow: number): number {
  const usedRows = usedRowsByColumn.get(column) ?? new Set<number>();
  usedRowsByColumn.set(column, usedRows);
  const row = nextUnusedRow(usedRows, preferredRow);
  usedRows.add(row);
  return row;
}

function nextUnusedRow(usedRows: ReadonlySet<number> | undefined, startRow: number): number {
  let row = startRow;
  while (usedRows?.has(row)) {
    row += 1;
  }
  return row;
}
