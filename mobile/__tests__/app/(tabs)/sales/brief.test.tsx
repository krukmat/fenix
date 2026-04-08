// Sales brief route tests — W4-T3
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';
import type { SalesBrief } from '../../../../src/services/api';

const mockUseLocalSearchParams = jest.fn();

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: jest.fn(), replace: jest.fn() }),
  useLocalSearchParams: () => mockUseLocalSearchParams(),
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
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseLocalSearchParams.mockReturnValue({ id: 'acc-1', entity_type: 'account', entity_id: 'acc-1' });
  });

  const completedBrief: SalesBrief = {
    outcome: 'completed',
    entityType: 'account',
    entityId: 'acc-1',
    summary: 'Strong Q1 pipeline',
    risks: ['Renewal timing is slipping'],
    nextBestActions: ['Follow up on Acme'],
    confidence: 'high',
    evidencePack: {
      schema_version: 'v1',
      query: 'account brief',
      sources: [],
      source_count: 0,
      dedup_count: 0,
      filtered_count: 0,
      confidence: 'high',
      warnings: ['Pipeline data is 3 days old'],
      retrieval_methods_used: ['crm'],
      built_at: '2026-04-08T10:00:00Z',
    },
  };

  const abstainedBrief: SalesBrief = {
    outcome: 'abstained',
    entityType: 'account',
    entityId: 'acc-1',
    abstentionReason: 'Insufficient evidence',
    confidence: 'low',
    evidencePack: {
      schema_version: 'v1',
      query: 'account brief',
      sources: [],
      source_count: 0,
      dedup_count: 0,
      filtered_count: 0,
      confidence: 'low',
      warnings: ['No recent activity'],
      retrieval_methods_used: ['crm'],
      built_at: '2026-04-08T10:00:00Z',
    },
  };

  it('renders the brief screen', () => {
    mockUseSalesBrief.mockReturnValue({ data: completedBrief, isLoading: false, error: null });
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

  it('shows canonical completed brief fields when loaded', () => {
    mockUseSalesBrief.mockReturnValue({
      data: completedBrief,
      isLoading: false,
      error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-outcome')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-confidence')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-summary')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-risks')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-next-best-actions')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-evidence-pack')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-evidence-query')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-evidence-methods')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-evidence-warnings')).toBeTruthy();
    expect(screen.getByText('completed')).toBeTruthy();
    expect(screen.getByText('high')).toBeTruthy();
    expect(screen.getByText('Strong Q1 pipeline')).toBeTruthy();
    expect(screen.getByText('Renewal timing is slipping')).toBeTruthy();
    expect(screen.getByText('Follow up on Acme')).toBeTruthy();
    expect(screen.getByText('0 sources · high confidence')).toBeTruthy();
    expect(screen.getByText('Query: account brief')).toBeTruthy();
    expect(screen.getByText('Methods: crm')).toBeTruthy();
    expect(screen.getByText('Pipeline data is 3 days old')).toBeTruthy();
  });

  it('shows canonical abstained brief fields when loaded', () => {
    mockUseSalesBrief.mockReturnValue({
      data: abstainedBrief,
      isLoading: false,
      error: null,
    });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-outcome')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-confidence')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-abstention-reason')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-evidence-pack')).toBeTruthy();
    expect(screen.queryByTestId('sales-brief-summary')).toBeNull();
    expect(screen.queryByTestId('sales-brief-risks')).toBeNull();
    expect(screen.queryByTestId('sales-brief-next-best-actions')).toBeNull();
    expect(screen.getByTestId('sales-brief-evidence-methods')).toBeTruthy();
    expect(screen.getByText('abstained')).toBeTruthy();
    expect(screen.getByText('low')).toBeTruthy();
    expect(screen.getByText('Insufficient evidence')).toBeTruthy();
    expect(screen.getByText('Methods: crm')).toBeTruthy();
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

  it('supports deal entry by deriving the canonical entity id from the route slug', () => {
    mockUseLocalSearchParams.mockReturnValue({ id: 'deal-deal-1', entity_type: 'deal' });
    mockUseSalesBrief.mockReturnValue({ data: completedBrief, isLoading: false, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(mockUseSalesBrief).toHaveBeenCalledWith('deal', 'deal-1', true);
  });

  it('fails if confidence disappears from the visible brief contract', () => {
    mockUseSalesBrief.mockReturnValue({ data: completedBrief, isLoading: false, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-confidence')).toBeTruthy();
    expect(screen.getByText('high')).toBeTruthy();
  });

  it('fails if evidence disappears from the visible brief contract', () => {
    mockUseSalesBrief.mockReturnValue({ data: completedBrief, isLoading: false, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/sales/[id]/brief');
    render(React.createElement(Screen));
    expect(screen.getByTestId('sales-brief-evidence-pack')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-evidence-summary')).toBeTruthy();
    expect(screen.getByTestId('sales-brief-evidence-query')).toBeTruthy();
  });
});
