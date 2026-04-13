// Traces: FR-001
import { describe, it, expect } from '@jest/globals';
import { renderCaseContent } from '../../app/(tabs)/cases/[id]';

const mockColors = {
  background: '#fff',
  surface: '#f5f5f5',
  surfaceVariant: '#eee',
  primary: '#6200EE',
  onPrimary: '#fff',
  onSurface: '#000',
  onSurfaceVariant: '#666',
  error: '#B00020',
  outline: '#ccc',
};

const mockRouter = {
  push: () => undefined,
} as any;
const mockTriggerKB = {
  mutate: () => undefined,
  isPending: false,
};

jest.mock('../../src/components/agents/AgentActivitySection', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    AgentActivitySection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-section` }),
  };
});

jest.mock('../../src/components/signals/EntitySignalsSection', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    EntitySignalsSection: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-section` }),
  };
});

jest.mock('../../src/components/signals/SignalCountBadge', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    SignalCountBadge: ({ testID }: { testID: string }) => React.createElement(View, { testID }),
  };
});

jest.mock('../../src/components/crm', () => {
  const React = require('react');
  const { View } = require('react-native');
  return {
    CRMDetailHeader: ({ testIDPrefix }: { testIDPrefix: string }) =>
      React.createElement(View, { testID: `${testIDPrefix}-header` }),
  };
});

jest.mock('../../src/hooks/useWedge', () => ({
  useTriggerKBAgent: () => ({ mutate: () => undefined, isPending: false }),
}));

describe('Cases detail — SLA and handoff section', () => {
  function collectTestIDs(node: unknown, acc: string[] = []): string[] {
    if (!node || typeof node !== 'object') return acc;

    const n = node as { props?: { testID?: string; children?: unknown } };
    if (n.props?.testID) acc.push(n.props.testID);

    const children = n.props?.children;
    if (Array.isArray(children)) {
      children.forEach((c) => collectTestIDs(c, acc));
    } else if (children) {
      collectTestIDs(children, acc);
    }

    return acc;
  }

  it('shows SLA deadline when slaDeadline is present', () => {
    const caseData = {
      id: '1',
      subject: 'Case A',
      status: 'open',
      priority: 'high' as const,
      slaDeadline: '2026-03-01T00:00:00Z',
      handoffStatus: undefined,
      accountId: undefined,
      accountName: undefined,
    };

    const element = renderCaseContent(caseData, mockColors, mockRouter, mockTriggerKB);
    expect(collectTestIDs(element)).toContain('case-sla-deadline');
  });

  it('shows handoff status when handoffStatus is present', () => {
    const caseData = {
      id: '2',
      subject: 'Case B',
      status: 'open',
      priority: 'medium' as const,
      slaDeadline: undefined,
      handoffStatus: 'escalated',
      accountId: undefined,
      accountName: undefined,
    };

    const element = renderCaseContent(caseData, mockColors, mockRouter, mockTriggerKB);
    expect(collectTestIDs(element)).toContain('case-handoff-status');
  });

  it('does not show SLA/handoff section when both are absent', () => {
    const caseData = {
      id: '3',
      subject: 'Case C',
      status: 'closed',
      priority: 'low' as const,
      slaDeadline: undefined,
      handoffStatus: undefined,
      accountId: undefined,
      accountName: undefined,
    };

    const element = renderCaseContent(caseData, mockColors, mockRouter, mockTriggerKB);
    const testIDs = collectTestIDs(element);
    expect(testIDs).not.toContain('case-sla-deadline');
    expect(testIDs).not.toContain('case-handoff-status');
  });

  it('shows KB trigger when case is resolved', () => {
    const caseData = {
      id: '4',
      subject: 'Case D',
      status: 'resolved',
      priority: 'medium' as const,
      slaDeadline: undefined,
      handoffStatus: undefined,
      accountId: undefined,
      accountName: undefined,
    };

    const element = renderCaseContent(caseData, mockColors, mockRouter, mockTriggerKB);
    expect(collectTestIDs(element)).toContain('kb-trigger-button');
  });

  it('does not show KB trigger when case is not resolved', () => {
    const caseData = {
      id: '5',
      subject: 'Case E',
      status: 'open',
      priority: 'medium' as const,
      slaDeadline: undefined,
      handoffStatus: undefined,
      accountId: undefined,
      accountName: undefined,
    };

    const element = renderCaseContent(caseData, mockColors, mockRouter, mockTriggerKB);
    expect(collectTestIDs(element)).not.toContain('kb-trigger-button');
  });
});
