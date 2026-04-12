// HomeFeed — unified feed, filter chips, badge count, pull-to-refresh
// FR-300 (Home), UC-A5/A6: signals + approvals in unified feed


import React from 'react';
import { describe, it, expect, jest } from '@jest/globals';
import { render, fireEvent, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { HomeFeed } from '../../../src/components/home/HomeFeed';
import type { Signal, ApprovalRequest } from '../../../src/services/api';

const signal: Signal = {
  id: 'sig-1',
  workspace_id: 'ws-1',
  entity_type: 'deal',
  entity_id: 'd-1',
  signal_type: 'churn_risk',
  confidence: 0.9,
  evidence_ids: [],
  source_type: 'llm',
  source_id: 'run-1',
  metadata: {},
  status: 'active',
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

const approval: ApprovalRequest = {
  id: 'apr-1',
  workspace_id: 'ws-1',
  requested_by: 'user-1',
  approver_id: 'user-2',
  action: 'send_email',
  payload: {},
  status: 'pending',
  expiresAt: new Date(Date.now() + 3_600_000).toISOString(),
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

const defaultProps = {
  signals: [signal],
  approvals: [approval],
  loadingSignals: false,
  loadingApprovals: false,
  onRefresh: jest.fn(),
  onDismissSignal: jest.fn(),
  onApprove: jest.fn(),
  onReject: jest.fn(),
};

function renderFeed(props?: Partial<typeof defaultProps>) {
  return render(
    <PaperProvider>
      <HomeFeed {...defaultProps} {...props} />
    </PaperProvider>
  );
}

describe('HomeFeed', () => {
  it('renders all chips', () => {
    const { getByTestId } = renderFeed();
    expect(getByTestId('home-feed-chip-all')).toBeTruthy();
    expect(getByTestId('home-feed-chip-signals')).toBeTruthy();
    expect(getByTestId('home-feed-chip-approvals')).toBeTruthy();
  });

  it('shows both signal and approval cards in All mode', () => {
    const { getByTestId } = renderFeed();
    expect(getByTestId(`home-feed-signal-${signal.id}`)).toBeTruthy();
    expect(getByTestId(`home-feed-approval-${approval.id}`)).toBeTruthy();
  });

  it('hides approvals when Signals chip is selected', () => {
    const { getByTestId, queryByTestId } = renderFeed();
    fireEvent.press(getByTestId('home-feed-chip-signals'));
    expect(getByTestId(`home-feed-signal-${signal.id}`)).toBeTruthy();
    expect(queryByTestId(`home-feed-approval-${approval.id}`)).toBeNull();
  });

  it('hides signals when Approvals chip is selected', () => {
    const { getByTestId, queryByTestId } = renderFeed();
    fireEvent.press(getByTestId('home-feed-chip-approvals'));
    expect(queryByTestId(`home-feed-signal-${signal.id}`)).toBeNull();
    expect(getByTestId(`home-feed-approval-${approval.id}`)).toBeTruthy();
  });

  it('shows pending count badge in Approvals chip when count > 0', () => {
    const { getByTestId } = renderFeed({ pendingApprovalCount: 3 });
    expect(within(getByTestId('home-feed-chip-approvals')).getByText(/3/)).toBeTruthy();
  });

  it('does not show badge when pendingApprovalCount is 0', () => {
    const { getByTestId } = renderFeed({ pendingApprovalCount: 0 });
    expect(within(getByTestId('home-feed-chip-approvals')).getByText('Approvals')).toBeTruthy();
  });

  it('shows empty state when no items', () => {
    const { getByTestId } = renderFeed({ signals: [], approvals: [] });
    expect(getByTestId('home-feed-empty')).toBeTruthy();
  });

  it('shows Loading… in empty state when refreshing', () => {
    const { getByTestId } = renderFeed({ signals: [], approvals: [], loadingSignals: true });
    expect(getByTestId('home-feed-empty').props.children).toBe('Loading…');
  });
});
