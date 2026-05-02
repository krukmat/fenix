// F9.A5: Support copilot route — canonical trigger contract + existing context tests
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

const mockPush = jest.fn();
const mockMutateAsync = jest.fn();
const mockUseTriggerSupportAgent = jest.fn();

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'case-1', entity_type: 'case', entity_id: 'case-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerSupportAgent: () => mockUseTriggerSupportAgent(),
}));

// Capture onSupportTrigger passed to CopilotPanel so tests can call it directly
let capturedOnSupportTrigger: ((query: string) => void) | undefined;
jest.mock('../../../../src/components/copilot', () => {
  const ReactModule = require('react');
  const { View } = require('react-native');
  return {
    CopilotPanel: ({
      initialContext,
      onSupportTrigger,
    }: {
      initialContext?: { entityType?: string; entityId?: string };
      onSupportTrigger?: (query: string) => void;
    }) => {
      capturedOnSupportTrigger = onSupportTrigger;
      return ReactModule.createElement(View, {
        testID: 'support-copilot-panel',
        accessibilityLabel: `${initialContext?.entityType ?? ''}:${initialContext?.entityId ?? ''}`,
      });
    },
  };
});

jest.mock('react-native-paper', () => ({
  useTheme: () => ({ colors: { background: '#fff', primary: '#E53935' } }),
}));

// Load screen once — no isolateModules to avoid double-React-instance issue
const { default: Screen } = require('../../../../app/(tabs)/support/[id]/copilot');

describe('Support copilot route — existing context tests', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    capturedOnSupportTrigger = undefined;
    mockUseTriggerSupportAgent.mockReturnValue({ mutateAsync: mockMutateAsync, isPending: false });
  });

  it('renders the copilot panel', () => {
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-copilot-panel')).toBeTruthy();
  });

  it('passes entity context to CopilotPanel', () => {
    render(React.createElement(Screen));
    const panel = screen.getByTestId('support-copilot-panel');
    expect(panel.props.accessibilityLabel).toBe('case:case-1');
  });
});

describe('SupportCopilotScreen — canonical trigger (F9.A5)', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    capturedOnSupportTrigger = undefined;
    mockUseTriggerSupportAgent.mockReturnValue({ mutateAsync: mockMutateAsync, isPending: false });
    mockMutateAsync.mockResolvedValue({ runId: 'run-abc', status: 'queued', agent: 'support' });
  });

  it('passes onSupportTrigger prop to CopilotPanel', () => {
    render(React.createElement(Screen));
    expect(typeof capturedOnSupportTrigger).toBe('function');
  });

  it('onSupportTrigger calls useTriggerSupportAgent with canonical shape', async () => {
    render(React.createElement(Screen));

    await capturedOnSupportTrigger!('the screen is broken');

    expect(mockMutateAsync).toHaveBeenCalledWith({
      caseId: 'case-1',
      customerQuery: 'the screen is broken',
      language: undefined,
      priority: undefined,
    });
  });

  it('onSupportTrigger navigates to run detail on success', async () => {
    render(React.createElement(Screen));

    await capturedOnSupportTrigger!('refund request');

    expect(mockPush).toHaveBeenCalledWith('/activity/run-abc');
  });
});
