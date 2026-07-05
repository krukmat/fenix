import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react-native';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush }),
}));

const mockUseInbox = jest.fn();
const mockUseApproveApproval = jest.fn();
const mockUseRejectApproval = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useInbox: () => mockUseInbox(),
  useApproveApproval: () => mockUseApproveApproval(),
  useRejectApproval: () => mockUseRejectApproval(),
}));

jest.mock('../../../../src/components/approvals/ApprovalCard', () => {
  const React = require('react');
  const { View, Text, Pressable } = require('react-native');
  return {
    ApprovalCard: ({
      approval,
      testIDPrefix,
      decisionFeedback,
      onApprove,
      onReject,
      disabled,
    }: {
      approval: { id: string; action: string };
      testIDPrefix: string;
      decisionFeedback?: { title: string; body: string };
      onApprove: (id: string, comment?: string) => void;
      onReject: (id: string, reason: string) => void;
      disabled?: boolean;
    }) =>
      React.createElement(View, { testID: testIDPrefix },
        React.createElement(Text, { testID: `${testIDPrefix}-action` }, approval.action),
        decisionFeedback
          ? React.createElement(Text, { testID: `${testIDPrefix}-feedback-title` }, decisionFeedback.title)
          : null,
        decisionFeedback
          ? React.createElement(Text, { testID: `${testIDPrefix}-feedback-body` }, decisionFeedback.body)
          : null,
        React.createElement(Pressable, {
          testID: `${testIDPrefix}-approve`,
          onPress: () => onApprove(approval.id, 'ship it'),
          accessibilityState: { disabled: !!disabled },
        }),
        React.createElement(Pressable, {
          testID: `${testIDPrefix}-reject`,
          onPress: () => onReject(approval.id, 'not authorized'),
          accessibilityState: { disabled: !!disabled },
        })
      ),
  };
});

jest.mock('../../../../src/components/signals/SignalCard', () => {
  const React = require('react');
  const { View, Text, Pressable } = require('react-native');
  return {
    SignalCard: ({
      signal,
      testIDPrefix,
      onPress,
    }: {
      signal: { id: string; signal_type: string };
      testIDPrefix: string;
      onPress?: (signal: { id: string; signal_type: string }) => void;
    }) =>
      React.createElement(View, { testID: testIDPrefix },
        React.createElement(Text, { testID: `${testIDPrefix}-type` }, signal.signal_type),
        React.createElement(Pressable, {
          testID: `${testIDPrefix}-open`,
          onPress: () => onPress?.(signal),
        })
      ),
  };
});

const futureExpiry = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(); // 24h from now
const laterFutureExpiry = new Date(Date.now() + 48 * 60 * 60 * 1000).toISOString(); // 48h from now

const approval = {
  id: 'apr-1',
  workspaceId: 'ws-1',
  requestedBy: 'u-1',
  approverId: 'u-2',
  action: 'send_email',
  payload: {},
  status: 'pending',
  expiresAt: futureExpiry,
  createdAt: '2026-04-08T10:00:00Z',
  updatedAt: '2026-04-08T10:00:00Z',
};

const approvalLaterExpiry = {
  ...approval,
  id: 'apr-2',
  action: 'update_case',
  expiresAt: laterFutureExpiry,
  createdAt: '2026-04-08T11:00:00Z',
  updatedAt: '2026-04-08T11:00:00Z',
};

const handoff = {
  run_id: 'run-1',
  handoff: {
    run_id: 'run-1',
    reason: 'Needs human review',
    conversation_context: 'Customer asked for exception approval',
    evidence_count: 2,
    entity_type: 'case',
    entity_id: 'case-1',
    created_at: '2026-04-08T10:00:00Z',
  },
};

const newerHandoff = {
  run_id: 'run-2',
  handoff: {
    ...handoff.handoff,
    run_id: 'run-2',
    reason: 'Recent escalation',
    created_at: '2026-04-08T12:00:00Z',
  },
};

const accountHandoff = {
  run_id: 'run-3',
  handoff: {
    ...handoff.handoff,
    run_id: 'run-3',
    entity_type: 'account',
    entity_id: 'acc-1',
  },
};

const dealHandoff = {
  run_id: 'run-4',
  handoff: {
    ...handoff.handoff,
    run_id: 'run-4',
    entity_type: 'deal',
    entity_id: 'deal-1',
  },
};

const fallbackHandoff = {
  run_id: 'run-5',
  handoff: {
    ...handoff.handoff,
    run_id: 'run-5',
    entity_type: undefined,
    entity_id: undefined,
  },
};

const caseContextOnlyHandoff = {
  run_id: 'run-6',
  handoff: {
    ...handoff.handoff,
    run_id: 'run-6',
    entity_type: undefined,
    entity_id: undefined,
    caseId: 'case-ctx-1',
    triggerContext: {
      entity_type: 'case',
      entity_id: 'case-ctx-1',
    },
  },
};

const signal = {
  id: 'sig-1',
  workspace_id: 'ws-1',
  entity_type: 'deal',
  entity_id: 'deal-1',
  signal_type: 'churn_risk',
  confidence: 0.9,
  evidence_ids: [],
  source_type: 'agent',
  source_id: 'src-1',
  metadata: {},
  status: 'active',
  created_at: '2026-04-08T10:00:00Z',
  updated_at: '2026-04-08T10:00:00Z',
};

const weakerSignal = {
  ...signal,
  id: 'sig-2',
  signal_type: 'renewal_risk',
  confidence: 0.6,
  created_at: '2026-04-08T12:00:00Z',
  updated_at: '2026-04-08T12:00:00Z',
};

const pastExpiry = new Date(Date.now() - 60_000).toISOString(); // 1 min ago

const expiredPendingApproval = {
  ...approval,
  id: 'apr-3',
  action: 'expired_pending',
  expiresAt: pastExpiry,
};

const rejectedRun = {
  id: 'run-denied-1',
  workspaceId: 'ws-1',
  agentDefinitionId: 'agent-1',
  triggerType: 'manual',
  status: 'denied_by_policy',
  entity_type: 'case',
  entity_id: 'case-1',
  rejection_reason: 'External send blocked by policy',
  startedAt: '2026-04-08T10:00:00Z',
  completedAt: '2026-04-08T10:05:00Z',
  createdAt: '2026-04-08T10:00:00Z',
};

const newerRejectedRun = {
  ...rejectedRun,
  id: 'run-denied-2',
  rejection_reason: 'Manager approval required',
  completedAt: '2026-04-08T12:05:00Z',
  createdAt: '2026-04-08T12:00:00Z',
};

function makeInboxState(data?: {
  approvals?: typeof approval[];
  handoffs?: typeof handoff[];
  signals?: typeof signal[];
  rejected?: typeof rejectedRun[];
}) {
  return {
    data: {
      approvals: data?.approvals ?? [],
      handoffs: data?.handoffs ?? [],
      signals: data?.signals ?? [],
      rejected: data?.rejected ?? [],
    },
    isLoading: false,
    error: null,
    refetch: jest.fn(),
  };
}

function renderInbox() {
  const { default: Screen } = require('../../../../app/(tabs)/inbox/index');
  render(React.createElement(Screen));
}

function renderInboxWith(data?: Parameters<typeof makeInboxState>[0]) {
  mockUseInbox.mockReturnValue(makeInboxState(data));
  renderInbox();
}

function mockApprove(mutate: ReturnType<typeof jest.fn>) {
  mockUseApproveApproval.mockReturnValue({ mutate, isPending: false });
}

function mockApproveSuccess() {
  const mutate = jest.fn((_vars, options: { onSuccess?: () => void }) => options?.onSuccess?.());
  mockApprove(mutate);
  return mutate;
}

function mockApproveError(errorMessage: string) {
  const mutate = jest.fn((_vars, options: { onError?: (error: Error) => void }) =>
    options?.onError?.(new Error(errorMessage))
  );
  mockApprove(mutate);
  return mutate;
}

describe('InboxScreen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseApproveApproval.mockReturnValue({ mutate: jest.fn(), isPending: false });
    mockUseRejectApproval.mockReturnValue({ mutate: jest.fn(), isPending: false });
    mockUseInbox.mockReturnValue(makeInboxState());
  });

  it('shows loading state while inbox is fetching', () => {
    mockUseInbox.mockReturnValue({ data: null, isLoading: true, error: null, refetch: jest.fn() });
    renderInbox();
    expect(screen.getByTestId('inbox-loading')).toBeTruthy();
  });

  it('shows error state when inbox query fails', () => {
    mockUseInbox.mockReturnValue({
      data: null,
      isLoading: false,
      error: new Error('network failed'),
      refetch: jest.fn(),
    });
    renderInbox();
    expect(screen.getByTestId('inbox-error')).toBeTruthy();
  });

  it('retries the inbox query from the error state', () => {
    const refetch = jest.fn();
    mockUseInbox.mockReturnValue({
      data: null,
      isLoading: false,
      error: new Error('network failed'),
      refetch,
    });
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-retry'));
    expect(refetch).toHaveBeenCalled();
  });

  it('shows empty state when inbox has no items', () => {
    renderInbox();
    expect(screen.getByTestId('inbox-empty')).toBeTruthy();
    expect(screen.getByTestId('inbox-total-count').props.children).toContain(0);
  });

  it('renders approvals, handoffs, signals, and rejected runs in one inbox feed', () => {
    mockUseInbox.mockReturnValue(makeInboxState({
      approvals: [approval],
      handoffs: [handoff],
      signals: [signal],
      rejected: [rejectedRun],
    }));
    renderInbox();
    expect(screen.getByTestId('inbox-approval-apr-1')).toBeTruthy();
    expect(screen.getByTestId('inbox-handoff-run-1')).toBeTruthy();
    expect(screen.getByTestId('inbox-signal-sig-1')).toBeTruthy();
    expect(screen.getByTestId('inbox-rejected-run-denied-1')).toBeTruthy();
    expect(screen.getByTestId('inbox-total-count').props.children).toContain(4);
  });

  it('renders the five inbox filter chips', () => {
    renderInbox();
    expect(screen.getByTestId('inbox-chip-all')).toBeTruthy();
    expect(screen.getByTestId('inbox-chip-approval')).toBeTruthy();
    expect(screen.getByTestId('inbox-chip-handoff')).toBeTruthy();
    expect(screen.getByTestId('inbox-chip-signal')).toBeTruthy();
    expect(screen.getByTestId('inbox-chip-rejected')).toBeTruthy();
  });

  it('groups inbox items into named decision-queue sections instead of interleaving them', () => {
    mockUseInbox.mockReturnValue({
      data: {
        approvals: [approvalLaterExpiry, approval],
        handoffs: [handoff, newerHandoff],
        signals: [weakerSignal, signal],
        rejected: [rejectedRun, newerRejectedRun],
      },
    });
    renderInbox();
    // Approvals group first (expiry-ordered), then handoffs, then signals (confidence-ordered), then rejected.
    expect(screen.getByTestId('inbox-item-0').props.accessibilityLabel).toBe('approval:apr-1');
    expect(screen.getByTestId('inbox-item-1').props.accessibilityLabel).toBe('approval:apr-2');
    expect(screen.getByTestId('inbox-item-2').props.accessibilityLabel).toBe('handoff:run-2');
    expect(screen.getByTestId('inbox-item-3').props.accessibilityLabel).toBe('handoff:run-1');
    expect(screen.getByTestId('inbox-item-4').props.accessibilityLabel).toBe('signal:sig-1');
    expect(screen.getByTestId('inbox-item-5').props.accessibilityLabel).toBe('signal:sig-2');
    expect(screen.getByTestId('inbox-item-6').props.accessibilityLabel).toBe('rejected:run-denied-2');
    expect(screen.getByTestId('inbox-item-7').props.accessibilityLabel).toBe('rejected:run-denied-1');
  });

  it('renders group headers with count eyebrows for each non-empty section', () => {
    renderInboxWith({
      approvals: [approval],
      handoffs: [handoff],
      signals: [signal],
      rejected: [rejectedRun],
    });
    expect(screen.getByTestId('inbox-group-approval')).toBeTruthy();
    expect(screen.getByTestId('inbox-group-approval-count').props.children).toBe(1);
    expect(screen.getByTestId('inbox-group-handoff-count').props.children).toBe(1);
    expect(screen.getByTestId('inbox-group-signal-count').props.children).toBe(1);
    expect(screen.getByTestId('inbox-group-rejected-count').props.children).toBe(1);
  });

  it('omits empty group headers when a section has no items', () => {
    renderInboxWith({ approvals: [approval] });
    expect(screen.getByTestId('inbox-group-approval')).toBeTruthy();
    expect(screen.queryByTestId('inbox-group-handoff')).toBeNull();
    expect(screen.queryByTestId('inbox-group-signal')).toBeNull();
    expect(screen.queryByTestId('inbox-group-rejected')).toBeNull();
  });

  it('shows the designed decision-queue empty state copy', () => {
    renderInbox();
    expect(screen.getByText('No decisions waiting')).toBeTruthy();
  });

  it('moves a just-decided approval out of the awaiting-approval group into a decided group', () => {
    mockApproveSuccess();
    renderInboxWith({ approvals: [approval, approvalLaterExpiry] });

    // Before deciding: both approvals sit under the "awaiting" group with count 2.
    expect(screen.getByTestId('inbox-group-approval-count').props.children).toBe(2);
    expect(screen.queryByTestId('inbox-group-approval-decided')).toBeNull();

    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-approve'));

    // After deciding apr-1: it moves to the decided group, apr-2 remains awaiting.
    expect(screen.getByTestId('inbox-group-approval-count').props.children).toBe(1);
    expect(screen.getByTestId('inbox-group-approval-decided')).toBeTruthy();
    expect(screen.getByTestId('inbox-group-approval-decided-count').props.children).toBe(1);
  });

  it('moves an already-decided conflict out of the awaiting-approval group even though status stays pending', () => {
    mockApproveError('approval request is already decided');
    renderInboxWith({ approvals: [approval, approvalLaterExpiry] });

    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-approve'));

    // apr-1's raw status is still 'pending' (server object was not mutated), but the
    // conflict feedback means the operator must not treat it as awaiting action.
    expect(screen.getByTestId('inbox-approval-apr-1-feedback-title').props.children).toBe('Approval already decided');
    expect(screen.getByTestId('inbox-group-approval-count').props.children).toBe(1);
    expect(screen.getByTestId('inbox-group-approval-decided-count').props.children).toBe(1);
  });

  it('orders the decided-approvals group by decision recency, not by original expiry', () => {
    const mutateApprove = jest.fn((_vars, options: { onSuccess?: () => void }) => options?.onSuccess?.());
    mockUseApproveApproval.mockReturnValue({ mutate: mutateApprove, isPending: false });
    mockUseRejectApproval.mockReturnValue({ mutate: jest.fn(), isPending: false });
    // approvalLaterExpiry expires after `approval`, so an expiry-based sort would list it second;
    // deciding it first must still place it first in the decided group (most recent decision first).
    mockUseInbox.mockReturnValue(makeInboxState({ approvals: [approval, approvalLaterExpiry] }));
    renderInbox();

    fireEvent.press(screen.getByTestId('inbox-approval-apr-2-approve'));
    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-approve'));

    expect(screen.getByTestId('inbox-item-0').props.accessibilityLabel).toBe('approval:apr-1');
    expect(screen.getByTestId('inbox-item-1').props.accessibilityLabel).toBe('approval:apr-2');
  });

  it('orders an already-decided conflict by its resolution time, not its stale pre-decision updatedAt', () => {
    // apr-1 resolves via a normal successful approve; apr-2 resolves via an
    // "already decided" conflict. Both must land in the decided group ordered by
    // when each was actually resolved (apr-1 first, then apr-2), not by their
    // original expiresAt/createdAt (which would rank apr-2 before apr-1).
    const mutate = jest.fn(
      (vars: { id: string }, options: { onSuccess?: () => void; onError?: (error: Error) => void }) => {
        if (vars.id === 'apr-1') {
          options?.onSuccess?.();
        } else {
          options?.onError?.(new Error('approval request is already decided'));
        }
      }
    );
    mockApprove(mutate);
    mockUseInbox.mockReturnValue(makeInboxState({ approvals: [approval, approvalLaterExpiry] }));
    renderInbox();

    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-approve'));
    fireEvent.press(screen.getByTestId('inbox-approval-apr-2-approve'));

    expect(screen.getByTestId('inbox-item-0').props.accessibilityLabel).toBe('approval:apr-2');
    expect(screen.getByTestId('inbox-item-1').props.accessibilityLabel).toBe('approval:apr-1');
  });

  it('uses the conflict accent color for an already-decided approval, not the stale pending accent', () => {
    mockApproveError('approval request is already decided');
    renderInboxWith({ approvals: [approval] });

    const flattenStyle = (style: unknown): Record<string, unknown> =>
      Object.assign({}, ...(Array.isArray(style) ? style : [style]));
    const pendingAccentColor = flattenStyle(screen.getByTestId('inbox-item-0').props.style).borderLeftColor;

    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-approve'));

    const conflictAccentColor = flattenStyle(screen.getByTestId('inbox-item-0').props.style).borderLeftColor;
    expect(conflictAccentColor).not.toBe(pendingAccentColor);
    expect(conflictAccentColor).toBe('#EF4444');
  });

  it('moves a wall-clock-expired pending approval out of the awaiting group even without a conflict response', () => {
    renderInboxWith({ approvals: [approval, expiredPendingApproval] });

    expect(screen.getByTestId('inbox-group-approval-count').props.children).toBe(1);
    expect(screen.getByTestId('inbox-group-approval-decided-count').props.children).toBe(1);
  });

  it('lets fresh server truth override a stale local already-decided conflict after refetch', () => {
    mockApproveError('approval request is already decided');
    mockUseInbox.mockReturnValue(makeInboxState({ approvals: [approval] }));
    const { rerender } = render(React.createElement(require('../../../../app/(tabs)/inbox/index').default));

    fireEvent.press(screen.getAllByTestId('inbox-approval-apr-1-approve')[0]);
    expect(screen.getByTestId('inbox-approval-apr-1-feedback-title').props.children).toBe('Approval already decided');

    // Refetch now returns fresh server truth: the request was actually rejected.
    mockUseInbox.mockReturnValue(makeInboxState({
      approvals: [{ ...approval, status: 'rejected', decidedAt: '2026-07-05T12:00:00Z', updatedAt: '2026-07-05T12:00:00Z' }],
    }));
    const { default: Screen } = require('../../../../app/(tabs)/inbox/index');
    rerender(React.createElement(Screen));

    expect(screen.queryByTestId('inbox-approval-apr-1-feedback-title')).toBeNull();
  });

  it('re-groups a pending approval into the decided section once it crosses expiry while the screen stays open', () => {
    jest.useFakeTimers();
    try {
      const almostExpired = { ...approval, expiresAt: new Date(Date.now() + 500).toISOString() };
      renderInboxWith({ approvals: [almostExpired] });

      expect(screen.getByTestId('inbox-group-approval-count').props.children).toBe(1);
      expect(screen.queryByTestId('inbox-group-approval-decided')).toBeNull();

      act(() => {
        jest.advanceTimersByTime(61_000);
      });

      expect(screen.queryByTestId('inbox-group-approval')).toBeNull();
      expect(screen.getByTestId('inbox-group-approval-decided-count').props.children).toBe(1);
    } finally {
      jest.useRealTimers();
    }
  });

  it.each([
    ['success', undefined, 'Approval recorded', 'Approval recorded'],
    ['already decided', 'approval request is already decided', 'Approval already decided', null],
    ['expired', 'approval request is expired', 'Approval expired', 'Approval expired'],
  ])(
    'reconciles a %s decision against a refetch that drops the approval from the payload',
    (_label, errorMessage, initialTitle, titleAfterRefetch) => {
      if (errorMessage) {
        mockApproveError(errorMessage);
      } else {
        mockApproveSuccess();
      }
      mockUseInbox.mockReturnValue(makeInboxState({ approvals: [approval] }));
      const { rerender } = render(React.createElement(require('../../../../app/(tabs)/inbox/index').default));

      fireEvent.press(screen.getAllByTestId('inbox-approval-apr-1-approve')[0]);
      expect(screen.getByTestId('inbox-approval-apr-1-feedback-title').props.children).toBe(initialTitle);

      // Refetch now returns an empty payload: the server no longer lists this approval anywhere.
      // A known outcome (success, or a deterministically-verifiable state like "expired") must
      // survive; an unresolved guess ("already decided") must not.
      mockUseInbox.mockReturnValue(makeInboxState());
      const { default: Screen } = require('../../../../app/(tabs)/inbox/index');
      rerender(React.createElement(Screen));

      if (titleAfterRefetch) {
        expect(screen.getByTestId('inbox-approval-apr-1-feedback-title').props.children).toBe(titleAfterRefetch);
      } else {
        expect(screen.queryByTestId('inbox-approval-apr-1')).toBeNull();
      }
    }
  );

  it('filters the feed to approvals', () => {
    mockUseInbox.mockReturnValue(makeInboxState({
      approvals: [approval],
      handoffs: [handoff],
      signals: [signal],
      rejected: [rejectedRun],
    }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-chip-approval'));
    expect(screen.getByTestId('inbox-approval-apr-1')).toBeTruthy();
    expect(screen.queryByTestId('inbox-handoff-run-1')).toBeNull();
    expect(screen.queryByTestId('inbox-signal-sig-1')).toBeNull();
    expect(screen.queryByTestId('inbox-rejected-run-denied-1')).toBeNull();
    expect(screen.getByTestId('inbox-visible-count').props.children).toContain(1);
  });

  it('filters the feed to handoffs', () => {
    mockUseInbox.mockReturnValue(makeInboxState({
      approvals: [approval],
      handoffs: [handoff],
      signals: [signal],
      rejected: [rejectedRun],
    }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-chip-handoff'));
    expect(screen.queryByTestId('inbox-approval-apr-1')).toBeNull();
    expect(screen.getByTestId('inbox-handoff-run-1')).toBeTruthy();
    expect(screen.queryByTestId('inbox-signal-sig-1')).toBeNull();
    expect(screen.queryByTestId('inbox-rejected-run-denied-1')).toBeNull();
  });

  it('filters the feed to signals', () => {
    mockUseInbox.mockReturnValue(makeInboxState({
      approvals: [approval],
      handoffs: [handoff],
      signals: [signal],
      rejected: [rejectedRun],
    }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-chip-signal'));
    expect(screen.queryByTestId('inbox-approval-apr-1')).toBeNull();
    expect(screen.queryByTestId('inbox-handoff-run-1')).toBeNull();
    expect(screen.getByTestId('inbox-signal-sig-1')).toBeTruthy();
  });

  it('filters the feed to rejected runs', () => {
    mockUseInbox.mockReturnValue(makeInboxState({
      approvals: [approval],
      handoffs: [handoff],
      signals: [signal],
      rejected: [rejectedRun],
    }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-chip-rejected'));
    expect(screen.queryByTestId('inbox-approval-apr-1')).toBeNull();
    expect(screen.queryByTestId('inbox-handoff-run-1')).toBeNull();
    expect(screen.queryByTestId('inbox-signal-sig-1')).toBeNull();
    expect(screen.getByTestId('inbox-rejected-run-denied-1')).toBeTruthy();
  });

  it('returns to the full mixed feed when All is selected', () => {
    mockUseInbox.mockReturnValue(makeInboxState({
      approvals: [approval],
      handoffs: [handoff],
      signals: [signal],
      rejected: [rejectedRun],
    }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-chip-signal'));
    fireEvent.press(screen.getByTestId('inbox-chip-all'));
    expect(screen.getByTestId('inbox-approval-apr-1')).toBeTruthy();
    expect(screen.getByTestId('inbox-handoff-run-1')).toBeTruthy();
    expect(screen.getByTestId('inbox-signal-sig-1')).toBeTruthy();
    expect(screen.getByTestId('inbox-rejected-run-denied-1')).toBeTruthy();
    expect(screen.getByTestId('inbox-visible-count').props.children).toContain(4);
  });

  it('approves inbox approvals inline and refreshes the feed on success', () => {
    const mutate = jest.fn((_vars, options: { onSuccess?: () => void }) => options?.onSuccess?.());
    mockApprove(mutate);
    renderInboxWith({ approvals: [approval] });
    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-approve'));
    expect(mutate).toHaveBeenCalledWith({ id: 'apr-1', reason: 'ship it' }, expect.any(Object));
    expect(screen.getByTestId('inbox-approval-apr-1-feedback-title').props.children).toBe('Approval recorded');
  });

  it('rejects inbox approvals inline and renders terminal feedback on success', () => {
    const mutate = jest.fn((_vars, options: { onSuccess?: () => void }) => options?.onSuccess?.());
    mockUseRejectApproval.mockReturnValue({ mutate, isPending: false });
    renderInboxWith({ approvals: [approval] });
    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-reject'));
    expect(mutate).toHaveBeenCalledWith({ id: 'apr-1', reason: 'not authorized' }, expect.any(Object));
    expect(screen.getByTestId('inbox-approval-apr-1-feedback-title').props.children).toBe('Rejection recorded');
  });

  it('shows an inline error when approval mutation fails', () => {
    const mutate = jest.fn((_vars, options: { onError?: (error: Error) => void }) =>
      options?.onError?.(new Error('mutation failed'))
    );
    mockApprove(mutate);
    renderInboxWith({ approvals: [approval] });
    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-approve'));
    expect(screen.getByTestId('inbox-approval-action-error')).toBeTruthy();
    expect(screen.getByText('mutation failed')).toBeTruthy();
  });

  it.each([
    ['approval request is expired', 'Approval expired'],
    ['approval request is already decided', 'Approval already decided'],
  ])('renders a designed governed conflict state for "%s" instead of a generic error', (errorMessage, feedbackTitle) => {
    mockApprove(jest.fn((_vars, options: { onError?: (error: Error) => void }) => options?.onError?.(new Error(errorMessage))));
    renderInboxWith({ approvals: [approval] });
    fireEvent.press(screen.getByTestId('inbox-approval-apr-1-approve'));
    expect(screen.getByTestId('inbox-approval-apr-1-feedback-title').props.children).toBe(feedbackTitle);
    expect(screen.queryByTestId('inbox-approval-action-error')).toBeNull();
  });

  it('disables inbox approval actions while a decision mutation is pending', () => {
    mockUseApproveApproval.mockReturnValue({ mutate: jest.fn(), isPending: true });
    mockUseInbox.mockReturnValue(makeInboxState({ approvals: [approval] }));
    renderInbox();
    expect(screen.getByTestId('inbox-approval-apr-1-approve').props.accessibilityState.disabled).toBe(true);
    expect(screen.getByTestId('inbox-approval-apr-1-reject').props.accessibilityState.disabled).toBe(true);
  });

  it('navigates handoff items to support for case context', () => {
    mockUseInbox.mockReturnValue(makeInboxState({ handoffs: [handoff] }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-handoff-run-1'));
    expect(mockPush).toHaveBeenCalledWith('/support/case-1');
  });

  it('navigates handoff items to sales for account context', () => {
    mockUseInbox.mockReturnValue(makeInboxState({ handoffs: [accountHandoff] }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-handoff-run-3'));
    expect(mockPush).toHaveBeenCalledWith('/sales/acc-1');
  });

  it('navigates handoff items to sales deal detail for deal context', () => {
    mockUseInbox.mockReturnValue(makeInboxState({ handoffs: [dealHandoff] }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-handoff-run-4'));
    expect(mockPush).toHaveBeenCalledWith('/sales/deals/deal-1');
  });

  it('falls back to activity detail when handoff lacks entity context', () => {
    mockUseInbox.mockReturnValue(makeInboxState({ handoffs: [fallbackHandoff] }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-handoff-run-5'));
    expect(mockPush).toHaveBeenCalledWith('/activity/run-5');
  });

  it('does not navigate handoffs to legacy routes', () => {
    mockUseInbox.mockReturnValue(makeInboxState({ handoffs: [accountHandoff, dealHandoff, fallbackHandoff] }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-handoff-run-3'));
    fireEvent.press(screen.getByTestId('inbox-handoff-run-4'));
    fireEvent.press(screen.getByTestId('inbox-handoff-run-5'));
    expect(mockPush).not.toHaveBeenCalledWith(expect.stringMatching(/\/\(tabs\)\/crm\//));
    expect(mockPush).not.toHaveBeenCalledWith(expect.stringMatching(/\/\(tabs\)\/copilot/));
  });

  it('navigates handoffs using triggerContext/caseId when top-level entity fields are absent', () => {
    mockUseInbox.mockReturnValue(makeInboxState({ handoffs: [caseContextOnlyHandoff] }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-handoff-run-6'));
    expect(mockPush).toHaveBeenCalledWith('/support/case-ctx-1');
  });

  it('opens signal detail from inbox signal items', () => {
    mockUseInbox.mockReturnValue(makeInboxState({ signals: [signal] }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-signal-sig-1-open'));
    expect(mockPush).toHaveBeenCalledWith({
      pathname: '/(tabs)/home/signal/[id]',
      params: { id: 'sig-1', entity_type: 'deal', entity_id: 'deal-1' },
    });
  });

  it('opens rejected items on the activity detail screen', () => {
    mockUseInbox.mockReturnValue(makeInboxState({ rejected: [rejectedRun] }));
    renderInbox();
    fireEvent.press(screen.getByTestId('inbox-rejected-run-denied-1'));
    expect(mockPush).toHaveBeenCalledWith('/activity/run-denied-1');
  });
});
