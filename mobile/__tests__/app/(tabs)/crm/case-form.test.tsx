import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';
import CRMCaseNewScreen from '../../../../app/(tabs)/crm/cases/new';
import CRMCaseEditScreen from '../../../../app/(tabs)/crm/cases/edit/[id]';

const mockReplace = jest.fn();
const mockCreateMutateAsync = jest.fn();
const mockUpdateMutateAsync = jest.fn();
let mockCaseData: unknown = null;
let mockUserId: string | null = 'user-1';

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
  useLocalSearchParams: () => ({ id: 'case-1' }),
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
  useCase: () => ({ data: mockCaseData, isLoading: false, error: null }),
  useCreateCase: () => ({ mutateAsync: mockCreateMutateAsync, isPending: false }),
  useUpdateCase: () => ({ mutateAsync: mockUpdateMutateAsync, isPending: false }),
}));

describe('CRM case forms', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockCaseData = null;
    mockUserId = 'user-1';
    mockCreateMutateAsync.mockResolvedValue({});
    mockUpdateMutateAsync.mockResolvedValue({});
  });

  it('validates required subject before create submit', async () => {
    render(<CRMCaseNewScreen />);
    fireEvent.press(screen.getByTestId('crm-case-form-submit'));

    expect(await screen.findByText('Case subject is required')).toBeTruthy();
    expect(mockCreateMutateAsync).not.toHaveBeenCalled();
  });

  it('creates a standalone case with the signed-in user as owner', async () => {
    render(<CRMCaseNewScreen />);
    fireEvent.changeText(screen.getByTestId('crm-case-form-subject'), 'Cannot login');
    fireEvent.changeText(screen.getByTestId('crm-case-form-channel'), 'email');
    fireEvent.press(screen.getByTestId('crm-case-form-submit'));

    await waitFor(() => expect(mockCreateMutateAsync).toHaveBeenCalledWith({
      ownerId: 'user-1',
      subject: 'Cannot login',
      priority: 'medium',
      status: 'open',
      channel: 'email',
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/cases');
  });

  it('creates a linked case when account and contact are selected', async () => {
    render(<CRMCaseNewScreen />);
    fireEvent.changeText(screen.getByTestId('crm-case-form-subject'), 'Billing issue');
    fireEvent.press(screen.getByTestId('crm-case-form-account-acc-1'));
    fireEvent.press(screen.getByTestId('crm-case-form-contact-contact-1'));
    fireEvent.press(screen.getByTestId('crm-case-form-submit'));

    await waitFor(() => expect(mockCreateMutateAsync).toHaveBeenCalledWith({
      ownerId: 'user-1',
      accountId: 'acc-1',
      contactId: 'contact-1',
      subject: 'Billing issue',
      priority: 'medium',
      status: 'open',
    }));
  });

  it('loads existing case data and updates the CRM case detail', async () => {
    mockCaseData = {
      case: {
        id: 'case-1',
        accountId: 'acc-1',
        contactId: 'contact-1',
        subject: 'Cannot login',
        priority: 'high',
        status: 'open',
        channel: 'email',
      },
    };

    render(<CRMCaseEditScreen />);
    expect(screen.getByDisplayValue('Cannot login')).toBeTruthy();
    fireEvent.changeText(screen.getByTestId('crm-case-form-status'), 'resolved');
    fireEvent.press(screen.getByTestId('crm-case-form-submit'));

    await waitFor(() => expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
      id: 'case-1',
      data: {
        ownerId: 'user-1',
        accountId: 'acc-1',
        contactId: 'contact-1',
        subject: 'Cannot login',
        priority: 'high',
        status: 'resolved',
        channel: 'email',
      },
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/cases/case-1');
  });
});
