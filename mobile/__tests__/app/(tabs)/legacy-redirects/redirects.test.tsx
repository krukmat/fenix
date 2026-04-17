// W2-T2 (mobile_wedge_harmonization_plan): legacy route redirect tests
// Verifies that /cases, /accounts, /deals redirect to their wedge destinations.
// Note: home/index is now the real HomeFeed screen, not a redirect.
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

// ─── Mocks ────────────────────────────────────────────────────────────────────

jest.mock('expo-router', () => {
  const React = require('react');
  const { View, Text } = require('react-native');
  return {
    __esModule: true,
    Redirect: ({ href }: { href: string }) =>
      React.createElement(
        View,
        { testID: 'redirect-component' },
        React.createElement(Text, { testID: 'redirect-href' }, String(href))
      ),
    useLocalSearchParams: () => ({}),
    Stack: { Screen: jest.fn(() => null) },
  };
});

// ─── cases/index ──────────────────────────────────────────────────────────────

describe('cases/index redirect → /support', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders a Redirect to /support', async () => {
    const { default: CasesRedirect } = await require('../../../../app/(tabs)/cases/index');
    render(React.createElement(CasesRedirect));
    expect(screen.getByTestId('redirect-href').props.children).toBe('/support');
  });
});

// ─── accounts/index ───────────────────────────────────────────────────────────

describe('accounts/index redirect → /sales', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders a Redirect to /sales', async () => {
    const { default: AccountsRedirect } = await require('../../../../app/(tabs)/accounts/index');
    render(React.createElement(AccountsRedirect));
    expect(screen.getByTestId('redirect-href').props.children).toBe('/sales');
  });
});

// ─── deals/index ──────────────────────────────────────────────────────────────

describe('deals/index redirect → /sales', () => {
  beforeEach(() => jest.clearAllMocks());

  it('renders a Redirect to /sales', async () => {
    const { default: DealsRedirect } = await require('../../../../app/(tabs)/deals/index');
    render(React.createElement(DealsRedirect));
    expect(screen.getByTestId('redirect-href').props.children).toBe('/sales');
  });
});
