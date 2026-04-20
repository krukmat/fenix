import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';
import CRMLeadNewScreen from '../../../../app/(tabs)/crm/leads/new';
import CRMLeadEditScreen from '../../../../app/(tabs)/crm/leads/edit/[id]';

const mockReplace = jest.fn();
const mockCreateMutateAsync = jest.fn();
const mockUpdateMutateAsync = jest.fn();
let mockLeadData: unknown = null;

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ replace: mockReplace }),
  useLocalSearchParams: () => ({ id: 'lead-1' }),
}));

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

jest.mock('../../../../src/hooks/useCRM', () => ({
  useLead: () => ({ data: mockLeadData, isLoading: false, error: null }),
  useCreateLead: () => ({ mutateAsync: mockCreateMutateAsync, isPending: false }),
  useUpdateLead: () => ({ mutateAsync: mockUpdateMutateAsync, isPending: false }),
}));

describe('CRM lead forms', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockLeadData = null;
    mockCreateMutateAsync.mockResolvedValue({});
    mockUpdateMutateAsync.mockResolvedValue({});
  });

  it('validates required lead name before create submit', async () => {
    render(<CRMLeadNewScreen />);
    fireEvent.press(screen.getByTestId('crm-lead-form-submit'));

    expect(await screen.findByText('Lead name is required')).toBeTruthy();
    expect(mockCreateMutateAsync).not.toHaveBeenCalled();
  });

  it('creates a lead with metadata and returns to the CRM lead list', async () => {
    render(<CRMLeadNewScreen />);
    fireEvent.changeText(screen.getByTestId('crm-lead-form-name'), 'Jane Lead');
    fireEvent.changeText(screen.getByTestId('crm-lead-form-email'), 'jane@example.test');
    fireEvent.changeText(screen.getByTestId('crm-lead-form-company'), 'Example Co');
    fireEvent.changeText(screen.getByTestId('crm-lead-form-source'), 'web');
    fireEvent.changeText(screen.getByTestId('crm-lead-form-score'), '80');
    fireEvent.press(screen.getByTestId('crm-lead-form-submit'));

    await waitFor(() => expect(mockCreateMutateAsync).toHaveBeenCalledWith({
      source: 'web',
      status: 'new',
      score: 80,
      metadata: {
        name: 'Jane Lead',
        email: 'jane@example.test',
        company: 'Example Co',
      },
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/leads');
  });

  it('loads existing lead data and updates the CRM lead detail', async () => {
    mockLeadData = {
      id: 'lead-1',
      source: 'web',
      status: 'qualified',
      score: 72,
      metadata: { name: 'Jane Lead', email: 'jane@example.test', company: 'Example Co' },
    };

    render(<CRMLeadEditScreen />);
    expect(screen.getByDisplayValue('Jane Lead')).toBeTruthy();
    fireEvent.changeText(screen.getByTestId('crm-lead-form-status'), 'converted');
    fireEvent.press(screen.getByTestId('crm-lead-form-submit'));

    await waitFor(() => expect(mockUpdateMutateAsync).toHaveBeenCalledWith({
      id: 'lead-1',
      data: {
        source: 'web',
        status: 'converted',
        score: 72,
        metadata: {
          name: 'Jane Lead',
          email: 'jane@example.test',
          company: 'Example Co',
        },
      },
    }));
    expect(mockReplace).toHaveBeenCalledWith('/crm/leads/lead-1');
  });
});
