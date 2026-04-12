import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

const mockBack = jest.fn();
const mockPush = jest.fn();
const mockUseSignalsByEntity = jest.fn();
const mockUseDismissSignal = jest.fn();
const mockSignalDetailView = jest.fn();

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, back: mockBack }),
  useLocalSearchParams: () => ({ id: 'sig-1', entity_type: 'deal', entity_id: 'deal-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

jest.mock('../../../../../src/hooks/useAgentSpec', () => ({
  useSignalsByEntity: (...args: unknown[]) => mockUseSignalsByEntity(...args),
  useDismissSignal: () => mockUseDismissSignal(),
}));

jest.mock('../../../../../src/components/signals/SignalDetailView', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
  return {
    SignalDetailView: ({ signal, testIDPrefix }: { signal: { id: string; signal_type: string }; testIDPrefix: string }) => {
      mockSignalDetailView({ signal, testIDPrefix });
      return React.createElement(
        View,
        { testID: testIDPrefix },
        React.createElement(Text, { testID: `${testIDPrefix}-type` }, signal.signal_type),
      );
    },
  };
});

const signal = {
  id: 'sig-1',
  workspace_id: 'ws-1',
  entity_type: 'deal',
  entity_id: 'deal-1',
  signal_type: 'churn_risk',
  confidence: 0.9,
  evidence_ids: [],
  source_type: 'agent',
  source_id: 'src-1',
  metadata: {},
  status: 'active',
  created_at: '2026-04-08T10:00:00Z',
  updated_at: '2026-04-08T10:00:00Z',
};

function renderScreen() {
  const { default: Screen } = require('../../../../../app/(tabs)/home/signal/[id]');
  render(React.createElement(Screen));
}

describe('SignalDetailScreen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseSignalsByEntity.mockReturnValue({ data: [signal], isLoading: false, error: null });
    mockUseDismissSignal.mockReturnValue({ mutate: jest.fn(), isPending: false });
  });

  it('renders the selected signal detail', () => {
    renderScreen();
    expect(screen.getByTestId('signal-detail')).toBeTruthy();
    expect(screen.getByTestId('signal-detail-type').props.children).toBe('churn_risk');
    expect(mockSignalDetailView).toHaveBeenCalledWith({
      signal: expect.objectContaining({ id: 'sig-1' }),
      testIDPrefix: 'signal-detail',
    });
  });

  it('shows loading state while signals are fetching', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: undefined, isLoading: true, error: null });
    renderScreen();
    expect(screen.queryByTestId('signal-detail')).toBeNull();
  });

  it('shows not found when the requested signal is missing', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [], isLoading: false, error: null });
    renderScreen();
    expect(screen.getByText('Signal not found')).toBeTruthy();
  });

  it('does not crash when the hook returns undefined data', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: undefined, isLoading: false, error: null });
    renderScreen();
    expect(screen.getByText('Signal not found')).toBeTruthy();
  });
});
