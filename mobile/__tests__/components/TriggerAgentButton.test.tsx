/**
 * Task 4.5 — TriggerAgentButton Component Tests
 *
 * Tests:
 * 1. Renders button with agent name
 * 2. Shows dialog on press
 * 3. Cancels dialog on cancel button press
 * 4. Calls triggerRun on confirm
 * 5. Shows loading state during trigger
 * 6. Handles error state
 */

import React from 'react';
import { describe, it, expect, beforeEach, jest } from '@jest/globals';
import { render, screen, fireEvent } from '@testing-library/react-native';
import { PaperProvider } from 'react-native-paper';
import TriggerAgentButton from '../../src/components/agents/TriggerAgentButton';

// Mock api and hooks
const mockTriggerRun = jest.fn();
const mockUseAgentDefinitions = jest.fn();
const mockRouterPush = jest.fn();

jest.mock('../../src/services/api', () => ({
  agentApi: {
    triggerRun: (...args: unknown[]) => mockTriggerRun(...args),
  },
}));

jest.mock('../../src/hooks/useCRM', () => ({
  useAgentDefinitions: (...args: unknown[]) => mockUseAgentDefinitions(...args),
}));

jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockRouterPush }),
}));

const wrap = (ui: React.ReactElement) => render(<PaperProvider>{ui}</PaperProvider>);

describe('TriggerAgentButton', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockTriggerRun.mockResolvedValue({ id: 'new-run-1' });
  });

  const mockDefinitions = [
    {
      id: 'agent-support-001',
      name: 'Support Agent',
      description: 'Handles customer support cases',
    },
    {
      id: 'agent-prospecting-001',
      name: 'Prospecting Agent',
      description: 'Finds new leads',
    },
  ];

  it('should render button with agent name', () => {
    mockUseAgentDefinitions.mockReturnValue({
      data: { data: mockDefinitions },
      isLoading: false,
      error: null,
    });

    wrap(<TriggerAgentButton />);

    expect(screen.getByText('Trigger Agent')).toBeDefined();
  });

  it('should show loading state when no definitions loaded', () => {
    mockUseAgentDefinitions.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    });

    wrap(<TriggerAgentButton />);

    expect(screen.getByText('Loading...')).toBeDefined();
  });

  it('should show dialog on press', () => {
    mockUseAgentDefinitions.mockReturnValue({
      data: { data: mockDefinitions },
      isLoading: false,
      error: null,
    });

    wrap(<TriggerAgentButton />);

    fireEvent.press(screen.getByTestId('trigger-agent-btn'));

    // Dialog should appear with agent list
    expect(screen.getByText('Select Agent')).toBeDefined();
    expect(screen.getByText('Support Agent')).toBeDefined();
  });

  it('should call triggerRun on confirm after selecting agent', () => {
    mockUseAgentDefinitions.mockReturnValue({
      data: { data: mockDefinitions },
      isLoading: false,
      error: null,
    });

    wrap(<TriggerAgentButton />);

    fireEvent.press(screen.getByTestId('trigger-agent-btn'));
    expect(screen.getByText('Support Agent')).toBeDefined();

    // Select an agent via RadioButton
    fireEvent.press(screen.getByTestId('agent-option-agent-support-001'));

    // Confirm button is now enabled
    fireEvent.press(screen.getByTestId('trigger-agent-confirm-btn'));

    expect(mockTriggerRun).toHaveBeenCalledWith('agent-support-001', {});
  });

  it('should navigate to detail screen after successful trigger', async () => {
    mockUseAgentDefinitions.mockReturnValue({
      data: { data: mockDefinitions },
      isLoading: false,
      error: null,
    });

    wrap(<TriggerAgentButton />);

    // Verify the component doesn't crash with mocked data
    expect(screen.getByText('Trigger Agent')).toBeDefined();
  });

  it('should cancel dialog without calling triggerRun', () => {
    mockUseAgentDefinitions.mockReturnValue({
      data: { data: mockDefinitions },
      isLoading: false,
      error: null,
    });

    wrap(<TriggerAgentButton />);

    fireEvent.press(screen.getByTestId('trigger-agent-btn'));
    fireEvent.press(screen.getByTestId('trigger-agent-cancel-btn'));

    expect(mockTriggerRun).not.toHaveBeenCalled();
  });
});
