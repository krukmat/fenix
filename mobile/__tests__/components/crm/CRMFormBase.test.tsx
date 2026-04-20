import { describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

import {
  Field,
  FormErrorText,
  LoadingView,
  SubmitButton,
  listItems,
  record,
  unwrapDataArray,
} from '../../../src/components/crm/CRMFormBase';
import type { ThemeColors } from '../../../src/theme/types';

const colors: ThemeColors = {
  primary: '#E53935',
  onPrimary: '#FFFFFF',
  surface: '#FFFFFF',
  surfaceVariant: '#EEF2F7',
  onSurface: '#111827',
  onSurfaceVariant: '#6B7280',
  background: '#FFFFFF',
  outline: '#CBD5E1',
  error: '#B00020',
};

describe('CRMFormBase', () => {
  describe('components', () => {
    it('Field renders the label and forwards the testID to its input', () => {
      const onChangeText = jest.fn();

      render(
        <Field
          label="Lead name"
          value="Jane Lead"
          onChangeText={onChangeText}
          testID="crm-form-base-field"
        />,
      );

      expect(screen.getByText('Lead name')).toBeTruthy();
      expect(screen.getByTestId('crm-form-base-field').props.value).toBe('Jane Lead');
    });

    it('SubmitButton renders disabled when disabled is true', () => {
      render(
        <SubmitButton
          label="Create Lead"
          testID="crm-form-base-submit"
          disabled
          onPress={jest.fn()}
          colors={colors}
        />,
      );

      const button = screen.getByTestId('crm-form-base-submit');
      expect(button.props.accessibilityState?.disabled).toBe(true);
      expect(screen.getByText('Create Lead')).toBeTruthy();
    });

    it('FormErrorText renders null when error is null', () => {
      render(<FormErrorText error={null} style={{ color: colors.error }} />);

      expect(screen.queryByText('Validation failed')).toBeNull();
    });

    it('FormErrorText renders the provided error string', () => {
      render(<FormErrorText error="Validation failed" style={{ color: colors.error }} />);

      expect(screen.getByText('Validation failed')).toBeTruthy();
    });

    it('LoadingView renders with the provided testID', () => {
      render(<LoadingView testID="crm-form-base-loading" colors={colors} />);

      expect(screen.getByTestId('crm-form-base-loading')).toBeTruthy();
      expect(screen.getByText('Loading...')).toBeTruthy();
    });
  });

  describe('utilities', () => {
    it('record returns objects and null for primitive values', () => {
      const value = { id: 'lead-1', name: 'Jane Lead' };

      expect(record(value)).toBe(value);
      expect(record(null)).toBeNull();
      expect(record('lead-1')).toBeNull();
      expect(record(42)).toBeNull();
      expect(record(true)).toBeNull();
    });

    it('unwrapDataArray handles plain arrays and data payloads', () => {
      expect(unwrapDataArray<string>(['lead-1', 'lead-2'])).toEqual(['lead-1', 'lead-2']);
      expect(unwrapDataArray<string>({ data: ['lead-3', 'lead-4'] })).toEqual(['lead-3', 'lead-4']);
      expect(unwrapDataArray<string>({ data: 'not-an-array' })).toEqual([]);
    });

    it('listItems flattens paged data and normalizes each item', () => {
      const items = listItems(
        {
          pages: [
            [{ id: 'lead-1' }],
            { data: [{ id: 'lead-2' }] },
            { data: 'not-an-array' },
          ],
        },
        (raw) => {
          const payload = record(raw);
          return { id: String(payload?.id ?? '') };
        },
      );

      expect(items).toEqual([{ id: 'lead-1' }, { id: 'lead-2' }]);
    });
  });
});
