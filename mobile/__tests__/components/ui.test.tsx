import React from 'react';
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
    const { getByText } = render(
      <AuthFormLayout title="Title" subtitle="Subtitle">
        <>{'Child content'}</>
      </AuthFormLayout>
    );

    expect(getByText('Title')).toBeTruthy();
    expect(getByText('Subtitle')).toBeTruthy();
    expect(getByText('Child content')).toBeTruthy();
  });

  it('ErrorBoundaryClass renders children when no error', () => {
    const { getByText } = render(
      <ErrorBoundaryClass>
        <>{'Safe child'}</>
      </ErrorBoundaryClass>
    );

    expect(getByText('Safe child')).toBeTruthy();
  });
});
