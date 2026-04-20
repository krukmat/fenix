import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';
import CRMContactNewScreen from '../../../../app/(tabs)/crm/contacts/new';
import CRMContactEditScreen from '../../../../app/(tabs)/crm/contacts/edit/[id]';

const mockReplace = jest.fn();
const mockCreateMutateAsync = jest.fn();
const mockUpdateMutateAsync = jest.fn();
let mockContactData: unknown = null;

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
  useLocalSearchParams: () => ({ id: 'contact-1' }),
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

jest.mock('../../../../src/hooks/useCRM', () => ({
  useAccounts: () => listQuery([{ id: 'acc-1', name: 'Acme' }]),
  useContact: () => ({ data: mockContactData, isLoading: false, error: null }),
  useCreateContact: () => ({ mutateAsync: mockCreateMutateAsync, isPending: false }),
  useUpdateContact: () => ({ mutateAsync: mockUpdateMutateAsync, isPending: false }),
}));

describe('CRM contact forms', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockContactData = null;
    mockCreateMutateAsync.mockResolvedValue({});
    mockUpdateMutateAsync.mockResolvedValue({});
  });

  it('validates required account before create submit', async () => {
    render(<CRMContactNewScreen />);
    fireEvent.changeText(screen.getByTestId('crm-contact-form-email'), 'ada@example.test');
    fireEvent.press(screen.getByTestId('crm-contact-form-submit'));

    expect(await screen.findByText('Account is required')).toBeTruthy();
    expect(mockCreateMutateAsync).not.toHaveBeenCalled();
  });

  it('creates a contact under the selected account', async () => {
    render(<CRMContactNewScreen />);
    fireEvent.press(screen.getByTestId('crm-contact-form-account-acc-1'));
    fireEvent.changeText(screen.getByTestId('crm-contact-form-first-name'), 'Ada');
    fireEvent.changeText(screen.getByTestId('crm-contact-form-last-name'), 'Lovelace');
    fireEvent.changeText(screen.getByTestId('crm-contact-form-email'), 'ada@example.test');
    fireEvent.press(screen.getByTestId('crm-contact-form-submit'));

    await waitFor(() => expect(mockCreateMutateAsync).toHaveBeenCalledWith({
      accountId: 'acc-1',
      firstName: 'Ada',
      lastName: 'Lovelace',
      email: 'ada@example.test',
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/contacts');
  });

  it('loads existing contact data and updates the CRM contact detail', async () => {
    mockContactData = {
      id: 'contact-1',
      accountId: 'acc-1',
      firstName: 'Ada',
      lastName: 'Lovelace',
      email: 'ada@example.test',
      title: 'CTO',
    };

    render(<CRMContactEditScreen />);
    expect(screen.getByDisplayValue('Ada')).toBeTruthy();
    fireEvent.changeText(screen.getByTestId('crm-contact-form-phone'), '+1 555 0101');
    fireEvent.press(screen.getByTestId('crm-contact-form-submit'));

    await waitFor(() => expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
      id: 'contact-1',
      data: {
        accountId: 'acc-1',
        firstName: 'Ada',
        lastName: 'Lovelace',
        email: 'ada@example.test',
        phone: '+1 555 0101',
        title: 'CTO',
      },
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/contacts/contact-1');
  });
});
