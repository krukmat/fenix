// CLSF-82: mobile read-only graph components
import React from 'react';
import { describe, it, expect } from '@jest/globals';
import { render } from '@testing-library/react-native';

import { FlowCanvas } from '../../../src/components/workflows/FlowCanvas';
import { type FlowLayoutResult } from '../../../src/lib/flowLayout';

const fixture: FlowLayoutResult = {
  nodes: [
    { id: 'workflow-1', kind: 'workflow', label: 'sales_followup', x: 24, y: 24, width: 200, height: 72 },
    { id: 'trigger-1', kind: 'trigger', label: 'deal.updated', x: 288, y: 24, width: 200, height: 72 },
    { id: 'action-1', kind: 'action', label: 'notify owner', x: 552, y: 24, width: 200, height: 72 },
    { id: 'grounds-1', kind: 'grounds', label: 'permit + grounds', x: 816, y: 24, width: 200, height: 72 },
  ],
  connectors: [
    {
      id: 'edge-wf-trigger',
      from: 'workflow-1',
      to: 'trigger-1',
      start: { x: 224, y: 60 },
      end: { x: 288, y: 60 },
      connectionType: 'execution',
    },
    {
      id: 'edge-trigger-grounds',
      from: 'trigger-1',
      to: 'grounds-1',
      start: { x: 488, y: 60 },
      end: { x: 816, y: 60 },
      connectionType: 'requirement',
    },
  ],
  bounds: { width: 1040, height: 120 },
};

describe('FlowCanvas', () => {
  it('renders all node labels', () => {
    const { getByTestId, getByText } = render(<FlowCanvas layout={fixture} />);

    expect(getByTestId('flow-canvas')).toBeTruthy();
    expect(getByText('sales_followup')).toBeTruthy();
    expect(getByText('deal.updated')).toBeTruthy();
    expect(getByText('notify owner')).toBeTruthy();
    expect(getByText('permit + grounds')).toBeTruthy();
  });

  it('renders a node box per layout node with testID', () => {
    const { getByTestId } = render(<FlowCanvas layout={fixture} />);

    expect(getByTestId('flow-node-workflow-1')).toBeTruthy();
    expect(getByTestId('flow-node-trigger-1')).toBeTruthy();
    expect(getByTestId('flow-node-action-1')).toBeTruthy();
    expect(getByTestId('flow-node-grounds-1')).toBeTruthy();
  });

  it('renders a connector per layout connector with testID', () => {
    const { getByTestId } = render(<FlowCanvas layout={fixture} />);

    expect(getByTestId('flow-connector-edge-wf-trigger')).toBeTruthy();
    expect(getByTestId('flow-connector-edge-trigger-grounds')).toBeTruthy();
  });

  it('renders empty canvas without errors when layout has no nodes or connectors', () => {
    const empty: FlowLayoutResult = { nodes: [], connectors: [], bounds: { width: 48, height: 48 } };
    const { queryByTestId } = render(<FlowCanvas layout={empty} />);

    expect(queryByTestId('flow-node-any')).toBeNull();
    expect(queryByTestId('flow-connector-any')).toBeNull();
  });

  it('shows kind label on each node', () => {
    const { getAllByText } = render(<FlowCanvas layout={fixture} />);
    expect(getAllByText('workflow').length).toBeGreaterThanOrEqual(1);
    expect(getAllByText('trigger').length).toBeGreaterThanOrEqual(1);
  });

  // WFG-T2: scroll content sizing fix
  it('canvas View receives explicit width and height from layout.bounds', () => {
    const { getByTestId } = render(<FlowCanvas layout={fixture} />);
    const canvas = getByTestId('flow-canvas');
    const flatStyle = Array.isArray(canvas.props.style)
      ? Object.assign({}, ...canvas.props.style)
      : canvas.props.style;
    expect(flatStyle.width).toBe(fixture.bounds.width);
    expect(flatStyle.height).toBe(fixture.bounds.height);
  });

  it('inner ScrollView has nestedScrollEnabled prop set to true', () => {
    const { UNSAFE_getAllByType } = render(<FlowCanvas layout={fixture} />);
    const scrollViews = UNSAFE_getAllByType(require('react-native').ScrollView);
    // inner ScrollView is index 1 (outer is horizontal, inner is vertical)
    const inner = scrollViews[1];
    expect(inner.props.nestedScrollEnabled).toBe(true);
  });

  it('inner ScrollView contentContainerStyle does not set flexGrow', () => {
    const { UNSAFE_getAllByType } = render(<FlowCanvas layout={fixture} />);
    const scrollViews = UNSAFE_getAllByType(require('react-native').ScrollView);
    const inner = scrollViews[1];
    const ccs = inner.props.contentContainerStyle ?? {};
    const flat = Array.isArray(ccs) ? Object.assign({}, ...ccs) : ccs;
    expect(flat.flexGrow).toBeUndefined();
  });
});
