// EntitySignalsSection — signals in entity detail, dismiss, Ask Copilot, graceful degradation
// UC-A5/B4: signals visible in entity detail, B4.3: stale evidence


import React from 'react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render, fireEvent, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { EntitySignalsSection } from '../../../src/components/signals/EntitySignalsSection';
import type { Signal } from '../../../src/services/api';

// ─── Mocks ───────────────────────────────────────────────────────────────────

const mockUseSignalsByEntity = jest.fn();
const mockUseDismissSignal = jest.fn();
const mockPush = jest.fn();
const mockMutate = jest.fn();

jest.mock('../../../src/hooks/useAgentSpec', () => ({
  useSignalsByEntity: (...args: unknown[]) => mockUseSignalsByEntity(...args),
  useDismissSignal: () => mockUseDismissSignal(),
}));

jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockPush }),
}));

// ─── Fixtures ────────────────────────────────────────────────────────────────

const activeSignal: Signal = {
  id: 'sig-1',
  workspace_id: 'ws-1',
  entity_type: 'deal',
  entity_id: 'd-1',
  signal_type: 'churn_risk',
  confidence: 0.9,
  evidence_ids: ['e-1'],
  source_type: 'llm',
  source_id: 'r-1',
  metadata: { summary: 'Churn risk detected' },
  status: 'active',
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

const dismissedSignal: Signal = { ...activeSignal, id: 'sig-2', status: 'dismissed' };

function renderSection(props?: Partial<Parameters<typeof EntitySignalsSection>[0]>) {
  return render(
    <PaperProvider>
      <EntitySignalsSection entityType="deal" entityId="d-1" {...props} />
    </PaperProvider>
  );
}

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('EntitySignalsSection', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseDismissSignal.mockReturnValue({ mutate: mockMutate, isPending: false });
  });

  it('renders nothing when loading', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: undefined, isLoading: true });
    const { queryByTestId } = renderSection();
    expect(queryByTestId('entity-signals')).toBeNull();
  });

  it('renders nothing when there are no active signals', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [], isLoading: false });
    const { queryByTestId } = renderSection();
    expect(queryByTestId('entity-signals')).toBeNull();
  });

  it('renders nothing when signals only has dismissed/expired entries (B4.1 graceful)', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [dismissedSignal], isLoading: false });
    const { queryByTestId } = renderSection();
    expect(queryByTestId('entity-signals')).toBeNull();
  });

  it('renders section when there are active signals', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [activeSignal], isLoading: false });
    const { getByTestId } = renderSection();
    expect(getByTestId('entity-signals')).toBeTruthy();
  });

  it('shows heading with signal count', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [activeSignal], isLoading: false });
    const { getByTestId } = renderSection();
    expect(getByTestId('entity-signals-heading').props.children).toBe('Signals (1)');
  });

  it('shows correct count when multiple active signals', () => {
    const sig2 = { ...activeSignal, id: 'sig-3' };
    mockUseSignalsByEntity.mockReturnValue({ data: [activeSignal, sig2], isLoading: false });
    const { getByTestId } = renderSection();
    expect(getByTestId('entity-signals-heading').props.children).toBe('Signals (2)');
  });

  it('renders a SignalCard per active signal', () => {
    const sig2 = { ...activeSignal, id: 'sig-3' };
    mockUseSignalsByEntity.mockReturnValue({ data: [activeSignal, sig2], isLoading: false });
    const { getByTestId } = renderSection();
    expect(getByTestId(`entity-signals-card-${activeSignal.id}`)).toBeTruthy();
    expect(getByTestId(`entity-signals-card-${sig2.id}`)).toBeTruthy();
  });

  it('calls useDismissSignal mutate with signal id on dismiss', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [activeSignal], isLoading: false });
    const { getByTestId } = renderSection();
    fireEvent.press(getByTestId(`entity-signals-card-${activeSignal.id}-dismiss-btn`));
    fireEvent.press(getByTestId(`entity-signals-card-${activeSignal.id}-dismiss-confirm`));
    expect(mockMutate).toHaveBeenCalledWith(activeSignal.id);
  });

  it('renders Ask Copilot button', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [activeSignal], isLoading: false });
    const { getByTestId } = renderSection();
    expect(getByTestId('entity-signals-ask-copilot')).toBeTruthy();
  });

  it('Ask Copilot button label includes entity type', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [activeSignal], isLoading: false });
    const { getByTestId } = renderSection();
    expect(within(getByTestId('entity-signals-ask-copilot')).getByText(/deal/)).toBeTruthy();
  });

  it('navigates to copilot with entity context on Ask Copilot press', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [activeSignal], isLoading: false });
    const { getByTestId } = renderSection();
    fireEvent.press(getByTestId('entity-signals-ask-copilot'));
    expect(mockPush).toHaveBeenCalledWith({
      pathname: '/(tabs)/copilot',
      params: { entity_type: 'deal', entity_id: 'd-1' },
    });
  });

  it('passes entityType and entityId to useSignalsByEntity', () => {
    mockUseSignalsByEntity.mockReturnValue({ data: [], isLoading: false });
    renderSection({ entityType: 'account', entityId: 'a-99' });
    expect(mockUseSignalsByEntity).toHaveBeenCalledWith('account', 'a-99');
  });

  // B4.3 — stale evidence: component renders with empty evidence_ids (no crash)
  it('renders correctly when signal has empty evidence_ids (B4.3 graceful degradation)', () => {
    const noEvidence = { ...activeSignal, evidence_ids: [] };
    mockUseSignalsByEntity.mockReturnValue({ data: [noEvidence], isLoading: false });
    const { getByTestId } = renderSection();
    expect(getByTestId('entity-signals')).toBeTruthy();
  });
});