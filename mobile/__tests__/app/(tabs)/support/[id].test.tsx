// Support case detail screen tests — UIX-31
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react-native';

const mockPush = jest.fn();
jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: mockPush, replace: jest.fn() }),
  useLocalSearchParams: () => ({ id: 'case-1' }),
  Stack: { Screen: jest.fn(() => null) },
}));

const mockUseCase = jest.fn();
const mockUseEntityTimeline = jest.fn();
const mockUseCaseActivities = jest.fn();
const mockUseCaseNotes = jest.fn();
jest.mock('../../../../src/hooks/useCRM', () => ({
  useCase: (...args: unknown[]) => mockUseCase(...args),
  useEntityTimeline: (...args: unknown[]) => mockUseEntityTimeline(...args),
  useCaseActivities: (...args: unknown[]) => mockUseCaseActivities(...args),
  useCaseNotes: (...args: unknown[]) => mockUseCaseNotes(...args),
}));

const mockTriggerAgentMutate = jest.fn();
jest.mock('../../../../src/hooks/useWedge', () => ({
  useTriggerSupportAgent: () => ({ mutate: mockTriggerAgentMutate, isPending: false }),
  useTriggerKBAgent: () => ({ mutate: jest.fn(), isPending: false }),
  useAgentRuns: () => ({ data: null }),
}));

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
        testID: 'support-case-detail-copilot',
        accessibilityLabel: `${initialContext?.entityType ?? ''}:${initialContext?.entityId ?? ''}`,
      });
    },
  };
});

jest.mock('../../../../src/components/crm', () => {
  const ReactModule = require('react');
  const { View, Text } = require('react-native');
  return {
    CRMDetailHeader: ({ title, testIDPrefix }: { title: string; testIDPrefix: string }) =>
      ReactModule.createElement(View, { testID: `${testIDPrefix}-header` }, ReactModule.createElement(Text, null, title)),
    EntityTimeline: ({ events, testIDPrefix }: { events: { title: string }[]; testIDPrefix: string }) =>
      ReactModule.createElement(
        View,
        {
          testID: `${testIDPrefix}-list`,
          accessibilityLabel: events.map((event) => event.title).join(' > '),
        },
      ),
  };
});

jest.mock('../../../../src/components/agents/AgentActivitySection', () => {
  const ReactModule = require('react');
  const { View } = require('react-native');
  return {
    AgentActivitySection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      ReactModule.createElement(View, { testID: `${testIDPrefix}-section` }),
  };
});

jest.mock('../../../../src/components/signals/EntitySignalsSection', () => {
  const ReactModule = require('react');
  const { View } = require('react-native');
  return {
    EntitySignalsSection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      ReactModule.createElement(View, { testID: `${testIDPrefix}-section` }),
  };
});

jest.mock('react-native-paper', () => {
  const ReactModule = require('react');
  const { TouchableOpacity, Text } = require('react-native');
  return {
    useTheme: () => ({
      colors: {
        primary: '#E53935',
        surface: '#f5f5f5',
        onSurface: '#000',
        onSurfaceVariant: '#666',
        background: '#fff',
        error: '#B00020',
        surfaceVariant: '#ddd',
        outline: '#999',
        outlineVariant: '#ccc',
      },
    }),
    Button: ({
      testID,
      onPress,
      children,
      disabled,
    }: {
      testID: string;
      onPress: () => void;
      children: React.ReactNode;
      disabled?: boolean;
    }) =>
      ReactModule.createElement(
        TouchableOpacity,
        { testID, onPress, accessibilityState: { disabled: !!disabled } },
        ReactModule.createElement(Text, null, children),
      ),
    Text: ({ children }: { children: React.ReactNode }) => ReactModule.createElement(Text, null, children),
  };
});

const casePayload = {
  case: {
    id: 'case-1',
    subject: 'Login broken',
    status: 'open',
    priority: 'high',
    description: 'Users cannot log in',
    accountId: 'acc-1',
    contactId: 'contact-1',
    sla_deadline: '2026-04-10T00:00:00Z',
    assignee: 'Ada Lovelace',
  },
  account: { name: 'Acme Corp' },
  contact: { firstName: 'Grace', lastName: 'Hopper' },
  handoff: { status: 'escalated' },
  active_signal_count: 2,
};

const timelinePayload = {
  data: [
    { id: 'tl-1', event_type: 'created', title: 'Case created', created_at: '2026-04-10T09:00:00Z' },
  ],
};

const activitiesPayload = {
  data: [
    { id: 'act-1', subject: 'Call customer', completed_at: '2026-04-10T10:00:00Z' },
  ],
};

const notesPayload = {
  data: [
    { id: 'note-1', content: 'Customer confirmed issue', created_at: '2026-04-10T11:00:00Z' },
  ],
};

describe('Support case detail screen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    capturedOnSupportTrigger = undefined;
    mockUseCase.mockReturnValue({ data: casePayload, isLoading: false, error: null });
    mockUseEntityTimeline.mockReturnValue({ data: timelinePayload, isLoading: false });
    mockUseCaseActivities.mockReturnValue({ data: activitiesPayload, isLoading: false });
    mockUseCaseNotes.mockReturnValue({ data: notesPayload, isLoading: false });
  });

  it('renders the detail screen container', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-detail-screen')).toBeTruthy();
  });

  it('renders the case header with subject', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-detail-header')).toBeTruthy();
  });

  it('renders highlights and status path', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-account-highlight')).toBeTruthy();
    expect(screen.getByTestId('support-case-contact-highlight')).toBeTruthy();
    expect(screen.getByTestId('support-case-owner-highlight')).toBeTruthy();
    expect(screen.getByTestId('support-case-status-path')).toBeTruthy();
  });

  it('renders SLA deadline when present', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-sla-deadline')).toBeTruthy();
  });

  it('renders handoff status when present', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-handoff-status')).toBeTruthy();
  });

  it('renders agent activity section', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-agent-activity-section')).toBeTruthy();
  });

  it('renders the merged timeline feed on the timeline tab', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));

    fireEvent.press(screen.getByTestId('support-case-tab-timeline'));

    const timeline = screen.getByTestId('support-case-timeline-list');
    expect(timeline).toBeTruthy();
    expect(timeline.props.accessibilityLabel).toBe('Note > Call customer > Case created');
  });

  it('embeds the CopilotPanel scoped to the case', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    const panel = screen.getByTestId('support-case-detail-copilot');
    expect(panel).toBeTruthy();
    expect(panel.props.accessibilityLabel).toBe('case:case-1');
  });

  it('wires CopilotPanel support trigger to the canonical mutation shape', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));

    capturedOnSupportTrigger?.('please help');

    expect(mockTriggerAgentMutate).toHaveBeenCalledWith({ caseId: 'case-1', customerQuery: 'please help' });
  });

  it('does not render the separate copilot navigation button anymore', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('support-copilot-button')).toBeNull();
  });

  it('does NOT show edit case button', () => {
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.queryByTestId('case-edit-button')).toBeNull();
  });

  it('shows loading state', () => {
    mockUseCase.mockReturnValue({ data: undefined, isLoading: true, error: null });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-detail-loading')).toBeTruthy();
  });

  it('shows error state', () => {
    mockUseCase.mockReturnValue({ data: undefined, isLoading: false, error: new Error('Not found') });
    const { default: Screen } = require('../../../../app/(tabs)/support/[id]');
    render(React.createElement(Screen));
    expect(screen.getByTestId('support-case-detail-error')).toBeTruthy();
  });
});
