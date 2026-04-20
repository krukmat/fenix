import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { Alert } from 'react-native';
import { fireEvent, render, screen } from '@testing-library/react-native';
import CRMAccountsScreen from '../../../../app/(tabs)/crm/accounts/index';
import CRMAccountDetailScreen from '../../../../app/(tabs)/crm/accounts/[id]';
import CRMCasesScreen from '../../../../app/(tabs)/crm/cases/index';
import CRMCaseDetailScreen from '../../../../app/(tabs)/crm/cases/[id]';
import CRMContactsScreen from '../../../../app/(tabs)/crm/contacts/index';
import CRMContactDetailScreen from '../../../../app/(tabs)/crm/contacts/[id]';
import CRMDealDetailScreen from '../../../../app/(tabs)/crm/deals/[id]';
import CRMDealsScreen from '../../../../app/(tabs)/crm/deals/index';
import CRMHubScreen from '../../../../app/(tabs)/crm/index';
import CRMLeadsScreen from '../../../../app/(tabs)/crm/leads/index';
import CRMLeadDetailScreen from '../../../../app/(tabs)/crm/leads/[id]';

const mockPush = jest.fn();

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush }),
  useLocalSearchParams: () => ({ id: 'acc-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

jest.mock('react-native-paper', () => {
  const React = require('react');
  const { Text, TouchableOpacity, View } = require('react-native');
  const Card = ({ children, onPress, testID, style }: { children: React.ReactNode; onPress?: () => void; testID?: string; style?: unknown }) =>
    React.createElement(TouchableOpacity, { onPress, testID, style }, children);
  Card.Content = ({ children, style }: { children: React.ReactNode; style?: unknown }) =>
    React.createElement(View, { style }, children);
  return {
    Card,
    Text,
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
  };
});

jest.mock('../../../../src/stores/authStore', () => ({
  useAuthStore: (selector: (state: { userId: string | null }) => unknown) => selector({ userId: 'user-1' }),
}));

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

function textContent(value: unknown): string {
  return Array.isArray(value) ? value.join('') : String(value);
}

const mockDeleteAccount = jest.fn();
const mockDeleteContact = jest.fn();
const mockDeleteLead = jest.fn();
const mockDeleteDeal = jest.fn();
const mockDeleteCase = jest.fn();

jest.mock('../../../../src/hooks/useCRM', () => ({
  useAccounts: () => listQuery([{ id: 'acc-1', name: 'Acme', industry: 'Manufacturing' }]),
  useContacts: () => listQuery([{ id: 'contact-1', accountId: 'acc-1', firstName: 'Ada', lastName: 'Lovelace', email: 'ada@example.test' }]),
  useCases: () => listQuery([{ id: 'case-1', subject: 'Cannot login', priority: 'high', status: 'open' }]),
  useDeals: () => listQuery([{ id: 'deal-1', title: 'Expansion', status: 'open', amount: 12000 }]),
  useLeads: () => listQuery([{ id: 'lead-1', source: 'web', status: 'new', metadata: { name: 'Jane Lead' } }]),
  useDeleteAccount: () => ({ mutateAsync: mockDeleteAccount, isPending: false }),
  useDeleteContact: () => ({ mutateAsync: mockDeleteContact, isPending: false }),
  useDeleteLead: () => ({ mutateAsync: mockDeleteLead, isPending: false }),
  useDeleteDeal: () => ({ mutateAsync: mockDeleteDeal, isPending: false }),
  useDeleteCase: () => ({ mutateAsync: mockDeleteCase, isPending: false }),
  useAccount: () => ({
    data: {
      account: { id: 'acc-1', name: 'Acme', industry: 'Manufacturing' },
      contacts: { data: [{ id: 'contact-1', firstName: 'Ada', lastName: 'Lovelace', email: 'ada@example.test' }] },
      deals: { data: [{ id: 'deal-1', title: 'Expansion', status: 'open', amount: 12000 }] },
    },
    isLoading: false,
    error: null,
  }),
  useLead: () => ({
    data: { id: 'lead-1', source: 'web', status: 'new', score: 80, metadata: { name: 'Jane Lead', email: 'jane@example.test' } },
    isLoading: false,
    error: null,
  }),
  useContact: () => ({
    data: { id: 'contact-1', accountId: 'acc-1', firstName: 'Ada', lastName: 'Lovelace', email: 'ada@example.test', title: 'CTO' },
    isLoading: false,
    error: null,
  }),
  useCase: () => ({
    data: { case: { id: 'case-1', subject: 'Cannot login', priority: 'high', status: 'open', channel: 'email' } },
    isLoading: false,
    error: null,
  }),
  useDeal: () => ({
    data: { deal: { id: 'deal-1', title: 'Expansion', status: 'open', amount: 12000, pipelineId: 'pipe-1', stageId: 'stage-1' } },
    isLoading: false,
    error: null,
  }),
  useCreateActivity: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useCreateNote: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useCreateAttachment: () => ({ mutateAsync: jest.fn(), isPending: false }),
}));

describe('CRM read-only routes', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders the real /crm/accounts list and navigates inside /crm', () => {
    render(<CRMAccountsScreen />);
    expect(screen.getByTestId('crm-accounts-list')).toBeTruthy();
    expect(screen.getByText('Acme')).toBeTruthy();
    expect(screen.getByTestId('crm-accounts-primary-action')).toBeTruthy();
    expect(screen.getByTestId('crm-accounts-item-0-select')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-accounts-item-0'));
    expect(mockPush).toHaveBeenCalledWith('/crm/accounts/acc-1');
  });

  it('supports multi-selection controls in the accounts list', () => {
    render(<CRMAccountsScreen />);
    const checkbox = screen.getByTestId('crm-accounts-item-0-select');

    expect(checkbox.props.accessibilityState.checked).toBe(false);
    expect(textContent(screen.getByTestId('crm-accounts-selection-count').props.children)).toBe('0 selected');

    fireEvent.press(checkbox);
    expect(screen.getByTestId('crm-accounts-item-0-select').props.accessibilityState.checked).toBe(true);
    expect(textContent(screen.getByTestId('crm-accounts-selection-count').props.children)).toBe('1 selected');

    fireEvent.press(screen.getByTestId('crm-accounts-clear-selection'));
    expect(screen.getByTestId('crm-accounts-item-0-select').props.accessibilityState.checked).toBe(false);
    expect(textContent(screen.getByTestId('crm-accounts-selection-count').props.children)).toBe('0 selected');

    fireEvent.press(screen.getByTestId('crm-accounts-select-all'));
    expect(screen.getByTestId('crm-accounts-item-0-select').props.accessibilityState.checked).toBe(true);
  });

  it('renders Leads in the CRM hub and navigates to /crm/leads', () => {
    render(<CRMHubScreen />);
    expect(screen.getByTestId('crm-hub-leads')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-hub-leads'));
    expect(mockPush).toHaveBeenCalledWith('/crm/leads');
  });

  it('renders the real /crm/leads list and navigates inside /crm', () => {
    render(<CRMLeadsScreen />);
    expect(screen.getByTestId('crm-leads-list')).toBeTruthy();
    expect(screen.getByText('Jane Lead')).toBeTruthy();
    expect(screen.getByTestId('crm-leads-primary-action')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-leads-item-0'));
    expect(mockPush).toHaveBeenCalledWith('/crm/leads/lead-1');
  });

  it('renders the real /crm/contacts list and exposes create navigation', () => {
    render(<CRMContactsScreen />);
    expect(screen.getByTestId('crm-contacts-list')).toBeTruthy();
    expect(screen.getByText('Ada Lovelace')).toBeTruthy();
    expect(screen.getByTestId('crm-contacts-primary-action')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-contacts-primary-action'));
    expect(mockPush).toHaveBeenCalledWith('/crm/contacts/new');
  });

  it('renders the real /crm/cases list and exposes create navigation', () => {
    render(<CRMCasesScreen />);
    expect(screen.getByTestId('crm-cases-list')).toBeTruthy();
    expect(screen.getByText('Cannot login')).toBeTruthy();
    expect(screen.getByTestId('crm-cases-primary-action')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-cases-primary-action'));
    expect(mockPush).toHaveBeenCalledWith('/crm/cases/new');
  });

  it('renders the real /crm/deals list and exposes create navigation', () => {
    render(<CRMDealsScreen />);
    expect(screen.getByTestId('crm-deals-list')).toBeTruthy();
    expect(screen.getByText('Expansion')).toBeTruthy();
    expect(screen.getByTestId('crm-deals-primary-action')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-deals-primary-action'));
    expect(mockPush).toHaveBeenCalledWith('/crm/deals/new');
  });

  // Task 5 — Detail screens are read-only; Edit moved to list rows
  it('renders account detail as read-only — no edit primary action', () => {
    render(<CRMAccountDetailScreen />);
    expect(screen.getByTestId('crm-account-detail-screen')).toBeTruthy();
    expect(screen.getByTestId('crm-entity-child-forms')).toBeTruthy();
    expect(screen.getByText('Acme')).toBeTruthy();
    expect(screen.queryByTestId('account-copilot-open-button')).toBeNull();
    expect(screen.queryByTestId('crm-account-detail-primary-action')).toBeNull();
  });

  it('renders lead detail as read-only — no edit primary action', () => {
    render(<CRMLeadDetailScreen />);
    expect(screen.getByTestId('crm-lead-detail-screen')).toBeTruthy();
    expect(screen.getByText('Jane Lead')).toBeTruthy();
    expect(screen.getByText('jane@example.test')).toBeTruthy();
    expect(screen.queryByTestId('crm-lead-detail-primary-action')).toBeNull();
  });

  it('renders contact detail as read-only — no edit primary action', () => {
    render(<CRMContactDetailScreen />);
    expect(screen.getByTestId('crm-contact-detail-screen')).toBeTruthy();
    expect(screen.getByText('Ada Lovelace')).toBeTruthy();
    expect(screen.queryByTestId('crm-contact-detail-primary-action')).toBeNull();
  });

  it('renders case detail as read-only — no edit primary action', () => {
    render(<CRMCaseDetailScreen />);
    expect(screen.getByTestId('crm-case-detail-screen')).toBeTruthy();
    expect(screen.getByText('Cannot login')).toBeTruthy();
    expect(screen.queryByTestId('crm-case-detail-primary-action')).toBeNull();
  });

  it('renders deal detail as read-only — no edit primary action', () => {
    render(<CRMDealDetailScreen />);
    expect(screen.getByTestId('crm-deal-detail-screen')).toBeTruthy();
    expect(screen.getByText('Expansion')).toBeTruthy();
    expect(screen.queryByTestId('crm-deal-detail-primary-action')).toBeNull();
  });

  // Task 3 — Bulk Delete
  it('hides Delete selected when no rows are selected', () => {
    render(<CRMAccountsScreen />);
    expect(screen.queryByTestId('crm-accounts-delete-selected')).toBeNull();
  });

  it('shows Delete selected when at least one row is selected', () => {
    render(<CRMAccountsScreen />);
    fireEvent.press(screen.getByTestId('crm-accounts-item-0-select'));
    expect(screen.getByTestId('crm-accounts-delete-selected')).toBeTruthy();
  });

  it('cancel on delete confirmation does not call delete mutation', () => {
    jest.spyOn(Alert, 'alert').mockImplementation((_title, _msg, buttons = []) => {
      const cancel = buttons.find((b) => b.style === 'cancel');
      cancel?.onPress?.();
    });
    render(<CRMAccountsScreen />);
    fireEvent.press(screen.getByTestId('crm-accounts-item-0-select'));
    fireEvent.press(screen.getByTestId('crm-accounts-delete-selected'));
    expect(mockDeleteAccount).not.toHaveBeenCalled();
  });

  it('confirm on delete calls delete mutation for each selected id', async () => {
    (mockDeleteAccount as ReturnType<typeof jest.fn>).mockResolvedValue(undefined);
    jest.spyOn(Alert, 'alert').mockImplementation((_title, _msg, buttons = []) => {
      const confirm = buttons.find((b) => b.style === 'destructive');
      confirm?.onPress?.();
    });
    render(<CRMAccountsScreen />);
    fireEvent.press(screen.getByTestId('crm-accounts-item-0-select'));
    fireEvent.press(screen.getByTestId('crm-accounts-delete-selected'));
    await Promise.resolve();
    expect(mockDeleteAccount).toHaveBeenCalledWith('acc-1');
  });

  // Task 2 — Centralize Edit in List Screens
  it('exposes edit action on account list row and navigates to edit route', () => {
    render(<CRMAccountsScreen />);
    expect(screen.getByTestId('crm-accounts-item-0-edit')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-accounts-item-0-edit'));
    expect(mockPush).toHaveBeenCalledWith('/crm/accounts/edit/acc-1');
  });

  it('exposes edit action on contact list row and navigates to edit route', () => {
    render(<CRMContactsScreen />);
    expect(screen.getByTestId('crm-contacts-item-0-edit')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-contacts-item-0-edit'));
    expect(mockPush).toHaveBeenCalledWith('/crm/contacts/edit/contact-1');
  });

  it('exposes edit action on lead list row and navigates to edit route', () => {
    render(<CRMLeadsScreen />);
    expect(screen.getByTestId('crm-leads-item-0-edit')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-leads-item-0-edit'));
    expect(mockPush).toHaveBeenCalledWith('/crm/leads/edit/lead-1');
  });

  it('exposes edit action on deal list row and navigates to edit route', () => {
    render(<CRMDealsScreen />);
    expect(screen.getByTestId('crm-deals-item-0-edit')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-deals-item-0-edit'));
    expect(mockPush).toHaveBeenCalledWith('/crm/deals/edit/deal-1');
  });

  it('exposes edit action on case list row and navigates to edit route', () => {
    render(<CRMCasesScreen />);
    expect(screen.getByTestId('crm-cases-item-0-edit')).toBeTruthy();

    fireEvent.press(screen.getByTestId('crm-cases-item-0-edit'));
    expect(mockPush).toHaveBeenCalledWith('/crm/cases/edit/case-1');
  });
});
