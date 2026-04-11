// Home screen and CRM hub — render, navigation, loading/error states
// FR-300 (Home screen), FR-301 (CRM hub)


import React from 'react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render, fireEvent, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import HomeScreen from '../../app/(tabs)/home/index';
import CRMHub from '../../app/(tabs)/crm/index';

// ─── Mocks ───────────────────────────────────────────────────────────────────

const mockUseSignals = jest.fn();
const mockUsePendingApprovals = jest.fn();
const mockUseDismissSignal = jest.fn();
const mockUseDecideApproval = jest.fn();
const mockPush = jest.fn();
const mockBack = jest.fn();

jest.mock('../../src/hooks/useAgentSpec', () => ({
  useSignals: (...args: unknown[]) => mockUseSignals(...args),
  usePendingApprovals: (...args: unknown[]) => mockUsePendingApprovals(...args),
  useDismissSignal: (...args: unknown[]) => mockUseDismissSignal(...args),
  useDecideApproval: (...args: unknown[]) => mockUseDecideApproval(...args),
  useSignalsByEntity: jest.fn().mockReturnValue({ data: [], isLoading: false }),
  useWorkflow: jest.fn().mockReturnValue({ data: null, isLoading: false }),
  useActivateWorkflow: jest.fn().mockReturnValue({ mutate: jest.fn(), isPending: false }),
  useExecuteWorkflow: jest.fn().mockReturnValue({ mutate: jest.fn(), isPending: false }),
  useWorkflows: jest.fn().mockReturnValue({ data: { pages: [] }, isLoading: false, isRefetching: false, isFetchingNextPage: false, hasNextPage: false, fetchNextPage: jest.fn(), refetch: jest.fn() }),
}));

jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockPush, back: mockBack }),
  useLocalSearchParams: () => ({ id: 'sig-1' }),
  Stack: {
    Screen: ({ children }: { children?: React.ReactNode }) => children ?? null,
  },
}));

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: (sel: (s: { workspaceId: string }) => unknown) => sel({ workspaceId: 'ws-1' }),
}));

jest.mock('../../src/services/api', () => ({
  workflowApi: { verifyWorkflow: jest.fn() },
}));

// ─── Helpers ─────────────────────────────────────────────────────────────────

const baseSignal = {
  id: 'sig-1',
  workspace_id: 'ws-1',
  entity_type: 'deal',
  entity_id: 'd-1',
  signal_type: 'churn_risk',
  confidence: 0.9,
  evidence_ids: [],
  source_type: 'llm',
  source_id: 'r-1',
  metadata: {},
  status: 'active',
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

const baseApproval = {
  id: 'apr-1',
  workspace_id: 'ws-1',
  requested_by: 'u-1',
  approver_id: 'u-2',
  action: 'send_email',
  payload: {},
  status: 'pending',
  expires_at: new Date(Date.now() + 3_600_000).toISOString(),
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

function setupMocks() {
  mockUseSignals.mockReturnValue({
    data: { pages: [[baseSignal]] },
    isLoading: false,
    refetch: jest.fn(),
  });
  mockUsePendingApprovals.mockReturnValue({
    data: [baseApproval],
    isLoading: false,
    refetch: jest.fn(),
  });
  mockUseDismissSignal.mockReturnValue({ mutate: jest.fn(), isPending: false });
  mockUseDecideApproval.mockReturnValue({ mutate: jest.fn(), isPending: false });
}

// ─── Home screen ─────────────────────────────────────────────────────────────

describe('Home screen (index)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    setupMocks();
  });

  it('renders HomeFeed with signals and approvals', async () => {
    const { getByTestId } = render(
      <PaperProvider>
        <HomeScreen />
      </PaperProvider>
    );
    expect(getByTestId('home-feed')).toBeTruthy();
  });

  it('passes pending approval count to HomeFeed', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <HomeScreen />
      </PaperProvider>
    );
    // pendingApprovalCount = 1 → chip shows "Approvals (1)"
    expect(within(getByTestId('home-feed-chip-approvals')).getByText(/1/)).toBeTruthy();
  });

  it('navigates to signal detail on signal press', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <HomeScreen />
      </PaperProvider>
    );
    fireEvent.press(getByTestId(`home-feed-signal-${baseSignal.id}`));
    expect(mockPush).toHaveBeenCalledWith(`/home/signal/${baseSignal.id}`);
  });

  it('shows empty state when no signals and no approvals', () => {
    mockUseSignals.mockReturnValue({ data: { pages: [] }, isLoading: false, refetch: jest.fn() });
    mockUsePendingApprovals.mockReturnValue({ data: [], isLoading: false, refetch: jest.fn() });
    const { getByTestId } = render(
      <PaperProvider>
        <HomeScreen />
      </PaperProvider>
    );
    expect(getByTestId('home-feed-empty')).toBeTruthy();
  });
});

// ─── CRM Hub ─────────────────────────────────────────────────────────────────

describe('CRM Hub screen', () => {
  it('renders 4 entity cards', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <CRMHub />
      </PaperProvider>
    );
    expect(getByTestId('crm-hub-accounts')).toBeTruthy();
    expect(getByTestId('crm-hub-contacts')).toBeTruthy();
    expect(getByTestId('crm-hub-deals')).toBeTruthy();
    expect(getByTestId('crm-hub-cases')).toBeTruthy();
  });

  it('navigates to /crm/accounts when Accounts card is pressed', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <CRMHub />
      </PaperProvider>
    );
    fireEvent.press(getByTestId('crm-hub-accounts'));
    expect(mockPush).toHaveBeenCalledWith('/crm/accounts');
  });

  it('navigates to /crm/deals when Deals card is pressed', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <CRMHub />
      </PaperProvider>
    );
    fireEvent.press(getByTestId('crm-hub-deals'));
    expect(mockPush).toHaveBeenCalledWith('/crm/deals');
  });
});