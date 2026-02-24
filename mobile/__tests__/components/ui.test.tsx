import React from 'react';
import { Text } from 'react-native';
import { describe, it, expect } from '@jest/globals';
import { render } from '@testing-library/react-native';

import { LoadingScreen } from '../../src/components/ui/LoadingScreen';
import { AuthFormLayout } from '../../src/components/ui/AuthFormLayout';
import { ErrorBoundaryClass } from '../../src/components/ui/ErrorBoundary';

describe('UI components', () => {
  it('LoadingScreen renders with optional message', () => {
    const { getByText } = render(<LoadingScreen message="Loading data" />);
    expect(getByText('Loading data')).toBeTruthy();
  });

  it('AuthFormLayout renders title, subtitle and children', () => {
    // children must be a <Text> node — RNTL getByText only finds text inside RN Text elements
    const { getByText } = render(
      <AuthFormLayout title="Title" subtitle="Subtitle">
        <Text>Child content</Text>
      </AuthFormLayout>
    );

    expect(getByText('Title')).toBeTruthy();
    expect(getByText('Subtitle')).toBeTruthy();
    expect(getByText('Child content')).toBeTruthy();
  });

  it('ErrorBoundaryClass renders children when no error', () => {
    // same: wrap in <Text> so RNTL can find the node
    const { getByText } = render(
      <ErrorBoundaryClass>
        <Text>Safe child</Text>
      </ErrorBoundaryClass>
    );

    expect(getByText('Safe child')).toBeTruthy();
  });
});
