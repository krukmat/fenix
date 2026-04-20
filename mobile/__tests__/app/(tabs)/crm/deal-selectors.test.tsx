import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react-native';
import {
  CRMDealSelectors,
  emptyDealSelectorValues,
  validateDealSelectors,
  type CRMDealSelectorValues,
} from '../../../../src/components/crm/CRMDealSelectors';

const mockOnChange = jest.fn();
let selectorValues: CRMDealSelectorValues = emptyDealSelectorValues;
let pipelineIdForStages = '';

const listQuery = (items: unknown[]) => ({
  data: { pages: [{ data: items, total: items.length }] },
  isLoading: false,
  isFetchingNextPage: false,
  hasNextPage: false,
  fetchNextPage: jest.fn(),
  error: null,
  refetch: jest.fn(),
  isRefetching: false,
});

jest.mock('react-native-paper', () => ({
  useTheme: () => ({
    colors: {
      primary: '#E53935',
      onPrimary: '#FFFFFF',
      surface: '#FFFFFF',
      surfaceVariant: '#EEF2F7',
      onSurface: '#111827',
      onSurfaceVariant: '#6B7280',
      background: '#FFFFFF',
      outline: '#CBD5E1',
      error: '#B00020',
    },
  }),
}));

jest.mock('../../../../src/hooks/useCRM', () => ({
  useAccounts: () => listQuery([{ id: 'acc-1', name: 'Acme' }]),
  useContacts: () => listQuery([{ id: 'contact-1', accountId: 'acc-1', firstName: 'Ada', lastName: 'Lovelace' }]),
  usePipelines: () => listQuery([{ id: 'pipe-1', name: 'Sales', entityType: 'deal' }]),
  usePipelineStages: (pipelineId: string) => {
    pipelineIdForStages = pipelineId;
    return {
      data: pipelineId ? { data: [{ id: 'stage-1', pipelineId, name: 'Qualified' }] } : { data: [] },
      isLoading: false,
    };
  },
}));

describe('CRM deal selector foundation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    pipelineIdForStages = '';
    selectorValues = { ...emptyDealSelectorValues, pipelineId: 'pipe-1' };
  });

  it('renders Account, optional Contact, Pipeline, and Stage selectors', () => {
    render(<CRMDealSelectors values={selectorValues} onChange={mockOnChange} />);

    expect(screen.getByTestId('crm-deal-selectors')).toBeTruthy();
    expect(screen.getByText('Acme')).toBeTruthy();
    expect(screen.getByText('Ada Lovelace')).toBeTruthy();
    expect(screen.getByText('Sales')).toBeTruthy();
    expect(screen.getByText('Qualified')).toBeTruthy();
    expect(pipelineIdForStages).toBe('pipe-1');
  });

  it('emits field changes for every selector group', () => {
    render(<CRMDealSelectors values={selectorValues} onChange={mockOnChange} />);

    fireEvent.press(screen.getByTestId('crm-deal-selector-account-acc-1'));
    fireEvent.press(screen.getByTestId('crm-deal-selector-contact-contact-1'));
    fireEvent.press(screen.getByTestId('crm-deal-selector-contact-none'));
    fireEvent.press(screen.getByTestId('crm-deal-selector-pipeline-pipe-1'));
    fireEvent.press(screen.getByTestId('crm-deal-selector-stage-stage-1'));

    expect(mockOnChange).toHaveBeenNthCalledWith(1, 'accountId', 'acc-1');
    expect(mockOnChange).toHaveBeenNthCalledWith(2, 'contactId', 'contact-1');
    expect(mockOnChange).toHaveBeenNthCalledWith(3, 'contactId', '');
    expect(mockOnChange).toHaveBeenNthCalledWith(4, 'pipelineId', 'pipe-1');
    expect(mockOnChange).toHaveBeenNthCalledWith(5, 'stageId', 'stage-1');
  });

  it('validates required selectors and relationship mismatches', () => {
    expect(validateDealSelectors(emptyDealSelectorValues, [], [])).toBe('Account is required');
    expect(validateDealSelectors({ ...emptyDealSelectorValues, accountId: 'acc-1' }, [], [])).toBe('Pipeline is required');
    expect(validateDealSelectors({ ...emptyDealSelectorValues, accountId: 'acc-1', pipelineId: 'pipe-1' }, [], [])).toBe('Stage is required');
    expect(validateDealSelectors(
      { accountId: 'acc-2', contactId: 'contact-1', pipelineId: 'pipe-1', stageId: 'stage-1' },
      [{ id: 'contact-1', accountId: 'acc-1', activeSignalCount: 0 }],
      [{ id: 'stage-1', pipelineId: 'pipe-1', name: 'Qualified' }],
    )).toBe('Selected contact belongs to another account');
    expect(validateDealSelectors(
      { accountId: 'acc-1', contactId: '', pipelineId: 'pipe-2', stageId: 'stage-1' },
      [],
      [{ id: 'stage-1', pipelineId: 'pipe-1', name: 'Qualified' }],
    )).toBe('Selected stage belongs to another pipeline');
  });
});
