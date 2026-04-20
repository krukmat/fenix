import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';
import { CRMEntityChildForms } from '../../../../src/components/crm/CRMEntityChildForms';

const mockCreateActivity = jest.fn();
const mockCreateNote = jest.fn();
const mockCreateAttachment = jest.fn();
let mockUserId: string | null = 'user-1';

jest.mock('react-native-paper', () => ({
  useTheme: () => ({
    colors: {
      primary: '#E53935',
      onPrimary: '#FFFFFF',
      surface: '#FFFFFF',
      surfaceVariant: '#EEF2F7',
      onSurface: '#111827',
      onSurfaceVariant: '#6B7280',
      background: '#FFFFFF',
      outline: '#CBD5E1',
      error: '#B00020',
    },
  }),
}));

jest.mock('../../../../src/stores/authStore', () => ({
  useAuthStore: (selector: (state: { userId: string | null }) => unknown) => selector({ userId: mockUserId }),
}));

jest.mock('../../../../src/hooks/useCRM', () => ({
  useCreateActivity: () => ({ mutateAsync: mockCreateActivity, isPending: false }),
  useCreateNote: () => ({ mutateAsync: mockCreateNote, isPending: false }),
  useCreateAttachment: () => ({ mutateAsync: mockCreateAttachment, isPending: false }),
}));

describe('CRM entity child forms', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUserId = 'user-1';
    mockCreateActivity.mockResolvedValue({});
    mockCreateNote.mockResolvedValue({});
    mockCreateAttachment.mockResolvedValue({});
  });

  it('creates an activity linked to the current CRM entity', async () => {
    render(<CRMEntityChildForms entityType="case" entityId="case-1" />);
    fireEvent.changeText(screen.getByTestId('crm-entity-child-activity-subject'), 'Follow up');
    fireEvent.changeText(screen.getByTestId('crm-entity-child-activity-body'), 'Call the customer');
    fireEvent.press(screen.getByTestId('crm-entity-child-activity-submit'));

    await waitFor(() => expect(mockCreateActivity).toHaveBeenCalledWith({
      entityType: 'case',
      entityId: 'case-1',
      ownerId: 'user-1',
      activityType: 'task',
      subject: 'Follow up',
      body: 'Call the customer',
      status: 'open',
      metadata: { source: 'mobile-crm-validation' },
    }));
  });

  it('creates an internal note with the signed-in user as author', async () => {
    render(<CRMEntityChildForms entityType="account" entityId="acc-1" />);
    fireEvent.changeText(screen.getByTestId('crm-entity-child-note-content'), 'Decision maker confirmed');
    fireEvent.press(screen.getByTestId('crm-entity-child-note-internal'));
    fireEvent.press(screen.getByTestId('crm-entity-child-note-submit'));

    await waitFor(() => expect(mockCreateNote).toHaveBeenCalledWith({
      entityType: 'account',
      entityId: 'acc-1',
      authorId: 'user-1',
      content: 'Decision maker confirmed',
      isInternal: true,
      metadata: { source: 'mobile-crm-validation' },
    }));
  });

  it('creates attachment metadata without binary upload', async () => {
    render(<CRMEntityChildForms entityType="deal" entityId="deal-1" />);
    fireEvent.changeText(screen.getByTestId('crm-entity-child-attachment-filename'), 'quote.pdf');
    fireEvent.changeText(screen.getByTestId('crm-entity-child-attachment-storage-path'), 'crm/deals/deal-1/quote.pdf');
    fireEvent.changeText(screen.getByTestId('crm-entity-child-attachment-content-type'), 'application/pdf');
    fireEvent.changeText(screen.getByTestId('crm-entity-child-attachment-size-bytes'), '2048');
    fireEvent.press(screen.getByTestId('crm-entity-child-attachment-submit'));

    await waitFor(() => expect(mockCreateAttachment).toHaveBeenCalledWith({
      entityType: 'deal',
      entityId: 'deal-1',
      uploaderId: 'user-1',
      filename: 'quote.pdf',
      storagePath: 'crm/deals/deal-1/quote.pdf',
      contentType: 'application/pdf',
      sizeBytes: 2048,
      metadata: { source: 'mobile-crm-validation' },
    }));
  });

  it('blocks child mutations when no signed-in user is available', async () => {
    mockUserId = null;
    render(<CRMEntityChildForms entityType="lead" entityId="lead-1" />);
    fireEvent.changeText(screen.getByTestId('crm-entity-child-note-content'), 'Needs owner');
    fireEvent.press(screen.getByTestId('crm-entity-child-note-submit'));

    expect(await screen.findByText('Signed-in user is required')).toBeTruthy();
    expect(mockCreateNote).not.toHaveBeenCalled();
  });

  it('shows child mutation failures without clearing entered data', async () => {
    mockCreateActivity.mockRejectedValue(new Error('Activity failed'));
    render(<CRMEntityChildForms entityType="case" entityId="case-1" />);
    fireEvent.changeText(screen.getByTestId('crm-entity-child-activity-subject'), 'Follow up');
    fireEvent.press(screen.getByTestId('crm-entity-child-activity-submit'));

    expect(await screen.findByText('Activity failed')).toBeTruthy();
    expect(screen.getByDisplayValue('Follow up')).toBeTruthy();
  });
});
