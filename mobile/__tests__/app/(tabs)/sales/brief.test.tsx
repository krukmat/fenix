// Sales brief route tests — W4-T3
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'acc-1', entity_type: 'account', entity_id: 'acc-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseSalesBrief = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useSalesBrief: (...args: unknown[]) => mockUseSalesBrief(...args),
}));

jest.mock('react-native-paper', () => {
  const mockReact = require('react');
  const { View } = require('react-native');
  return {
    useTheme: () => ({
      colors: { background: '#fff', primary: '#E53935', onSurface: '#000', onSurfaceVariant: '#666', surface: '#fff', error: '#B00020' },
    }),
    ActivityIndicator: ({ testID }: { testID: string }) => mockReact.createElement(View, { testID }),
  };
});

describe('Sales brief route', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders the brief screen', () => {
    mockUseSalesBrief.mockReturnValue({ data: { summary: 'Strong Q1 pipeline', recommendations: [] }, isLoading: false, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-screen')).toBeTruthy();
  });

  it('shows loading indicator while fetching', () => {
    mockUseSalesBrief.mockReturnValue({ data: undefined, isLoading: true, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-loading')).toBeTruthy();
  });

  it('shows summary text when brief is loaded', () => {
    mockUseSalesBrief.mockReturnValue({
      data: { summary: 'Strong Q1 pipeline', recommendations: ['Follow up on Acme'] },
      isLoading: false,
      error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-summary')).toBeTruthy();
  });

  it('shows error state when brief fails', () => {
    mockUseSalesBrief.mockReturnValue({ data: null, isLoading: false, error: new Error('Failed') });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-error')).toBeTruthy();
  });

  it('calls useSalesBrief with entity_type and entity_id from params', () => {
    mockUseSalesBrief.mockReturnValue({ data: null, isLoading: false, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(mockUseSalesBrief).toHaveBeenCalledWith('account', 'acc-1', true);
  });
});
