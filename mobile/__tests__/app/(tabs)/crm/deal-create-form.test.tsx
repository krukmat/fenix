import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';
import CRMDealEditScreen from '../../../../app/(tabs)/crm/deals/edit/[id]';
import CRMDealNewScreen from '../../../../app/(tabs)/crm/deals/new';

const mockReplace = jest.fn();
const mockCreateMutateAsync = jest.fn();
const mockUpdateMutateAsync = jest.fn();
let mockUserId: string | null = 'user-1';
let mockDealData: unknown = null;

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

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ replace: mockReplace }),
  useLocalSearchParams: () => ({ id: 'deal-1' }),
}));

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

jest.mock('../../../../src/stores/authStore', () => ({
  useAuthStore: (selector: (state: { userId: string | null }) => unknown) => selector({ userId: mockUserId }),
}));

jest.mock('../../../../src/hooks/useCRM', () => ({
  useAccounts: () => listQuery([{ id: 'acc-1', name: 'Acme' }]),
  useContacts: () => listQuery([{ id: 'contact-1', accountId: 'acc-1', firstName: 'Ada', lastName: 'Lovelace' }]),
  usePipelines: () => listQuery([{ id: 'pipe-1', name: 'Sales', entityType: 'deal' }]),
  usePipelineStages: (pipelineId: string) => ({
    data: pipelineId ? { data: [{ id: 'stage-1', pipelineId, name: 'Qualified' }] } : { data: [] },
    isLoading: false,
  }),
  useDeal: () => ({ data: mockDealData, isLoading: false, error: null }),
  useCreateDeal: () => ({ mutateAsync: mockCreateMutateAsync, isPending: false }),
  useUpdateDeal: () => ({ mutateAsync: mockUpdateMutateAsync, isPending: false }),
}));

describe('CRM deal create form', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUserId = 'user-1';
    mockDealData = null;
    mockCreateMutateAsync.mockResolvedValue({});
    mockUpdateMutateAsync.mockResolvedValue({});
  });

  it('validates required deal title before selector validation', async () => {
    render(<CRMDealNewScreen />);
    fireEvent.press(screen.getByTestId('crm-deal-form-submit'));

    expect(await screen.findByText('Deal title is required')).toBeTruthy();
    expect(mockCreateMutateAsync).not.toHaveBeenCalled();
  });

  it('validates required selectors after title is present', async () => {
    render(<CRMDealNewScreen />);
    fireEvent.changeText(screen.getByTestId('crm-deal-form-title'), 'Expansion');
    fireEvent.press(screen.getByTestId('crm-deal-form-submit'));

    expect(await screen.findByText('Account is required')).toBeTruthy();
    expect(mockCreateMutateAsync).not.toHaveBeenCalled();
  });

  it('creates a deal and returns to the CRM deal list', async () => {
    render(<CRMDealNewScreen />);
    fireEvent.changeText(screen.getByTestId('crm-deal-form-title'), 'Expansion');
    fireEvent.changeText(screen.getByTestId('crm-deal-form-amount'), '12000');
    fireEvent.press(screen.getByTestId('crm-deal-selector-account-acc-1'));
    fireEvent.press(screen.getByTestId('crm-deal-selector-contact-contact-1'));
    fireEvent.press(screen.getByTestId('crm-deal-selector-pipeline-pipe-1'));
    fireEvent.press(screen.getByTestId('crm-deal-selector-stage-stage-1'));
    fireEvent.press(screen.getByTestId('crm-deal-form-submit'));

    await waitFor(() => expect(mockCreateMutateAsync).toHaveBeenCalledWith({
      ownerId: 'user-1',
      accountId: 'acc-1',
      contactId: 'contact-1',
      pipelineId: 'pipe-1',
      stageId: 'stage-1',
      title: 'Expansion',
      amount: 12000,
      currency: 'USD',
      status: 'open',
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/deals');
  });

  it('loads existing deal data and updates the CRM deal detail', async () => {
    mockDealData = {
      deal: {
        id: 'deal-1',
        ownerId: 'user-1',
        accountId: 'acc-1',
        contactId: 'contact-1',
        pipelineId: 'pipe-1',
        stageId: 'stage-1',
        title: 'Expansion',
        amount: 12000,
        currency: 'USD',
        status: 'open',
      },
    };

    render(<CRMDealEditScreen />);
    expect(screen.getByDisplayValue('Expansion')).toBeTruthy();
    fireEvent.changeText(screen.getByTestId('crm-deal-form-status'), 'won');
    fireEvent.press(screen.getByTestId('crm-deal-form-submit'));

    await waitFor(() => expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
      id: 'deal-1',
      data: {
        ownerId: 'user-1',
        accountId: 'acc-1',
        contactId: 'contact-1',
        pipelineId: 'pipe-1',
        stageId: 'stage-1',
        title: 'Expansion',
        amount: 12000,
        currency: 'USD',
        status: 'won',
      },
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/deals/deal-1');
  });
});
