// ApprovalCard — countdown, approve/reject flow, expired state
// FR-071 (Approvals), UC-A6: human approval decision


import React from 'react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render, fireEvent } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { ApprovalCard } from '../../../src/components/approvals/ApprovalCard';
import type { ApprovalRequest } from '../../../src/services/api';

const futureExpiry = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(); // 2h from now
const pastExpiry = new Date(Date.now() - 60_000).toISOString(); // 1 min ago

const baseApproval: ApprovalRequest = {
  id: 'apr-1',
  workspace_id: 'ws-1',
  requested_by: 'user-1',
  approver_id: 'user-2',
  action: 'send_email',
  resource_type: 'contact',
  resource_id: 'c-1',
  payload: {},
  reason: 'Customer requested follow-up',
  status: 'pending',
  expiresAt: futureExpiry,
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
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
  it('renders action and resource', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('approval-card-action').props.children).toBe('send_email');
    expect(getByTestId('approval-card-resource').props.children).toBe('contact · c-1');
  });

  it('renders reason when present', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('approval-card-reason').props.children).toBe('Customer requested follow-up');
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

  it('calls onApprove with approval id when approve is pressed', () => {
    const { getByTestId, onApprove } = renderCard();
    fireEvent.press(getByTestId('approval-card-approve'));
    expect(onApprove).toHaveBeenCalledWith('apr-1');
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
});
