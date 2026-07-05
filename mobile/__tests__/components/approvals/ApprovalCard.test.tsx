// ApprovalCard — countdown, approve/reject flow, expired state
// FR-071 (Approvals), UC-A6: human approval decision


import React from 'react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render, fireEvent } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { ApprovalCard } from '../../../src/components/approvals/ApprovalCard';
import type { ApprovalRequest } from '../../../src/services/api';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush }),
}));

const futureExpiry = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(); // 2h from now
const pastExpiry = new Date(Date.now() - 60_000).toISOString(); // 1 min ago

const baseApproval: ApprovalRequest = {
  id: 'apr-1',
  workspaceId: 'ws-1',
  requestedBy: 'user-1',
  approverId: 'user-2',
  action: 'send_email',
  resourceType: 'contact',
  resourceId: 'c-1',
  payload: {},
  reason: 'Customer requested follow-up',
  status: 'pending',
  expiresAt: futureExpiry,
  createdAt: '2026-03-01T10:00:00Z',
  updatedAt: '2026-03-01T10:00:00Z',
};

function renderCard(props?: Partial<Parameters<typeof ApprovalCard>[0]>) {
  const onApprove = jest.fn();
  const onReject = jest.fn();
  const utils = render(
    <PaperProvider>
      <ApprovalCard approval={baseApproval} onApprove={onApprove} onReject={onReject} {...props} />
    </PaperProvider>
  );
  return { ...utils, onApprove, onReject };
}

describe('ApprovalCard', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders action and resource', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('approval-card-action').props.children).toBe('send_email');
    expect(getByTestId('approval-card-resource').props.children).toBe('contact · c-1');
  });

  it('renders reason when present', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('approval-card-reason').props.children).toBe('Customer requested follow-up');
  });

  it('renders the approval path with the current state', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('approval-card-path-pending-label').props.children).toBe('Pending');
  });

  it('renders a policy explanation block when payload carries policy metadata', () => {
    const { getByTestId, getAllByTestId } = renderCard({
      approval: {
        ...baseApproval,
        payload: {
          policy_id: 'pol-9',
          policy_type: 'approval_gate',
          decision: 'requires_human_approval',
          reason_codes: ['quota_high', 'sensitive_action'],
        },
      },
    });

    expect(getByTestId('approval-card-policy')).toBeTruthy();
    expect(getAllByTestId('approval-card-policy-line').map((node) => node.props.children)).toEqual([
      'Type: approval_gate',
      'Policy: pol-9',
      'Decision: requires_human_approval',
      'Rules: quota_high, sensitive_action',
    ]);
  });

  it('does not render a policy explanation block when payload has no policy details', () => {
    const { queryByTestId } = renderCard({
      approval: { ...baseApproval, payload: { unrelated: true } },
    });
    expect(queryByTestId('approval-card-policy')).toBeNull();
  });

  it('shows countdown when not expired', () => {
    const { getByTestId } = renderCard();
    const countdown = getByTestId('approval-card-countdown');
    expect(countdown.props.children).toContain('Expires in');
  });

  it('shows Expired status and hides action buttons when past expiry', () => {
    const { getByTestId, queryByTestId } = renderCard({
      approval: { ...baseApproval, expiresAt: pastExpiry },
    });
    expect(getByTestId('approval-card-countdown').props.children).toBe('Expired');
    expect(queryByTestId('approval-card-approve')).toBeNull();
    expect(queryByTestId('approval-card-reject')).toBeNull();
  });

  it('opens approve dialog when approve is pressed', () => {
    const { getByTestId } = renderCard();
    fireEvent.press(getByTestId('approval-card-approve'));
    expect(getByTestId('approval-card-approve-dialog')).toBeTruthy();
  });

  it('calls onApprove with undefined comment when approved without one', () => {
    const { getByTestId, onApprove } = renderCard();
    fireEvent.press(getByTestId('approval-card-approve'));
    fireEvent.press(getByTestId('approval-card-approve-submit'));
    expect(onApprove).toHaveBeenCalledWith('apr-1', undefined);
  });

  it('calls onApprove with a trimmed optional comment when submitted', () => {
    const { getByTestId, onApprove } = renderCard();
    fireEvent.press(getByTestId('approval-card-approve'));
    fireEvent.changeText(getByTestId('approval-card-approve-comment-input'), '  Ready to send  ');
    fireEvent.press(getByTestId('approval-card-approve-submit'));
    expect(onApprove).toHaveBeenCalledWith('apr-1', 'Ready to send');
  });

  it('opens reject dialog when reject is pressed', () => {
    const { getByTestId } = renderCard();
    fireEvent.press(getByTestId('approval-card-reject'));
    expect(getByTestId('approval-card-reject-dialog')).toBeTruthy();
  });

  it('submit button is disabled when reason is empty', () => {
    const { getByTestId } = renderCard();
    fireEvent.press(getByTestId('approval-card-reject'));
    const submitBtn = getByTestId('approval-card-reject-submit');
    expect(submitBtn.props.accessibilityState?.disabled).toBe(true);
  });

  it('calls onReject with id and trimmed reason when submitted', () => {
    const { getByTestId, onReject } = renderCard();
    fireEvent.press(getByTestId('approval-card-reject'));
    fireEvent.changeText(getByTestId('approval-card-reject-reason-input'), '  Not authorized  ');
    fireEvent.press(getByTestId('approval-card-reject-submit'));
    expect(onReject).toHaveBeenCalledWith('apr-1', 'Not authorized');
  });

  it('does not call onReject when dialog is cancelled', () => {
    const { getByTestId, onReject } = renderCard();
    fireEvent.press(getByTestId('approval-card-reject'));
    fireEvent.press(getByTestId('approval-card-reject-cancel'));
    expect(onReject).not.toHaveBeenCalled();
  });

  it('renders post-decision feedback, terminal path, and audit link when decision feedback exists', () => {
    const { getByTestId, queryByTestId } = renderCard({
      decisionFeedback: {
        kind: 'success',
        title: 'Approval recorded',
        body: 'The server accepted this approval decision and the flow is now complete.',
        visibleStatus: 'approved',
        comment: 'Ready to send',
        traceId: 'trace-42',
      },
    });

    expect(getByTestId('approval-card-countdown').props.children).toBe('Approved');
    expect(getByTestId('approval-card-feedback-title').props.children).toBe('Approval recorded');
    expect(getByTestId('approval-card-feedback-comment').props.children).toBe('Comment: Ready to send');
    expect(getByTestId('approval-card-feedback-trace').props.children).toBe('Trace: trace-42');
    expect(queryByTestId('approval-card-approve')).toBeNull();
    fireEvent.press(getByTestId('approval-card-audit-link'));
    expect(mockPush).toHaveBeenCalledWith({
      pathname: '/governance/audit',
      params: { trace_id: 'trace-42' },
    });
  });

  it('renders designed conflict feedback for expired approvals', () => {
    const { getByTestId, queryByTestId } = renderCard({
      decisionFeedback: {
        kind: 'conflict',
        title: 'Approval expired',
        body: 'This approval expired before your decision could be recorded. Review the audit trail for the final governed state.',
        visibleStatus: 'expired',
      },
    });

    expect(getByTestId('approval-card-countdown').props.children).toBe('Expired');
    expect(getByTestId('approval-card-feedback-title').props.children).toBe('Approval expired');
    expect(queryByTestId('approval-card-approve')).toBeNull();
  });

  it('hides approve/reject actions for an already-decided conflict even though the raw status is still pending', () => {
    const { getByTestId, queryByTestId } = renderCard({
      decisionFeedback: {
        kind: 'conflict',
        title: 'Approval already decided',
        body: 'Another operator or process already recorded a final decision for this request. Refresh context from the audit trail before acting again.',
      },
    });

    expect(getByTestId('approval-card-feedback-title').props.children).toBe('Approval already decided');
    expect(queryByTestId('approval-card-approve')).toBeNull();
    expect(queryByTestId('approval-card-reject')).toBeNull();
  });

  it('does not show a pending countdown or a Pending FSM step for an already-decided conflict', () => {
    const { getByTestId, queryByTestId } = renderCard({
      decisionFeedback: {
        kind: 'conflict',
        title: 'Approval already decided',
        body: 'Another operator or process already recorded a final decision for this request. Refresh context from the audit trail before acting again.',
      },
    });

    // No fabricated terminal status (approved/rejected/etc.) — neutral "Closed" cue instead,
    // and the FSM path is omitted entirely rather than showing a misleading active step.
    expect(getByTestId('approval-card-countdown').props.children).toBe('Closed');
    expect(queryByTestId('approval-card-path')).toBeNull();
  });
});
