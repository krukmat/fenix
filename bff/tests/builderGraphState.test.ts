// CLSF-77a: in-memory graph state model tests
import {
  createGraphState,
  SUPPORTED_KINDS,
  type GraphState,
  type VNodeKind,
} from '../src/routes/builderGraphState';

describe('createGraphState', () => {
  let state: GraphState;

  beforeEach(() => {
    state = createGraphState();
  });

  // --- initial state ---

  it('starts with empty nodes and edges', () => {
    const payload = state.toPayload('wf');
    expect(payload.graph.nodes).toHaveLength(0);
    expect(payload.graph.edges).toHaveLength(0);
  });

  it('toPayload includes workflow_name', () => {
    const payload = state.toPayload('sales_followup');
    expect(payload.graph.workflow_name).toBe('sales_followup');
  });

  // --- addNode ---

  it('addNode returns a deterministic id vnode-<kind>-1 for the first node of that kind', () => {
    const id = state.addNode('workflow', 'Sales Followup', { x: 0, y: 0 });
    expect(id).toBe('vnode-workflow-1');
  });

  it('addNode ordinal increments per kind independently', () => {
    const id1 = state.addNode('action', 'Notify', { x: 0, y: 0 });
    const id2 = state.addNode('action', 'Send Email', { x: 260, y: 0 });
    const id3 = state.addNode('trigger', 'deal.updated', { x: 130, y: 0 });
    expect(id1).toBe('vnode-action-1');
    expect(id2).toBe('vnode-action-2');
    expect(id3).toBe('vnode-trigger-1');
  });

  it('addNode added node appears in toPayload nodes with correct fields', () => {
    state.addNode('trigger', 'deal.updated', { x: 100, y: 50 });
    const { graph } = state.toPayload('wf');
    expect(graph.nodes).toHaveLength(1);
    expect(graph.nodes[0]).toMatchObject({
      id: 'vnode-trigger-1',
      kind: 'trigger',
      label: 'deal.updated',
      position: { x: 100, y: 50 },
    });
  });

  it('addNode throws for unsupported kind', () => {
    expect(() => state.addNode('unknown_kind' as VNodeKind, 'Bad', { x: 0, y: 0 })).toThrow();
  });

  // --- removeNode ---

  it('removeNode removes the node from toPayload', () => {
    const id = state.addNode('action', 'Notify', { x: 0, y: 0 });
    state.removeNode(id);
    expect(state.toPayload('wf').graph.nodes).toHaveLength(0);
  });

  it('removeNode also removes edges that reference the removed node', () => {
    const n1 = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    const n2 = state.addNode('trigger', 'ev', { x: 200, y: 0 });
    const n3 = state.addNode('action', 'act', { x: 400, y: 0 });
    state.addEdge(n1, n2, 'execution');
    state.addEdge(n2, n3, 'execution');
    state.removeNode(n2);
    const { graph } = state.toPayload('wf');
    expect(graph.nodes).toHaveLength(2);
    expect(graph.edges).toHaveLength(0);
  });

  it('removeNode on unknown id is a no-op', () => {
    state.addNode('workflow', 'wf', { x: 0, y: 0 });
    expect(() => state.removeNode('nonexistent')).not.toThrow();
    expect(state.toPayload('wf').graph.nodes).toHaveLength(1);
  });

  // --- addEdge ---

  it('addEdge returns a deterministic id vedge-<ordinal>', () => {
    const n1 = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    const n2 = state.addNode('trigger', 'ev', { x: 200, y: 0 });
    const edgeId = state.addEdge(n1, n2, 'execution');
    expect(edgeId).toBe('vedge-1');
  });

  it('addEdge ordinal increments globally', () => {
    const n1 = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    const n2 = state.addNode('trigger', 'ev', { x: 200, y: 0 });
    const n3 = state.addNode('action', 'act', { x: 400, y: 0 });
    const e1 = state.addEdge(n1, n2, 'execution');
    const e2 = state.addEdge(n2, n3, 'execution');
    expect(e1).toBe('vedge-1');
    expect(e2).toBe('vedge-2');
  });

  it('addEdge edge appears in toPayload with correct fields', () => {
    const n1 = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    const n2 = state.addNode('trigger', 'ev', { x: 200, y: 0 });
    state.addEdge(n1, n2, 'execution');
    const { graph } = state.toPayload('wf');
    expect(graph.edges).toHaveLength(1);
    expect(graph.edges[0]).toMatchObject({
      id: 'vedge-1',
      from: n1,
      to: n2,
      connection_type: 'execution',
    });
  });

  it('addEdge throws when source node does not exist', () => {
    const n2 = state.addNode('trigger', 'ev', { x: 0, y: 0 });
    expect(() => state.addEdge('nonexistent', n2, 'execution')).toThrow();
  });

  it('addEdge throws when target node does not exist', () => {
    const n1 = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    expect(() => state.addEdge(n1, 'nonexistent', 'execution')).toThrow();
  });

  it('addEdge throws for self-loop', () => {
    const n1 = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    expect(() => state.addEdge(n1, n1, 'execution')).toThrow();
  });

  it('addEdge is a no-op (returns existing id) for duplicate from/to pair', () => {
    const n1 = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    const n2 = state.addNode('trigger', 'ev', { x: 200, y: 0 });
    const e1 = state.addEdge(n1, n2, 'execution');
    const e2 = state.addEdge(n1, n2, 'execution');
    expect(e1).toBe(e2);
    expect(state.toPayload('wf').graph.edges).toHaveLength(1);
  });

  // --- removeEdge ---

  it('removeEdge removes the edge from toPayload', () => {
    const n1 = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    const n2 = state.addNode('trigger', 'ev', { x: 200, y: 0 });
    const edgeId = state.addEdge(n1, n2, 'execution');
    state.removeEdge(edgeId);
    expect(state.toPayload('wf').graph.edges).toHaveLength(0);
  });

  it('removeEdge on unknown id is a no-op', () => {
    expect(() => state.removeEdge('nonexistent')).not.toThrow();
  });

  // --- getSelectedNodeId / setSelectedNodeId ---

  it('getSelectedNodeId returns null initially', () => {
    expect(state.getSelectedNodeId()).toBeNull();
  });

  it('setSelectedNodeId updates selection', () => {
    const id = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    state.setSelectedNodeId(id);
    expect(state.getSelectedNodeId()).toBe(id);
  });

  it('setSelectedNodeId null clears selection', () => {
    const id = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    state.setSelectedNodeId(id);
    state.setSelectedNodeId(null);
    expect(state.getSelectedNodeId()).toBeNull();
  });

  it('removeNode clears selection if removed node was selected', () => {
    const id = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    state.setSelectedNodeId(id);
    state.removeNode(id);
    expect(state.getSelectedNodeId()).toBeNull();
  });

  // --- updateNodePosition ---

  it('updateNodePosition updates position in toPayload', () => {
    const id = state.addNode('action', 'Notify', { x: 0, y: 0 });
    state.updateNodePosition(id, { x: 300, y: 150 });
    const node = state.toPayload('wf').graph.nodes.find((n) => n.id === id);
    expect(node?.position).toEqual({ x: 300, y: 150 });
  });

  it('updateNodePosition on unknown id is a no-op', () => {
    expect(() => state.updateNodePosition('nonexistent', { x: 0, y: 0 })).not.toThrow();
  });

  // --- loadFromProjection ---

  it('loadFromProjection replaces state with projected nodes and edges', () => {
    state.addNode('workflow', 'old', { x: 0, y: 0 });
    state.loadFromProjection({
      nodes: [
        { id: 'n1', kind: 'workflow', label: 'new_wf', color: '#fff', position: { x: 10, y: 20 } },
        { id: 'n2', kind: 'action', label: 'notify', color: '#fff', position: { x: 200, y: 20 } },
      ],
      edges: [{ id: 'e1', from: 'n1', to: 'n2', connection_type: 'sequence' }],
    });
    const { graph } = state.toPayload('wf');
    expect(graph.nodes).toHaveLength(2);
    expect(graph.nodes[0]?.id).toBe('n1');
    expect(graph.edges).toHaveLength(1);
    expect(graph.edges[0]?.id).toBe('e1');
  });

  it('loadFromProjection resets selection', () => {
    const id = state.addNode('workflow', 'wf', { x: 0, y: 0 });
    state.setSelectedNodeId(id);
    state.loadFromProjection({ nodes: [], edges: [] });
    expect(state.getSelectedNodeId()).toBeNull();
  });

  // --- SUPPORTED_KINDS ---

  it('SUPPORTED_KINDS contains all 9 allowed kinds', () => {
    expect(SUPPORTED_KINDS).toHaveLength(9);
    expect(SUPPORTED_KINDS).toContain('workflow');
    expect(SUPPORTED_KINDS).toContain('trigger');
    expect(SUPPORTED_KINDS).toContain('action');
    expect(SUPPORTED_KINDS).toContain('decision');
    expect(SUPPORTED_KINDS).toContain('grounds');
    expect(SUPPORTED_KINDS).toContain('permit');
    expect(SUPPORTED_KINDS).toContain('delegate');
    expect(SUPPORTED_KINDS).toContain('invariant');
    expect(SUPPORTED_KINDS).toContain('budget');
  });
});
