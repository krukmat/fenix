import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';
import CRMAccountNewScreen from '../../../../app/(tabs)/crm/accounts/new';
import CRMAccountEditScreen from '../../../../app/(tabs)/crm/accounts/edit/[id]';

const mockReplace = jest.fn();
const mockCreateMutateAsync = jest.fn();
const mockUpdateMutateAsync = jest.fn();
let mockAccountData: unknown = null;

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ replace: mockReplace }),
  useLocalSearchParams: () => ({ id: 'acc-1' }),
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
  useAccount: () => ({ data: mockAccountData, isLoading: false, error: null }),
  useCreateAccount: () => ({ mutateAsync: mockCreateMutateAsync, isPending: false }),
  useUpdateAccount: () => ({ mutateAsync: mockUpdateMutateAsync, isPending: false }),
}));

describe('CRM account forms', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockAccountData = null;
    mockCreateMutateAsync.mockResolvedValue({});
    mockUpdateMutateAsync.mockResolvedValue({});
  });

  it('validates required account name before create submit', async () => {
    render(<CRMAccountNewScreen />);
    fireEvent.press(screen.getByTestId('crm-account-form-submit'));

    expect(await screen.findByText('Account name is required')).toBeTruthy();
    expect(mockCreateMutateAsync).not.toHaveBeenCalled();
  });

  it('creates an account and returns to the CRM account list', async () => {
    render(<CRMAccountNewScreen />);
    fireEvent.changeText(screen.getByTestId('crm-account-form-name'), 'Acme');
    fireEvent.changeText(screen.getByTestId('crm-account-form-industry'), 'Manufacturing');
    fireEvent.changeText(screen.getByTestId('crm-account-form-email'), 'ops@acme.test');
    fireEvent.press(screen.getByTestId('crm-account-form-submit'));

    await waitFor(() => expect(mockCreateMutateAsync).toHaveBeenCalledWith({
      name: 'Acme',
      industry: 'Manufacturing',
      email: 'ops@acme.test',
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/accounts');
  });

  it('loads existing account data and updates the CRM account detail', async () => {
    mockAccountData = {
      account: {
        id: 'acc-1',
        name: 'Acme',
        industry: 'Manufacturing',
        website: 'https://acme.test',
      },
    };

    render(<CRMAccountEditScreen />);
    expect(screen.getByDisplayValue('Acme')).toBeTruthy();
    fireEvent.changeText(screen.getByTestId('crm-account-form-phone'), '+1 555 0100');
    fireEvent.press(screen.getByTestId('crm-account-form-submit'));

    await waitFor(() => expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
      id: 'acc-1',
      data: {
        name: 'Acme',
        industry: 'Manufacturing',
        website: 'https://acme.test',
        phone: '+1 555 0100',
      },
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/accounts/acc-1');
  });
});
