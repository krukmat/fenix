// CopilotPanel — signal context banner, context forwarding, backward compatibility
// FR-200 (Copilot embedded), UC-A5: signal-aware copilot context

import React from 'react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render, fireEvent, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';

// ─── Mocks ───────────────────────────────────────────────────────────────────

const mockSendQuery = jest.fn();
const mockUseSSE = jest.fn();

jest.mock('../../../src/hooks/useSSE', () => ({
  useSSE: () => mockUseSSE(),
}));

jest.mock('../../../src/services/api', () => ({
  toolApi: { execute: jest.fn() },
}));

// ─── Helpers ─────────────────────────────────────────────────────────────────

function setupSSE() {
  mockUseSSE.mockReturnValue({
    messages: [],
    isStreaming: false,
    error: null,
    sendQuery: mockSendQuery,
    clearMessages: jest.fn(),
  });
}

function renderPanel(props?: Record<string, unknown>) {
  const { CopilotPanel } = require('../../../src/components/copilot/CopilotPanel');
  return render(
    <PaperProvider>
      <CopilotPanel {...props} />
    </PaperProvider>
  );
}

// ─── Tests ────────────────────────────────────────────────────────────────────

describe('CopilotPanel — backward compatibility (no context)', () => {
  beforeEach(() => { jest.clearAllMocks(); setupSSE(); });

  it('renders without initialContext prop', () => {
    const { getByTestId } = renderPanel();
    expect(getByTestId('copilot-panel')).toBeTruthy();
  });

  it('does NOT render context banner when initialContext is undefined', () => {
    const { queryByTestId } = renderPanel();
    expect(queryByTestId('copilot-context-banner')).toBeNull();
  });

  it('sends query without context when no initialContext', () => {
    const { getByTestId } = renderPanel();
    fireEvent.changeText(getByTestId('copilot-input'), 'Hello');
    fireEvent.press(getByTestId('copilot-send'));
    expect(mockSendQuery).toHaveBeenCalledWith('Hello', undefined);
  });
});

describe('CopilotPanel — signal context banner', () => {
  beforeEach(() => { jest.clearAllMocks(); setupSSE(); });

  it('shows context banner when initialContext has signalType', () => {
    const { getByTestId } = renderPanel({
      initialContext: { signalType: 'churn_risk', entityType: 'deal', entityId: 'd-1' },
    });
    expect(getByTestId('copilot-context-banner')).toBeTruthy();
  });

  it('banner text includes signal type', () => {
    const { getByTestId } = renderPanel({
      initialContext: { signalType: 'churn_risk' },
    });
    expect(within(getByTestId('copilot-context-banner')).getByText(/churn_risk/)).toBeTruthy();
  });

  it('banner text includes entity type and id', () => {
    const { getByTestId } = renderPanel({
      initialContext: { entityType: 'deal', entityId: 'd-99' },
    });
    expect(within(getByTestId('copilot-context-banner')).getByText(/deal/)).toBeTruthy();
    expect(within(getByTestId('copilot-context-banner')).getByText(/d-99/)).toBeTruthy();
  });

  it('does NOT show banner when initialContext is empty object', () => {
    const { queryByTestId } = renderPanel({ initialContext: {} });
    expect(queryByTestId('copilot-context-banner')).toBeNull();
  });
});

describe('CopilotPanel — context forwarding to sendQuery', () => {
  beforeEach(() => { jest.clearAllMocks(); setupSSE(); });

  it('forwards full signal context when sending a message', () => {
    const ctx = {
      signalId: 'sig-1',
      signalType: 'churn_risk',
      entityType: 'deal',
      entityId: 'd-1',
    };
    const { getByTestId } = renderPanel({ initialContext: ctx });
    fireEvent.changeText(getByTestId('copilot-input'), 'Why is this a risk?');
    fireEvent.press(getByTestId('copilot-send'));

    expect(mockSendQuery).toHaveBeenCalledWith('Why is this a risk?', {
      signalId: 'sig-1',
      signalType: 'churn_risk',
      entityType: 'deal',
      entityId: 'd-1',
    });
  });

  it('forwards partial context (entity only, no signal)', () => {
    const { getByTestId } = renderPanel({
      initialContext: { entityType: 'account', entityId: 'a-5' },
    });
    fireEvent.changeText(getByTestId('copilot-input'), 'Summarize this account');
    fireEvent.press(getByTestId('copilot-send'));

    expect(mockSendQuery).toHaveBeenCalledWith(
      'Summarize this account',
      expect.objectContaining({ entityType: 'account', entityId: 'a-5' })
    );
  });

  it('does not include undefined fields in context', () => {
    const { getByTestId } = renderPanel({
      initialContext: { entityType: 'deal', entityId: 'd-2' },
    });
    fireEvent.changeText(getByTestId('copilot-input'), 'test');
    fireEvent.press(getByTestId('copilot-send'));

    const ctx = mockSendQuery.mock.calls[0][1] as Record<string, unknown>;
    expect(ctx.signalId).toBeUndefined();
    expect(ctx.signalType).toBeUndefined();
  });
});

describe('CopilotPanel — context is stateless (derived from prop on each send)', () => {
  beforeEach(() => { jest.clearAllMocks(); setupSSE(); });

  it('both sends include the same context from prop', () => {
    const { getByTestId } = renderPanel({
      initialContext: { signalId: 'sig-1', signalType: 'risk' },
    });
    fireEvent.changeText(getByTestId('copilot-input'), 'first');
    fireEvent.press(getByTestId('copilot-send'));
    fireEvent.changeText(getByTestId('copilot-input'), 'second');
    fireEvent.press(getByTestId('copilot-send'));

    expect(mockSendQuery).toHaveBeenCalledTimes(2);
    const ctx1 = mockSendQuery.mock.calls[0][1] as Record<string, unknown>;
    const ctx2 = mockSendQuery.mock.calls[1][1] as Record<string, unknown>;
    expect(ctx1.signalId).toBe('sig-1');
    expect(ctx2.signalId).toBe('sig-1');
  });
});
