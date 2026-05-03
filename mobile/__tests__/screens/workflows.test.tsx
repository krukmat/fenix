import React from 'react';
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import { fireEvent, render, waitFor, within } from '@testing-library/react-native';
import { Alert } from 'react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import type { Workflow } from '../../src/services/api';
import WorkflowsIndex from '../../app/(tabs)/workflows/index';
import WorkflowDetail from '../../app/(tabs)/workflows/[id]';
import WorkflowNew from '../../app/(tabs)/workflows/new';
import WorkflowEdit from '../../app/(tabs)/workflows/edit/[id]';

const mockUseWorkflows = jest.fn();
const mockUseWorkflow = jest.fn();
const mockUseActivateWorkflow = jest.fn();
const mockUseExecuteWorkflow = jest.fn();
const mockUseCreateWorkflow = jest.fn();
const mockUseUpdateWorkflow = jest.fn();
const mockUseWorkflowVersions = jest.fn();
const mockUseNewVersion = jest.fn();
const mockUseRollback = jest.fn();
const mockPush = jest.fn();
const mockReplace = jest.fn();
jest.mock('../../src/hooks/useAgentSpec', () => ({
  useWorkflows: (...args: unknown[]) => mockUseWorkflows(...args),
  useWorkflow: (...args: unknown[]) => mockUseWorkflow(...args),
  useActivateWorkflow: () => mockUseActivateWorkflow(),
  useExecuteWorkflow: () => mockUseExecuteWorkflow(),
  useCreateWorkflow: () => mockUseCreateWorkflow(),
  useUpdateWorkflow: () => mockUseUpdateWorkflow(),
  useWorkflowVersions: (...args: unknown[]) => mockUseWorkflowVersions(...args),
  useNewVersion: () => mockUseNewVersion(),
  useRollback: () => mockUseRollback(),
  useSignals: jest.fn().mockReturnValue({ data: { pages: [] }, isLoading: false, refetch: jest.fn() }),
  usePendingApprovals: jest.fn().mockReturnValue({ data: [], isLoading: false, refetch: jest.fn() }),
  useDismissSignal: jest.fn().mockReturnValue({ mutate: jest.fn() }),
  useDecideApproval: jest.fn().mockReturnValue({ mutate: jest.fn() }),
}));

jest.mock('expo-router', () => ({
  useRouter: () => ({ push: mockPush, replace: mockReplace }),
  useLocalSearchParams: () => ({ id: 'wf-1' }),
  Stack: {
    Screen: ({ children }: { children?: React.ReactNode }) => children ?? null,
  },
}));

jest.mock('../../src/stores/authStore', () => ({
  useAuthStore: (selector: (state: { workspaceId: string }) => unknown) => selector({ workspaceId: 'ws-1' }),
}));

jest.mock('../../src/services/api', () => ({
  workflowApi: { verifyWorkflow: jest.fn() },
}));

const activeWorkflow: Workflow = {
  id: 'wf-1',
  workspace_id: 'ws-1',
  name: 'Lead Nurture',
  dsl_source: 'workflow:\n  name: lead_nurture',
  version: 1,
  status: 'active',
  created_at: '2026-03-01T10:00:00Z',
  updated_at: '2026-03-01T10:00:00Z',
};

const draftWorkflow: Workflow = {
  ...activeWorkflow,
  id: 'wf-2',
  name: 'Draft Lead Nurture',
  status: 'draft',
};

const archivedWorkflow: Workflow = {
  ...activeWorkflow,
  id: 'wf-3',
  name: 'Archived Lead Nurture',
  version: 3,
  status: 'archived',
};

function setupListMock(workflows: Workflow[]) {
  mockUseWorkflows.mockReturnValue({
    data: { pages: [workflows] },
    isLoading: false,
    isRefetching: false,
    isFetchingNextPage: false,
    hasNextPage: false,
    fetchNextPage: jest.fn(),
    refetch: jest.fn(),
  });
}

describe('Workflows screens', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(Alert, 'alert').mockImplementation(() => {});
    mockUseActivateWorkflow.mockReturnValue({ mutate: jest.fn(), isPending: false });
    mockUseExecuteWorkflow.mockReturnValue({ mutate: jest.fn(), isPending: false });
    mockUseCreateWorkflow.mockReturnValue({ mutateAsync: jest.fn(), isPending: false });
    mockUseUpdateWorkflow.mockReturnValue({ mutateAsync: jest.fn(), isPending: false });
    mockUseWorkflowVersions.mockReturnValue({ data: [activeWorkflow, draftWorkflow] });
    mockUseNewVersion.mockReturnValue({ mutate: jest.fn(), isPending: false });
    mockUseRollback.mockReturnValue({ mutate: jest.fn(), isPending: false });
  });

  describe('list', () => {
    it('renders workflow cards and list actions', () => {
      setupListMock([activeWorkflow, draftWorkflow]);
      const Screen = WorkflowsIndex;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      expect(getByTestId(`workflow-${activeWorkflow.id}`)).toBeTruthy();
      expect(getByTestId(`workflow-${draftWorkflow.id}`)).toBeTruthy();
      expect(getByTestId('workflows-new-btn')).toBeTruthy();
    });

    it('shows empty state when no workflows exist', () => {
      setupListMock([]);
      const Screen = WorkflowsIndex;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      expect(getByTestId('workflows-empty')).toBeTruthy();
    });

    it('passes status filter to useWorkflows and navigates to new/detail screens', () => {
      setupListMock([activeWorkflow]);
      const Screen = WorkflowsIndex;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      fireEvent.press(getByTestId('workflows-chip-active'));
      expect(mockUseWorkflows).toHaveBeenCalledWith({ status: 'active' });

      fireEvent.press(getByTestId('workflows-new-btn'));
      expect(mockPush).toHaveBeenCalledWith('/workflows/new');

      fireEvent.press(getByTestId(`workflow-${activeWorkflow.id}`));
      expect(mockPush).toHaveBeenCalledWith(`/workflows/${activeWorkflow.id}`);
    });
  });

  describe('detail', () => {
    it('renders workflow metadata and DSL', () => {
      mockUseWorkflow.mockReturnValue({ data: activeWorkflow, isLoading: false, refetch: jest.fn() });
      mockUseWorkflowVersions.mockReturnValue({ data: [activeWorkflow, draftWorkflow] });
      const Screen = WorkflowDetail;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      expect(getByTestId('workflow-detail-name').props.children).toBe('Lead Nurture');
      expect(getByTestId('workflow-detail-version').props.children).toBe('v1');
      expect(within(getByTestId('workflow-detail-status')).getByText('active')).toBeTruthy();
      expect(getByTestId('workflow-detail-dsl-code').props.children).toContain('lead_nurture');
      expect(getByTestId('workflow-version-history')).toBeTruthy();
      expect(getByTestId(`workflow-version-${activeWorkflow.id}`)).toBeTruthy();
    });

    it('does not crash when workflow versions payload is malformed', () => {
      mockUseWorkflow.mockReturnValue({ data: activeWorkflow, isLoading: false, refetch: jest.fn() });
      mockUseWorkflowVersions.mockReturnValue({ data: { data: [activeWorkflow] } });
      const Screen = WorkflowDetail;
      const { getByTestId, queryByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      expect(getByTestId('workflow-detail')).toBeTruthy();
      expect(getByTestId('workflow-version-history')).toBeTruthy();
      expect(queryByTestId(`workflow-version-${activeWorkflow.id}`)).toBeNull();
    });

    it('shows execute only for active workflows', () => {
      mockUseWorkflow.mockReturnValue({ data: activeWorkflow, isLoading: false, refetch: jest.fn() });
      const Screen = WorkflowDetail;
      const { getByTestId, queryByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      expect(getByTestId('workflow-new-version-btn')).toBeTruthy();
      expect(getByTestId('workflow-execute-btn')).toBeTruthy();
      expect(queryByTestId('workflow-activate-btn')).toBeNull();
      expect(queryByTestId('workflow-edit-btn')).toBeNull();
    });

    it('shows edit, activate and verify for draft workflows', () => {
      mockUseWorkflow.mockReturnValue({ data: draftWorkflow, isLoading: false, refetch: jest.fn() });
      const Screen = WorkflowDetail;
      const { getByTestId, queryByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      expect(getByTestId('workflow-edit-btn')).toBeTruthy();
      expect(getByTestId('workflow-activate-btn')).toBeTruthy();
      expect(getByTestId('workflow-verify-btn')).toBeTruthy();
      expect(queryByTestId('workflow-new-version-btn')).toBeNull();
      expect(queryByTestId('workflow-execute-btn')).toBeNull();

      fireEvent.press(getByTestId('workflow-edit-btn'));
      expect(mockPush).toHaveBeenCalledWith('/workflows/edit/wf-1');
    });

    it('creates a new version from active workflow and navigates to the new detail', async () => {
      const mutate = jest.fn((_id, options) => options?.onSuccess?.({ id: 'wf-9' }));
      mockUseWorkflow.mockReturnValue({ data: activeWorkflow, isLoading: false, refetch: jest.fn() });
      mockUseNewVersion.mockReturnValue({ mutate, isPending: false });
      const Screen = WorkflowDetail;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      fireEvent.press(getByTestId('workflow-new-version-btn'));

      await waitFor(() => {
        expect(mutate).toHaveBeenCalledWith('wf-1', expect.any(Object));
      });
      expect(mockPush).toHaveBeenCalledWith('/workflows/wf-9');
    });

    it('shows rollback action for archived current workflow and triggers rollback', () => {
      const mutate = jest.fn();
      mockUseWorkflow.mockReturnValue({ data: archivedWorkflow, isLoading: false, refetch: jest.fn() });
      mockUseWorkflowVersions.mockReturnValue({ data: [activeWorkflow, archivedWorkflow] });
      mockUseRollback.mockReturnValue({ mutate, isPending: false });
      const Screen = WorkflowDetail;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      fireEvent.press(getByTestId(`workflow-rollback-btn-${archivedWorkflow.id}`));
      expect(mutate).toHaveBeenCalledWith('wf-3', expect.any(Object));
    });
  });

  describe('new workflow', () => {
    it('blocks submit when required fields are empty', () => {
      const mutateAsync = jest.fn();
      mockUseCreateWorkflow.mockReturnValue({ mutateAsync, isPending: false });
      const Screen = WorkflowNew;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      fireEvent.press(getByTestId('workflow-new-submit'));
      expect(mutateAsync).not.toHaveBeenCalled();
    });

    it('creates a draft workflow and navigates to detail', async () => {
      const mutateAsync = jest.fn().mockResolvedValue({ id: 'wf-3' });
      mockUseCreateWorkflow.mockReturnValue({ mutateAsync, isPending: false });
      const Screen = WorkflowNew;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      fireEvent.changeText(getByTestId('workflow-form-name-input'), 'Lead Qualifier');
      fireEvent.changeText(getByTestId('workflow-form-description-input'), 'Draft mobile flow');
      fireEvent.changeText(getByTestId('workflow-form-dsl-input'), 'ON lead.created');
      fireEvent.press(getByTestId('workflow-new-submit'));

      await waitFor(() => {
        expect(mutateAsync).toHaveBeenCalledWith({
          name: 'Lead Qualifier',
          description: 'Draft mobile flow',
          dsl_source: 'ON lead.created',
        });
      });
      expect(mockReplace).toHaveBeenCalledWith('/workflows/wf-3');
    });
  });

  describe('edit workflow', () => {
    it('shows a disabled state for non-draft workflows', () => {
      mockUseWorkflow.mockReturnValue({ data: activeWorkflow, isLoading: false });
      const Screen = WorkflowEdit;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      expect(getByTestId('workflow-edit-disabled')).toBeTruthy();
    });

    it('updates a draft workflow and navigates back to detail', async () => {
      const mutateAsync = jest.fn().mockResolvedValue({ id: 'wf-1' });
      mockUseWorkflow.mockReturnValue({ data: { ...draftWorkflow, id: 'wf-1' }, isLoading: false });
      mockUseUpdateWorkflow.mockReturnValue({ mutateAsync, isPending: false });
      const Screen = WorkflowEdit;
      const { getByTestId } = render(
        <PaperProvider>
          <Screen />
        </PaperProvider>
      );

      fireEvent.changeText(getByTestId('workflow-form-description-input'), 'Updated draft');
      fireEvent.changeText(getByTestId('workflow-form-dsl-input'), 'ON lead.updated');
      fireEvent.press(getByTestId('workflow-edit-submit'));

      await waitFor(() => {
        expect(mutateAsync).toHaveBeenCalledWith({
          id: 'wf-1',
          data: {
            description: 'Updated draft',
            dsl_source: 'ON lead.updated',
          },
        });
      });
      expect(mockReplace).toHaveBeenCalledWith('/workflows/wf-1');
    });
  });
});
