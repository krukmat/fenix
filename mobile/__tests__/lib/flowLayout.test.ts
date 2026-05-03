import { describe, expect, it } from '@jest/globals';
import { layoutWorkflowGraph, type FlowNodeBox, type FlowVisualEdge, type FlowVisualNode } from '../../src/lib/flowLayout';

const fixtureNodes: FlowVisualNode[] = [
  { id: 'workflow-1', kind: 'workflow', label: 'sales_followup' },
  { id: 'trigger-1', kind: 'trigger', label: 'deal.updated' },
  { id: 'action-1', kind: 'action', label: 'notify owner' },
  { id: 'grounds-1', kind: 'grounds', label: 'permit + grounds' },
];

const fixtureEdges: FlowVisualEdge[] = [
  { id: 'edge-workflow-trigger', from: 'workflow-1', to: 'trigger-1', connection_type: 'execution' },
  { id: 'edge-trigger-action', from: 'trigger-1', to: 'action-1', connection_type: 'next' },
  { id: 'edge-trigger-grounds', from: 'trigger-1', to: 'grounds-1', connection_type: 'requirement' },
];

function boxesOverlap(a: FlowNodeBox, b: FlowNodeBox): boolean {
  return a.x < b.x + b.width && a.x + a.width > b.x && a.y < b.y + b.height && a.y + a.height > b.y;
}

function expectNodeInsideBounds(node: FlowNodeBox, width: number, height: number): void {
  expect(node.x).toBeGreaterThanOrEqual(0);
  expect(node.y).toBeGreaterThanOrEqual(0);
  expect(node.x + node.width).toBeLessThanOrEqual(width);
  expect(node.y + node.height).toBeLessThanOrEqual(height);
}

function getNode(layout: ReturnType<typeof layoutWorkflowGraph>, id: string): FlowNodeBox {
  const node = layout.nodes.find((candidate) => candidate.id === id);
  expect(node).toBeDefined();
  return node as FlowNodeBox;
}

function getConnector(layout: ReturnType<typeof layoutWorkflowGraph>, id: string) {
  const connector = layout.connectors.find((candidate) => candidate.id === id);
  expect(connector).toBeDefined();
  return connector;
}

describe('layoutWorkflowGraph', () => {
  it('lays out a four-node workflow fixture with valid connectors and no overlaps', () => {
    const layout = layoutWorkflowGraph({ nodes: fixtureNodes, edges: fixtureEdges });
    const nodeIds = layout.nodes.map((node) => node.id);
    const connectorIds = layout.connectors.map((connector) => connector.id);

    expect(nodeIds).toEqual(['workflow-1', 'trigger-1', 'action-1', 'grounds-1']);
    expect(connectorIds).toEqual(['edge-workflow-trigger', 'edge-trigger-action']);
    expect(layout.connectors).toEqual([
      expect.objectContaining({ from: 'workflow-1', to: 'trigger-1', connectionType: 'execution' }),
      expect.objectContaining({ from: 'trigger-1', to: 'action-1', connectionType: 'next' }),
    ]);

    for (let i = 0; i < layout.nodes.length; i += 1) {
      for (let j = i + 1; j < layout.nodes.length; j += 1) {
        expect(boxesOverlap(layout.nodes[i], layout.nodes[j])).toBe(false);
      }
    }
  });

  it('returns deterministic positive coordinates inside computed bounds', () => {
    const first = layoutWorkflowGraph({ nodes: fixtureNodes, edges: fixtureEdges });
    const second = layoutWorkflowGraph({ nodes: fixtureNodes, edges: fixtureEdges });

    expect(second).toEqual(first);
    expect(first.bounds.width).toBeGreaterThan(0);
    expect(first.bounds.height).toBeGreaterThan(0);
    first.nodes.forEach((node) => {
      expectNodeInsideBounds(node, first.bounds.width, first.bounds.height);
    });
  });

  // CLSF-81c3: edge cases

  it('returns minimal bounds and no connectors for an empty graph', () => {
    const layout = layoutWorkflowGraph({ nodes: [], edges: [] });

    expect(layout.nodes).toHaveLength(0);
    expect(layout.connectors).toHaveLength(0);
    expect(layout.bounds.width).toBeGreaterThan(0);
    expect(layout.bounds.height).toBeGreaterThan(0);
  });

  it('silently drops edges that reference non-existent node ids', () => {
    const nodes: FlowVisualNode[] = [
      { id: 'trigger-1', kind: 'trigger', label: 'deal.created' },
    ];
    const edges: FlowVisualEdge[] = [
      { id: 'edge-missing-from', from: 'ghost-99', to: 'trigger-1', connection_type: 'execution' },
      { id: 'edge-missing-to', from: 'trigger-1', to: 'ghost-99', connection_type: 'execution' },
      { id: 'edge-both-missing', from: 'ghost-a', to: 'ghost-b', connection_type: 'execution' },
    ];

    const layout = layoutWorkflowGraph({ nodes, edges });

    expect(layout.nodes).toHaveLength(1);
    expect(layout.connectors).toHaveLength(0);
  });

  it('renders connectors only for flow edge types', () => {
    const nodes: FlowVisualNode[] = [
      { id: 'workflow-1', kind: 'workflow', label: 'sales_followup' },
      { id: 'trigger-1', kind: 'trigger', label: 'deal.created' },
      { id: 'action-1', kind: 'action', label: 'notify owner' },
      { id: 'grounds-1', kind: 'grounds', label: 'permit + grounds' },
      { id: 'delegate-1', kind: 'delegate', label: 'delegate' },
    ];
    const edges: FlowVisualEdge[] = [
      { id: 'edge-contains', from: 'workflow-1', to: 'trigger-1', connection_type: 'contains' },
      { id: 'edge-next', from: 'trigger-1', to: 'action-1', connection_type: 'next' },
      { id: 'edge-requires', from: 'workflow-1', to: 'grounds-1', connection_type: 'requires' },
      { id: 'edge-governs', from: 'delegate-1', to: 'workflow-1', connection_type: 'governs' },
    ];

    const layout = layoutWorkflowGraph({ nodes, edges });

    expect(layout.connectors).toEqual([
      expect.objectContaining({ id: 'edge-next', from: 'trigger-1', to: 'action-1', connectionType: 'next' }),
    ]);
  });

  it('includes explicit execution edges and defaults missing edge types to execution', () => {
    const nodes: FlowVisualNode[] = [
      { id: 'workflow-1', kind: 'workflow', label: 'sales_followup' },
      { id: 'trigger-1', kind: 'trigger', label: 'deal.created' },
      { id: 'action-1', kind: 'action', label: 'notify owner' },
    ];
    const edges: FlowVisualEdge[] = [
      { id: 'edge-execution', from: 'workflow-1', to: 'trigger-1', connection_type: 'execution' },
      { id: 'edge-default', from: 'trigger-1', to: 'action-1' },
    ];

    const layout = layoutWorkflowGraph({ nodes, edges });

    expect(layout.connectors).toEqual([
      expect.objectContaining({ id: 'edge-execution', connectionType: 'execution' }),
      expect.objectContaining({ id: 'edge-default', connectionType: 'execution' }),
    ]);
  });

  it('lays out disconnected governance nodes without errors and inside bounds', () => {
    const nodes: FlowVisualNode[] = [
      { id: 'grounds-1', kind: 'grounds', label: 'permit + grounds' },
      { id: 'permit-1', kind: 'permit', label: 'send_reply' },
    ];

    const layout = layoutWorkflowGraph({ nodes, edges: [] });

    expect(layout.nodes).toHaveLength(2);
    expect(layout.connectors).toHaveLength(0);
    layout.nodes.forEach((node) => {
      expectNodeInsideBounds(node, layout.bounds.width, layout.bounds.height);
    });
    for (let i = 0; i < layout.nodes.length; i += 1) {
      for (let j = i + 1; j < layout.nodes.length; j += 1) {
        expect(boxesOverlap(layout.nodes[i], layout.nodes[j])).toBe(false);
      }
    }
  });

  it('keeps a sequential same-kind chain in distinct columns on a shared row lane', () => {
    const nodes: FlowVisualNode[] = [
      { id: 'trigger-1', kind: 'trigger', label: 'deal.updated' },
      { id: 'action-set', kind: 'action', label: 'SET stage' },
      { id: 'action-notify', kind: 'action', label: 'NOTIFY owner' },
    ];
    const edges: FlowVisualEdge[] = [
      { id: 'edge-trigger-set', from: 'trigger-1', to: 'action-set', connection_type: 'next' },
      { id: 'edge-set-notify', from: 'action-set', to: 'action-notify', connection_type: 'next' },
    ];

    const layout = layoutWorkflowGraph({ nodes, edges });
    const trigger = getNode(layout, 'trigger-1');
    const actionSet = getNode(layout, 'action-set');
    const actionNotify = getNode(layout, 'action-notify');
    const triggerToSet = getConnector(layout, 'edge-trigger-set');
    const setToNotify = getConnector(layout, 'edge-set-notify');

    expect(actionSet.x).toBeGreaterThan(trigger.x);
    expect(actionNotify.x).toBeGreaterThan(actionSet.x);
    expect(trigger.y).toBe(actionSet.y);
    expect(actionSet.y).toBe(actionNotify.y);
    expect(triggerToSet?.start.y).toBe(triggerToSet?.end.y);
    expect(setToNotify?.start.y).toBe(setToNotify?.end.y);
  });

  it('keeps screenshot-style flow connectors horizontal even with governance nodes present', () => {
    const nodes: FlowVisualNode[] = [
      { id: 'workflow-1', kind: 'workflow', label: 'sales_followup' },
      { id: 'trigger-1', kind: 'trigger', label: 'deal.updated' },
      { id: 'action-set', kind: 'action', label: 'SET stage' },
      { id: 'action-notify', kind: 'action', label: 'NOTIFY owner' },
      { id: 'delegate-1', kind: 'delegate', label: 'delegate' },
      { id: 'grounds-1', kind: 'grounds', label: 'grounds' },
      { id: 'permit-1', kind: 'permit', label: 'permit' },
    ];
    const edges: FlowVisualEdge[] = [
      { id: 'edge-workflow-trigger', from: 'workflow-1', to: 'trigger-1', connection_type: 'execution' },
      { id: 'edge-trigger-set', from: 'trigger-1', to: 'action-set', connection_type: 'next' },
      { id: 'edge-set-notify', from: 'action-set', to: 'action-notify', connection_type: 'next' },
      { id: 'edge-governs', from: 'delegate-1', to: 'workflow-1', connection_type: 'governs' },
      { id: 'edge-requires-grounds', from: 'workflow-1', to: 'grounds-1', connection_type: 'requires' },
      { id: 'edge-requires-permit', from: 'grounds-1', to: 'permit-1', connection_type: 'requires' },
    ];

    const layout = layoutWorkflowGraph({ nodes, edges });
    const triggerToSet = getConnector(layout, 'edge-trigger-set');
    const setToNotify = getConnector(layout, 'edge-set-notify');

    expect(triggerToSet?.start.y).toBe(triggerToSet?.end.y);
    expect(setToNotify?.start.y).toBe(setToNotify?.end.y);
  });

  it('assigns strictly increasing x positions to a minimal generic linear chain', () => {
    const nodes: FlowVisualNode[] = [
      { id: 'A', kind: 'custom', label: 'A' },
      { id: 'B', kind: 'custom', label: 'B' },
      { id: 'C', kind: 'custom', label: 'C' },
      { id: 'D', kind: 'custom', label: 'D' },
    ];
    const edges: FlowVisualEdge[] = [
      { id: 'edge-a-b', from: 'A', to: 'B', connection_type: 'execution' },
      { id: 'edge-b-c', from: 'B', to: 'C', connection_type: 'execution' },
      { id: 'edge-c-d', from: 'C', to: 'D', connection_type: 'execution' },
    ];

    const layout = layoutWorkflowGraph({ nodes, edges });
    const nodeA = getNode(layout, 'A');
    const nodeB = getNode(layout, 'B');
    const nodeC = getNode(layout, 'C');
    const nodeD = getNode(layout, 'D');

    expect(nodeB.x).toBeGreaterThan(nodeA.x);
    expect(nodeC.x).toBeGreaterThan(nodeB.x);
    expect(nodeD.x).toBeGreaterThan(nodeC.x);
    layout.connectors.forEach((connector) => {
      expect(connector.start.y).toBe(connector.end.y);
    });
  });

  it('places branch siblings in the same successor column on different rows without overlap', () => {
    const nodes: FlowVisualNode[] = [
      { id: 'trigger-1', kind: 'trigger', label: 'deal.updated' },
      { id: 'action-a', kind: 'action', label: 'Action A' },
      { id: 'action-b', kind: 'action', label: 'Action B' },
    ];
    const edges: FlowVisualEdge[] = [
      { id: 'edge-trigger-a', from: 'trigger-1', to: 'action-a', connection_type: 'next' },
      { id: 'edge-trigger-b', from: 'trigger-1', to: 'action-b', connection_type: 'next' },
    ];

    const layout = layoutWorkflowGraph({ nodes, edges });
    const actionA = getNode(layout, 'action-a');
    const actionB = getNode(layout, 'action-b');

    expect(actionA.x).toBe(actionB.x);
    expect(actionA.y).not.toBe(actionB.y);
    expect(boxesOverlap(actionA, actionB)).toBe(false);
  });
});
