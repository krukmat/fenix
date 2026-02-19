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

  const wrap = () => render(<PaperProvider><CopilotPanel /></PaperProvider>);

  it('renders user + assistant messages', () => {
    const { getByText } = wrap();
    expect(getByText('hola')).toBeTruthy();
    expect(getByText('respuesta')).toBeTruthy();
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

    expect(sendQuery).toHaveBeenCalledWith('nuevo prompt');
    expect(getByTestId('copilot-input').props.value).toBe('');
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
});
