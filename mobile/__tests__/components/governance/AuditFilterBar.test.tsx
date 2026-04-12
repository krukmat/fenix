import React from 'react';
import { describe, expect, it, jest } from '@jest/globals';
import { fireEvent, render } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { AuditFilterBar } from '../../../src/components/governance/AuditFilterBar';

function renderBar() {
  const onChange = jest.fn();
  const utils = render(
    <PaperProvider>
      <AuditFilterBar filters={{}} onChange={onChange} />
    </PaperProvider>
  );
  return { ...utils, onChange };
}

describe('AuditFilterBar', () => {
  it('renders all outcome chips', () => {
    const { getByTestId } = renderBar();
    expect(getByTestId('audit-filter-outcome-all')).toBeTruthy();
    expect(getByTestId('audit-filter-outcome-success')).toBeTruthy();
    expect(getByTestId('audit-filter-outcome-denied')).toBeTruthy();
    expect(getByTestId('audit-filter-outcome-error')).toBeTruthy();
  });

  it('calls onChange with the selected outcome filter', () => {
    const { getByTestId, onChange } = renderBar();
    fireEvent.press(getByTestId('audit-filter-outcome-denied'));
    expect(onChange).toHaveBeenCalledWith({ outcome: 'denied' });
  });

  it('renders the active chip with an override style', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <AuditFilterBar filters={{ outcome: 'success' }} onChange={jest.fn()} />
      </PaperProvider>
    );
    expect(getByTestId('audit-filter-outcome-success').props.style.backgroundColor).toBeDefined();
  });
});
