// CLSF-83: mobile workflow graph review screen tests
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen, waitFor } from '@testing-library/react-native';

// ─── Mocks ────────────────────────────────────────────────────────────────────

jest.mock('expo-router', () => ({
  __esModule: true,
  useLocalSearchParams: () => ({ id: 'wf-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockGetGraph = jest.fn();
jest.mock('../../../../src/services/api.workflows', () => ({
  workflowApi: { getGraph: (...args: unknown[]) => mockGetGraph(...args) },
}));

jest.mock('../../../../src/components/workflows/FlowCanvas', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
  return {
    FlowCanvas: ({ layout }: { layout: { nodes: unknown[] } }) =>
      React.createElement(
        View,
        { testID: 'flow-canvas' },
        React.createElement(Text, null, `nodes:${layout.nodes.length}`),
      ),
  };
});

jest.mock('react-native-paper', () => {
  const React = require('react');
  const { View, Text, ActivityIndicator } = require('react-native');
  return {
    useTheme: () => ({ colors: { primary: '#E53935', error: '#B00020', surface: '#fff', onSurface: '#000', onSurfaceVariant: '#666', background: '#fff' } }),
    ActivityIndicator: ({ testID }: { testID?: string }) =>
      React.createElement(ActivityIndicator, { testID }),
    Text: ({ children, testID }: { children: unknown; testID?: string }) =>
      React.createElement(Text, { testID }, children),
    Chip: ({ children }: { children: unknown }) =>
      React.createElement(View, null, React.createElement(Text, null, children)),
  };
});

// ─── Fixture ──────────────────────────────────────────────────────────────────

const graphFixture = {
  workflow_id: 'wf-1',
  conformance: { profile: 'safe', details: [] },
  nodes: [
    { id: 'workflow-1', kind: 'workflow', label: 'sales_followup' },
    { id: 'trigger-1', kind: 'trigger', label: 'deal.updated' },
  ],
  edges: [
    { id: 'e1', from: 'workflow-1', to: 'trigger-1', connection_type: 'execution' },
  ],
};

// ─── Tests ────────────────────────────────────────────────────────────────────

import WorkflowGraphScreen from '../../../../app/(tabs)/workflows/graph';

describe('WorkflowGraphScreen', () => {
  beforeEach(() => {
    mockGetGraph.mockReset();
  });

  it('shows loading indicator while fetching', () => {
    mockGetGraph.mockReturnValue(new Promise(() => {}));
    render(<WorkflowGraphScreen />);
    expect(screen.getByTestId('graph-loading')).toBeTruthy();
  });

  it('renders FlowCanvas with nodes after successful fetch', async () => {
    mockGetGraph.mockResolvedValue(graphFixture);
    render(<WorkflowGraphScreen />);
    await waitFor(() => expect(screen.getByTestId('flow-canvas')).toBeTruthy());
    expect(screen.getByText('nodes:2')).toBeTruthy();
  });

  it('shows error message when fetch fails', async () => {
    mockGetGraph.mockRejectedValue(new Error('network error'));
    render(<WorkflowGraphScreen />);
    await waitFor(() => expect(screen.getByTestId('graph-error')).toBeTruthy());
  });

  it('shows empty state when graph has no nodes', async () => {
    mockGetGraph.mockResolvedValue({ ...graphFixture, nodes: [], edges: [] });
    render(<WorkflowGraphScreen />);
    await waitFor(() => expect(screen.getByTestId('graph-empty')).toBeTruthy());
  });

  it('shows conformance profile chip', async () => {
    mockGetGraph.mockResolvedValue(graphFixture);
    render(<WorkflowGraphScreen />);
    await waitFor(() => expect(screen.getByText('safe')).toBeTruthy());
  });
});
