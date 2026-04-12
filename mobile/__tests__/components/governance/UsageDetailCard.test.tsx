// Wave 1 — governance_mobile_enhancement_plan: UsageDetailCard TDD tests
// Tests written BEFORE implementation per TDD approach.

import React from 'react';
import { describe, it, expect, jest } from '@jest/globals';
import { render, fireEvent } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { UsageDetailCard } from '../../../src/components/governance/UsageDetailCard';
import type { UsageEvent } from '../../../src/services/api.types';

const baseEvent: UsageEvent = {
  id: 'evt-1',
  workspaceId: 'ws-1',
  actorType: 'agent',
  toolName: 'send_email',
  modelName: 'claude-sonnet-4-6',
  estimatedCost: 0.00123,
  latencyMs: 842,
  inputUnits: 120,
  outputUnits: 45,
  createdAt: '2026-04-12T10:00:00Z',
};

function renderCard(props?: Partial<Parameters<typeof UsageDetailCard>[0]>) {
  const onPress = jest.fn();
  const utils = render(
    <PaperProvider>
      <UsageDetailCard event={baseEvent} testIDPrefix="udc" onPress={onPress} {...props} />
    </PaperProvider>
  );
  return { ...utils, onPress };
}

describe('UsageDetailCard', () => {
  it('renders actor type badge', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('udc-actor-type').props.children).toBe('agent');
  });

  it('renders tool name as primary label', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('udc-tool-name').props.children).toBe('send_email');
  });

  it('renders fallback dash when toolName is undefined', () => {
    const { getByTestId } = renderCard({ event: { ...baseEvent, toolName: undefined } });
    expect(getByTestId('udc-tool-name').props.children).toBe('—');
  });

  it('renders model name', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('udc-model-name').props.children).toBe('claude-sonnet-4-6');
  });

  it('renders fallback dash when modelName is undefined', () => {
    const { getByTestId } = renderCard({ event: { ...baseEvent, modelName: undefined } });
    expect(getByTestId('udc-model-name').props.children).toBe('—');
  });

  it('renders estimated cost formatted with € prefix and 5 decimals', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('udc-cost').props.children).toBe('€0.00123');
  });

  it('renders fallback dash when estimatedCost is undefined', () => {
    const { getByTestId } = renderCard({ event: { ...baseEvent, estimatedCost: undefined } });
    expect(getByTestId('udc-cost').props.children).toBe('—');
  });

  it('renders latency in ms', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('udc-latency').props.children).toBe('842 ms');
  });

  it('renders fallback dash when latencyMs is undefined', () => {
    const { getByTestId } = renderCard({ event: { ...baseEvent, latencyMs: undefined } });
    expect(getByTestId('udc-latency').props.children).toBe('—');
  });

  it('renders createdAt as a non-empty string', () => {
    const { getByTestId } = renderCard();
    const text = getByTestId('udc-timestamp').props.children;
    expect(typeof text).toBe('string');
    expect(text.length).toBeGreaterThan(0);
  });

  it('calls onPress when card is pressed', () => {
    const { getByTestId, onPress } = renderCard();
    fireEvent.press(getByTestId('udc-card'));
    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it('renders without crashing when onPress is not provided', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <UsageDetailCard event={baseEvent} testIDPrefix="udc" />
      </PaperProvider>
    );
    expect(getByTestId('udc-card')).toBeTruthy();
  });
});
