// HandoffBanner — escalated run banner, reason/evidence/context, Accept Handoff navigation
// FR-232, UC-A7: human handoff display


import React from 'react';
import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { render, screen, fireEvent } from '@testing-library/react-native';
import { HandoffBanner } from '../../../src/components/agents/HandoffBanner';
import type { HandoffPackage } from '../../../src/services/api';

const mockPush = jest.fn();

jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockPush }),
}));

const mockUseHandoffPackage = jest.fn();

jest.mock('../../../src/hooks/useAgentSpec', () => ({
  useHandoffPackage: (...args: unknown[]) => mockUseHandoffPackage(...args),
}));

const baseHandoff: HandoffPackage = {
  run_id: 'run-1',
  reason: 'Agent could not resolve the case automatically',
  conversation_context: 'Customer reported billing issue for 3 months',
  evidence_count: 4,
  entity_type: 'account',
  entity_id: 'acc-1',
  created_at: '2026-03-15T10:00:00Z',
};

describe('HandoffBanner', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders nothing when handoff data is null', () => {
    mockUseHandoffPackage.mockReturnValue({ data: null, isLoading: false });
    const { queryByTestId } = render(<HandoffBanner runId="run-1" />);
    expect(queryByTestId('handoff-banner')).toBeNull();
  });

  it('renders nothing when handoff data is undefined', () => {
    mockUseHandoffPackage.mockReturnValue({ data: undefined, isLoading: false });
    const { queryByTestId } = render(<HandoffBanner runId="run-1" />);
    expect(queryByTestId('handoff-banner')).toBeNull();
  });

  it('shows loading indicator while fetching', () => {
    mockUseHandoffPackage.mockReturnValue({ data: undefined, isLoading: true });
    const { getByTestId } = render(<HandoffBanner runId="run-1" />);
    expect(getByTestId('handoff-banner-loading')).toBeTruthy();
  });

  it('renders banner with reason when handoff data is available', () => {
    mockUseHandoffPackage.mockReturnValue({ data: baseHandoff, isLoading: false });
    const { getByTestId } = render(<HandoffBanner runId="run-1" />);
    expect(getByTestId('handoff-banner')).toBeTruthy();
    expect(getByTestId('handoff-banner-reason').props.children).toBe(
      'Agent could not resolve the case automatically',
    );
  });

  it('shows conversation context preview', () => {
    mockUseHandoffPackage.mockReturnValue({ data: baseHandoff, isLoading: false });
    const { getByTestId } = render(<HandoffBanner runId="run-1" />);
    expect(getByTestId('handoff-banner-context').props.children).toBe(
      'Customer reported billing issue for 3 months',
    );
  });

  it('shows evidence count with correct pluralization', () => {
    mockUseHandoffPackage.mockReturnValue({ data: baseHandoff, isLoading: false });
    const { getByTestId } = render(<HandoffBanner runId="run-1" />);
    const children = getByTestId('handoff-banner-evidence-count').props.children;
    const text = Array.isArray(children) ? children.join('') : String(children);
    expect(text).toContain('4');
    expect(text).toContain('item');
  });

  it('shows singular "item" when evidence_count is 1', () => {
    mockUseHandoffPackage.mockReturnValue({
      data: { ...baseHandoff, evidence_count: 1 },
      isLoading: false,
    });
    const { getByTestId } = render(<HandoffBanner runId="run-1" />);
    const children = getByTestId('handoff-banner-evidence-count').props.children;
    const text = Array.isArray(children) ? children.join('') : String(children);
    expect(text).toContain('1');
    expect(text).not.toContain('items');
  });

  it('navigates to entity detail on Accept Handoff when entity context is present', () => {
    mockUseHandoffPackage.mockReturnValue({ data: baseHandoff, isLoading: false });
    const { getByTestId } = render(<HandoffBanner runId="run-1" />);
    fireEvent.press(getByTestId('handoff-banner-accept'));
    expect(mockPush).toHaveBeenCalledWith('/(tabs)/crm/accounts/acc-1');
  });

  it('navigates to copilot when entity context is absent', () => {
    const noEntityHandoff: HandoffPackage = {
      ...baseHandoff,
      entity_type: undefined,
      entity_id: undefined,
    };
    mockUseHandoffPackage.mockReturnValue({ data: noEntityHandoff, isLoading: false });
    const { getByTestId } = render(<HandoffBanner runId="run-1" />);
    fireEvent.press(getByTestId('handoff-banner-accept'));
    expect(mockPush).toHaveBeenCalledWith('/(tabs)/copilot');
  });

  it('calls useHandoffPackage with runId and enabled=true', () => {
    mockUseHandoffPackage.mockReturnValue({ data: null, isLoading: false });
    render(<HandoffBanner runId="run-42" />);
    expect(mockUseHandoffPackage).toHaveBeenCalledWith('run-42', true);
  });

  it('uses custom testIDPrefix', () => {
    mockUseHandoffPackage.mockReturnValue({ data: baseHandoff, isLoading: false });
    const { getByTestId } = render(
      <HandoffBanner runId="run-1" testIDPrefix="my-handoff" />,
    );
    expect(getByTestId('my-handoff')).toBeTruthy();
    expect(getByTestId('my-handoff-reason')).toBeTruthy();
    expect(getByTestId('my-handoff-accept')).toBeTruthy();
  });

  it('does not show context element when conversation_context is empty', () => {
    mockUseHandoffPackage.mockReturnValue({
      data: { ...baseHandoff, conversation_context: '' },
      isLoading: false,
    });
    const { queryByTestId } = render(<HandoffBanner runId="run-1" />);
    expect(queryByTestId('handoff-banner-context')).toBeNull();
  });
});