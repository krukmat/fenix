// Sales copilot route tests — W4-T4
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'acc-1', entity_type: 'account', entity_id: 'acc-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

jest.mock('../../../../src/components/copilot', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    CopilotPanel: ({ initialContext }: { initialContext?: { entityType?: string; entityId?: string } }) =>
      React.createElement(View, {
        testID: 'sales-copilot-panel',
        accessibilityLabel: `${initialContext?.entityType ?? ''}:${initialContext?.entityId ?? ''}`,
      }),
  };
});

jest.mock('react-native-paper', () => ({
  useTheme: () => ({ colors: { background: '#fff', primary: '#E53935' } }),
}));

describe('Sales copilot route', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders the copilot panel', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/copilot');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-copilot-panel')).toBeTruthy();
  });

  it('passes entity context to CopilotPanel', () => {
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/copilot');
    render(React.createElement(Screen));
    const panel = screen.getByTestId('sales-copilot-panel');
    expect(panel.props.accessibilityLabel).toBe('account:acc-1');
  });
});
