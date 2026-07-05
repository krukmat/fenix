import React from 'react';
import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { render, fireEvent } from '@testing-library/react-native';
import { PaperProvider } from 'react-native-paper';

import { CopilotPanel } from '../../src/components/copilot/CopilotPanel';

const mockUseSSE = jest.fn();

jest.mock('../../src/hooks/useSSE', () => ({
  useSSE: () => mockUseSSE(),
}));

jest.mock('../../src/services/api', () => ({
  toolApi: {
    execute: jest.fn(async () => ({ ok: true })),
  },
}));

describe('CopilotPanel', () => {
  const sendQuery = jest.fn();
  const onSupportTrigger = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
    mockUseSSE.mockReturnValue({
      messages: [
        { id: 'u1', role: 'user', content: 'hola' },
        { id: 'a1', role: 'assistant', content: 'respuesta' },
      ],
      isStreaming: false,
      error: null,
      sendQuery,
      clearMessages: jest.fn(),
    });
  });

  const wrap = (props?: React.ComponentProps<typeof CopilotPanel>) => (
    render(
      <PaperProvider>
        <CopilotPanel {...props} />
      </PaperProvider>
    )
  );

  it('renders user + assistant messages', () => {
    const { getByText, getAllByText } = wrap();
    expect(getByText('hola')).toBeTruthy();
    expect(getAllByText('respuesta').length).toBeGreaterThan(0);
  });

  it('send disabled when input is empty', () => {
    const { getByTestId } = wrap();
    expect(getByTestId('copilot-send').props.accessibilityState.disabled).toBe(true);
  });

  it('types query and sends, then clears input', () => {
    const { getByTestId } = wrap();
    const input = getByTestId('copilot-input');

    fireEvent.changeText(input, 'nuevo prompt');
    fireEvent.press(getByTestId('copilot-send'));

    expect(sendQuery).toHaveBeenCalledWith('nuevo prompt', undefined);
    expect(getByTestId('copilot-input').props.value).toBe('');
  });

  it('keeps onSupportTrigger wired when sending', () => {
    const { getByTestId } = wrap({ onSupportTrigger });

    fireEvent.changeText(getByTestId('copilot-input'), 'nuevo prompt');
    fireEvent.press(getByTestId('copilot-send'));

    expect(onSupportTrigger).toHaveBeenCalledWith('nuevo prompt');
  });

  it('shows streaming indicator', () => {
    mockUseSSE.mockReturnValueOnce({
      messages: [],
      isStreaming: true,
      error: null,
      sendQuery,
      clearMessages: jest.fn(),
    });

    const { getByTestId } = wrap();
    expect(getByTestId('copilot-streaming')).toBeTruthy();
  });

  it('shows error message', () => {
    mockUseSSE.mockReturnValueOnce({
      messages: [],
      isStreaming: false,
      error: 'boom',
      sendQuery,
      clearMessages: jest.fn(),
    });

    const { getByTestId, getByText } = wrap();
    expect(getByTestId('copilot-error')).toBeTruthy();
    expect(getByText('boom')).toBeTruthy();
  });

  it('renders a confidence badge and warnings from the assistant trust metadata', () => {
    mockUseSSE.mockReturnValueOnce({
      messages: [
        {
          id: 'a1',
          role: 'assistant',
          content: 'respuesta',
          confidence: 'high',
          warnings: ['1 items stale'],
        },
      ],
      isStreaming: false,
      error: null,
      sendQuery,
      clearMessages: jest.fn(),
    });

    const { getByTestId, getByText } = wrap();
    expect(getByTestId('copilot-confidence-badge')).toBeTruthy();
    expect(getByText('High confidence')).toBeTruthy();
    expect(getByTestId('copilot-warnings-row')).toBeTruthy();
    expect(getByText('1 items stale')).toBeTruthy();
  });

  it('renders abstention as a designed panel instead of an empty assistant response', () => {
    mockUseSSE.mockReturnValueOnce({
      messages: [
        {
          id: 'a1',
          role: 'assistant',
          content: '',
          answerType: 'abstention',
          abstentionReason: 'insufficient_evidence',
          warnings: ['No recent evidence'],
          evidenceSources: [
            { id: 'e1', snippet: 'Partial snippet', score: 0.4, timestamp: '2026-07-05T10:00:00Z' },
          ],
        },
      ],
      isStreaming: false,
      error: null,
      sendQuery,
      clearMessages: jest.fn(),
    });

    const { getByTestId, getByText, queryByTestId } = wrap();
    expect(getByTestId('copilot-abstention-panel')).toBeTruthy();
    expect(getByText(/evidence was not strong enough/i)).toBeTruthy();
    expect(getByTestId('copilot-abstention-manual-lane')).toBeTruthy();
    expect(queryByTestId('copilot-confidence-badge')).toBeNull();
    expect(queryByTestId('copilot-message-a1')).toBeNull();
    expect(getByTestId('copilot-warnings-row')).toBeTruthy();
    expect(getByTestId('evidence-card-0')).toBeTruthy();
  });
});
