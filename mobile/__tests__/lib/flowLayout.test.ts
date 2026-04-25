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
  { id: 'edge-trigger-action', from: 'trigger-1', to: 'action-1', connection_type: 'execution' },
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

describe('layoutWorkflowGraph', () => {
  it('lays out a four-node workflow fixture with valid connectors and no overlaps', () => {
    const layout = layoutWorkflowGraph({ nodes: fixtureNodes, edges: fixtureEdges });
    const nodeIds = layout.nodes.map((node) => node.id);
    const connectorIds = layout.connectors.map((connector) => connector.id);

    expect(nodeIds).toEqual(['workflow-1', 'trigger-1', 'action-1', 'grounds-1']);
    expect(connectorIds).toEqual(['edge-workflow-trigger', 'edge-trigger-action', 'edge-trigger-grounds']);
    expect(layout.connectors).toEqual([
      expect.objectContaining({ from: 'workflow-1', to: 'trigger-1', connectionType: 'execution' }),
      expect.objectContaining({ from: 'trigger-1', to: 'action-1', connectionType: 'execution' }),
      expect.objectContaining({ from: 'trigger-1', to: 'grounds-1', connectionType: 'requirement' }),
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
});
