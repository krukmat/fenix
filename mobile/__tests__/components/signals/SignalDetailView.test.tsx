// SignalDetailView — full signal view, evidence list, empty evidence fallback
// UC-A5/B4.3: signal detail with evidence


import React from 'react';
import { describe, it, expect } from '@jest/globals';
import { render, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { SignalDetailView } from '../../../src/components/signals/SignalDetailView';
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

function renderView(signal: Signal = baseSignal) {
  return render(
    <PaperProvider>
      <SignalDetailView signal={signal} testIDPrefix="signal-detail" />
    </PaperProvider>
  );
}

describe('SignalDetailView', () => {
  it('renders signal type and entity', () => {
    const { getByTestId } = renderView();
    expect(getByTestId('signal-detail-type').props.children).toBe('churn_risk');
    expect(getByTestId('signal-detail-entity').props.children).toBe('deal · d-1');
  });

  it('renders high confidence badge', () => {
    const { getByTestId } = renderView();
    expect(within(getByTestId('signal-detail-confidence')).getByText(/High/)).toBeTruthy();
  });

  it('renders medium confidence badge (0.5–0.79)', () => {
    const { getByTestId } = renderView({ ...baseSignal, confidence: 0.6 });
    expect(within(getByTestId('signal-detail-confidence')).getByText(/Medium/)).toBeTruthy();
  });

  it('renders low confidence badge (<0.5)', () => {
    const { getByTestId } = renderView({ ...baseSignal, confidence: 0.3 });
    expect(within(getByTestId('signal-detail-confidence')).getByText(/Low/)).toBeTruthy();
  });

  it('renders metadata summary when present', () => {
    const { getByTestId } = renderView();
    expect(getByTestId('signal-detail-summary').props.children).toBe('Customer shows churn indicators');
  });

  it('does not render summary section when metadata.summary is absent', () => {
    const { queryByTestId } = renderView({ ...baseSignal, metadata: {} });
    expect(queryByTestId('signal-detail-summary')).toBeNull();
  });

  it('renders one EvidenceCard per evidence_id', () => {
    const { getByTestId } = renderView();
    expect(getByTestId('signal-detail-evidence-0')).toBeTruthy();
    expect(getByTestId('signal-detail-evidence-1')).toBeTruthy();
  });

  it('shows no-evidence message when evidence_ids is empty', () => {
    const { getByTestId } = renderView({ ...baseSignal, evidence_ids: [] });
    expect(getByTestId('signal-detail-no-evidence')).toBeTruthy();
  });
});