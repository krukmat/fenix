// WorkflowCard — status badge colors, name/version display
// FR-302 (Workflows), UC-A4: workflow visibility


import React from 'react';
import { describe, it, expect, jest } from '@jest/globals';
import { render, fireEvent, within } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { WorkflowCard } from '../../../src/components/workflows/WorkflowCard';
import { DSLViewer } from '../../../src/components/workflows/DSLViewer';
import type { Workflow } from '../../../src/services/api';

const baseWorkflow: Workflow = {
  id: 'wf-1',
  workspace_id: 'ws-1',
  name: 'Lead Nurture',
  dsl_source: 'workflow:\n  name: lead_nurture\n  steps: []',
  version: 2,
  status: 'active',
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

function renderCard(props?: Partial<Parameters<typeof WorkflowCard>[0]>) {
  const onPress = jest.fn();
  const utils = render(
    <PaperProvider>
      <WorkflowCard workflow={baseWorkflow} onPress={onPress} {...props} />
    </PaperProvider>
  );
  return { ...utils, onPress };
}

describe('WorkflowCard', () => {
  it('renders name and version', () => {
    const { getByTestId } = renderCard();
    expect(getByTestId('workflow-card-name').props.children).toBe('Lead Nurture');
    expect(getByTestId('workflow-card-version').props.children).toBe('v2');
  });

  it('renders status badge with status text', () => {
    const { getByTestId } = renderCard();
    expect(within(getByTestId('workflow-card-status')).getByText('active')).toBeTruthy();
  });

  it('renders description when present', () => {
    const { getByTestId } = renderCard({
      workflow: { ...baseWorkflow, description: 'Nurtures leads automatically' },
    });
    expect(getByTestId('workflow-card-description').props.children).toBe('Nurtures leads automatically');
  });

  it('does not render description when absent', () => {
    const { queryByTestId } = renderCard();
    expect(queryByTestId('workflow-card-description')).toBeNull();
  });

  it('calls onPress with workflow when card is pressed', () => {
    const { getByTestId, onPress } = renderCard();
    fireEvent.press(getByTestId('workflow-card'));
    expect(onPress).toHaveBeenCalledWith(baseWorkflow);
  });

  // Status color coverage (structural — confirms chip exists for each status)
  it.each([['draft'], ['testing'], ['active'], ['archived']] as [Workflow['status']][])(
    'renders %s status badge',
    (status) => {
      const { getByTestId } = renderCard({ workflow: { ...baseWorkflow, status } });
      expect(within(getByTestId('workflow-card-status')).getByText(status)).toBeTruthy();
    }
  );
});

describe('DSLViewer', () => {
  it('renders DSL source code', () => {
    const dsl = 'workflow:\n  name: test';
    const { getByTestId } = render(
      <PaperProvider>
        <DSLViewer dsl={dsl} />
      </PaperProvider>
    );
    expect(getByTestId('dsl-viewer-code').props.children).toBe(dsl);
  });

  it('renders both horizontal and vertical scroll views', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <DSLViewer dsl="step: foo" />
      </PaperProvider>
    );
    expect(getByTestId('dsl-viewer-hscroll')).toBeTruthy();
    expect(getByTestId('dsl-viewer-vscroll')).toBeTruthy();
  });
});