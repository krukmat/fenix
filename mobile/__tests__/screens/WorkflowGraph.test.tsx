// WFG-T3: graph screen header chips layout tests
import React from 'react';
import { View, ScrollView } from 'react-native';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render, waitFor } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';

import WorkflowGraphScreen from '../../app/(tabs)/workflows/graph';

const mockGetGraph = jest.fn();

jest.mock('../../src/services/api.workflows', () => ({
  workflowApi: {
    getGraph: (...args: unknown[]) => mockGetGraph(...args),
  },
}));

jest.mock('expo-router', () => ({
  useLocalSearchParams: () => ({ id: 'wf-graph-1' }),
  Stack: {
    Screen: ({ children }: { children?: React.ReactNode }) => children ?? null,
  },
}));

jest.mock('../../src/lib/flowLayout', () => ({
  layoutWorkflowGraph: () => ({
    nodes: [{ id: 'n1', kind: 'workflow', label: 'test', x: 0, y: 0, width: 100, height: 50 }],
    connectors: [],
    bounds: { width: 200, height: 100 },
  }),
}));

const mockGraphResponse = {
  nodes: [{ id: 'n1', kind: 'workflow', label: 'test', x: 0, y: 0 }],
  edges: [],
  conformance: {
    profile: 'safe',
    details: [{ code: 'C1' }, { code: 'C2' }],
  },
};

function renderScreen() {
  return render(
    <PaperProvider>
      <WorkflowGraphScreen />
    </PaperProvider>
  );
}

describe('WorkflowGraphScreen — conformance header', () => {
  beforeEach(() => {
    mockGetGraph.mockResolvedValue(mockGraphResponse);
  });

  it('renders the conformance profile chip', async () => {
    const { getByText } = renderScreen();
    await waitFor(() => getByText('safe'));
    expect(getByText('safe')).toBeTruthy();
  });

  it('renders one chip per conformance detail code', async () => {
    const { getByText } = renderScreen();
    await waitFor(() => getByText('C1'));
    expect(getByText('C1')).toBeTruthy();
    expect(getByText('C2')).toBeTruthy();
  });

  it('header container is a View, not a ScrollView', async () => {
    const { getByTestId, UNSAFE_getAllByType } = renderScreen();
    await waitFor(() => getByTestId('graph-conformance-header'));
    const header = getByTestId('graph-conformance-header');
    // The element must be a View instance, not a ScrollView
    const scrollViews = UNSAFE_getAllByType(ScrollView);
    const headerIsScrollView = scrollViews.some(
      (sv) => sv.props.testID === 'graph-conformance-header'
    );
    expect(headerIsScrollView).toBe(false);
    expect(header).toBeTruthy();
  });
});
