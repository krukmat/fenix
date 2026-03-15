// SignalCard — confidence badge, dismiss dialog, onPress
// UC-A5/B4: signal display in feeds and entity details


import React from 'react';
import { describe, it, expect, jest } from '@jest/globals';
import { render, fireEvent, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { SignalCard } from '../../../src/components/signals/SignalCard';
import type { Signal } from '../../../src/services/api';

const baseSignal: Signal = {
  id: 'sig-1',
  workspace_id: 'ws-1',
  entity_type: 'deal',
  entity_id: 'd-1',
  signal_type: 'churn_risk',
  confidence: 0.85,
  evidence_ids: ['e-1', 'e-2'],
  source_type: 'llm',
  source_id: 'run-1',
  metadata: { summary: 'Customer shows churn indicators' },
  status: 'active',
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

function renderCard(props?: Partial<Parameters<typeof SignalCard>[0]>) {
  const onDismiss = jest.fn();
  const utils = render(
    <PaperProvider>
      <SignalCard signal={baseSignal} onDismiss={onDismiss} {...props} />
    </PaperProvider>
  );
  return { ...utils, onDismiss };
}

describe('SignalCard', () => {
  it('renders signal type and entity', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('signal-card-type').props.children).toBe('churn_risk');
    expect(getByTestId('signal-card-entity').props.children).toBe('deal · d-1');
  });

  it('renders high confidence badge (≥0.8)', () => {
    const { getByTestId } = renderCard();
    expect(within(getByTestId('signal-card-confidence')).getByText(/High/)).toBeTruthy();
  });

  it('renders medium confidence badge (0.5–0.79)', () => {
    const { getByTestId } = renderCard({ signal: { ...baseSignal, confidence: 0.65 } });
    expect(within(getByTestId('signal-card-confidence')).getByText(/Medium/)).toBeTruthy();
  });

  it('renders low confidence badge (<0.5)', () => {
    const { getByTestId } = renderCard({ signal: { ...baseSignal, confidence: 0.3 } });
    expect(within(getByTestId('signal-card-confidence')).getByText(/Low/)).toBeTruthy();
  });

  it('renders metadata summary as snippet when present', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('signal-card-snippet').props.children).toBe('Customer shows churn indicators');
  });

  it('falls back to signal_type when metadata.summary is absent', () => {
    const { getByTestId } = renderCard({ signal: { ...baseSignal, metadata: {} } });
    expect(getByTestId('signal-card-snippet').props.children).toBe('churn_risk');
  });

  it('opens dismiss dialog when dismiss button is pressed', () => {
    const { getByTestId } = renderCard();
    fireEvent.press(getByTestId('signal-card-dismiss-btn'));
    expect(getByTestId('signal-card-dismiss-dialog')).toBeTruthy();
  });

  it('calls onDismiss with signal id when confirmed', () => {
    const { getByTestId, onDismiss } = renderCard();
    fireEvent.press(getByTestId('signal-card-dismiss-btn'));
    fireEvent.press(getByTestId('signal-card-dismiss-confirm'));
    expect(onDismiss).toHaveBeenCalledWith('sig-1');
  });

  it('does not call onDismiss when dialog is cancelled', () => {
    const { getByTestId, onDismiss } = renderCard();
    fireEvent.press(getByTestId('signal-card-dismiss-btn'));
    fireEvent.press(getByTestId('signal-card-dismiss-cancel'));
    expect(onDismiss).not.toHaveBeenCalled();
  });

  it('calls onPress when card is pressed', () => {
    const onPress = jest.fn();
    const { getByTestId } = renderCard({ onPress });
    fireEvent.press(getByTestId('signal-card'));
    expect(onPress).toHaveBeenCalledWith(baseSignal);
  });
});