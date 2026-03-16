import React from 'react';
import { describe, expect, it } from '@jest/globals';
import { render } from '@testing-library/react-native';
import { PaperProvider } from 'react-native-paper';
import { SignalCountBadge } from '../../../src/components/signals/SignalCountBadge';

describe('SignalCountBadge', () => {
  it('renders count when greater than zero', () => {
    const { getByTestId, getByText } = render(
      <PaperProvider>
        <SignalCountBadge count={3} testID="signal-badge" />
      </PaperProvider>
    );

    expect(getByTestId('signal-badge')).toBeTruthy();
    expect(getByText('3')).toBeTruthy();
  });

  it('returns null for zero count', () => {
    const { queryByTestId } = render(
      <PaperProvider>
        <SignalCountBadge count={0} testID="signal-badge" />
      </PaperProvider>
    );

    expect(queryByTestId('signal-badge')).toBeNull();
  });
});
