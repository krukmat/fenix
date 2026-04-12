import React from 'react';
import { describe, expect, it, jest } from '@jest/globals';
import { fireEvent, render } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { AuditEventCard } from '../../../src/components/governance/AuditEventCard';
import type { AuditEvent } from '../../../src/services/api.types';

const baseEvent: AuditEvent = {
  id: 'audit-1',
  workspace_id: 'ws-1',
  actor_id: 'user-1',
  actor_type: 'user',
  action: 'case.updated',
  entity_type: 'case',
  entity_id: 'case-1',
  details: { field: 'status', from: 'open', to: 'closed' },
  outcome: 'success',
  trace_id: 'trace-1',
  created_at: '2026-04-12T10:00:00Z',
};

function renderCard(props?: Partial<Parameters<typeof AuditEventCard>[0]>) {
  const onPress = jest.fn();
  const utils = render(
    <PaperProvider>
      <AuditEventCard event={baseEvent} expanded={false} onPress={onPress} testIDPrefix="aec" {...props} />
    </PaperProvider>
  );
  return { ...utils, onPress };
}

describe('AuditEventCard', () => {
  it('renders the action as title', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('aec-title').props.children).toBe('case.updated');
  });

  it('renders the outcome badge with success color', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('aec-outcome-badge').props.style).toEqual(
      expect.arrayContaining([expect.objectContaining({ backgroundColor: '#10B981' })])
    );
  });

  it('hides expanded fields when collapsed', () => {
    const { queryByTestId } = renderCard();
    expect(queryByTestId('aec-entity')).toBeNull();
    expect(queryByTestId('aec-trace-id')).toBeNull();
  });

  it('shows entity, trace and details when expanded', () => {
    const { getByTestId } = renderCard({ expanded: true });
    expect(getByTestId('aec-entity').props.children.join('')).toContain('case');
    expect(getByTestId('aec-trace-id').props.children.join('')).toContain('trace-1');
    expect(getByTestId('aec-details').props.children).toContain('"field": "status"');
  });

  it('calls onPress when tapped', () => {
    const { getByTestId, onPress } = renderCard();
    fireEvent.press(getByTestId('aec-card'));
    expect(onPress).toHaveBeenCalledTimes(1);
  });
});
