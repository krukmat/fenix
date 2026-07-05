import React from 'react';
import { describe, it, expect } from '@jest/globals';
import { render } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { ApprovalPath } from '../../../src/components/approvals/ApprovalPath';

function renderPath(status: Parameters<typeof ApprovalPath>[0]['status']) {
  return render(
    <PaperProvider>
      <ApprovalPath status={status} />
    </PaperProvider>
  );
}

describe('ApprovalPath', () => {
  it('renders the full five-state approval path', () => {
    const { getByTestId } = renderPath('pending');

    expect(getByTestId('approval-path-pending-label').props.children).toBe('Pending');
    expect(getByTestId('approval-path-approved-label').props.children).toBe('Approved');
    expect(getByTestId('approval-path-rejected-label').props.children).toBe('Rejected');
    expect(getByTestId('approval-path-expired-label').props.children).toBe('Expired');
    expect(getByTestId('approval-path-cancelled-label').props.children).toBe('Cancelled');
  });

  it('normalizes legacy denied to rejected for display', () => {
    const { getByTestId } = renderPath('denied');

    expect(getByTestId('approval-path-rejected-dot').props.style).toEqual(
      expect.arrayContaining([
        expect.objectContaining({ backgroundColor: '#F59E0B' }),
      ])
    );
  });

  it('highlights the active state and leaves other terminals dimmed', () => {
    const { getByTestId } = renderPath('approved');
    const approvedDot = getByTestId('approval-path-approved-dot');
    const expiredDot = getByTestId('approval-path-expired-dot');

    expect(approvedDot.props.style).toEqual(
      expect.arrayContaining([expect.objectContaining({ backgroundColor: '#10B981' })])
    );
    expect(expiredDot.props.style).toEqual(
      expect.arrayContaining([expect.objectContaining({ backgroundColor: 'rgba(231, 224, 236, 1)' })])
    );
  });
});
