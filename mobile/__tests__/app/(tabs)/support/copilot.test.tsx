// Support copilot route tests — W3-T4
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'case-1', entity_type: 'case', entity_id: 'case-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

jest.mock('../../../../src/components/copilot', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    CopilotPanel: ({ testIDPrefix, entityType, entityId }: { testIDPrefix: string; entityType: string; entityId: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-panel`, accessibilityLabel: `${entityType}:${entityId}` }),
  };
});

jest.mock('react-native-paper', () => ({
  useTheme: () => ({ colors: { background: '#fff', primary: '#E53935' } }),
}));

describe('Support copilot route', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders the copilot panel', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]/copilot');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-copilot-panel')).toBeTruthy();
  });

  it('passes entity context to CopilotPanel', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]/copilot');
    render(React.createElement(Screen));
    const panel = screen.getByTestId('support-copilot-panel');
    expect(panel.props.accessibilityLabel).toBe('case:case-1');
  });
});
