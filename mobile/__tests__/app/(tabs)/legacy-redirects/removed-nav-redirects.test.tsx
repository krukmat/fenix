// W2-T3 (mobile_wedge_harmonization_plan): removed visible nav items redirect tests
// CRM hub, Workflows, top-level Copilot, Contacts → wedge destinations
import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { render, screen } from '@testing-library/react-native';

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
    useRouter: () => ({ replace: jest.fn(), push: jest.fn() }),
    useLocalSearchParams: () => ({}),
    Stack: { Screen: jest.fn(() => null) },
  };
});

describe('crm/index redirect → /sales', () => {
  beforeEach(() => jest.clearAllMocks());
  it('renders a Redirect to /sales', () => {
    const { default: CrmRedirect } = require('../../../../app/(tabs)/crm/index');
    render(React.createElement(CrmRedirect));
    expect(screen.getByTestId('redirect-href').props.children).toBe('/sales');
  });
});

describe('workflows/index redirect → /inbox', () => {
  beforeEach(() => jest.clearAllMocks());
  it('renders a Redirect to /inbox', () => {
    const { default: WorkflowsRedirect } = require('../../../../app/(tabs)/workflows/index');
    render(React.createElement(WorkflowsRedirect));
    expect(screen.getByTestId('redirect-href').props.children).toBe('/inbox');
  });
});

describe('copilot/index redirect → /support', () => {
  beforeEach(() => jest.clearAllMocks());
  it('renders a Redirect to /support', () => {
    const { default: CopilotRedirect } = require('../../../../app/(tabs)/copilot/index');
    render(React.createElement(CopilotRedirect));
    expect(screen.getByTestId('redirect-href').props.children).toBe('/support');
  });
});

describe('contacts/index redirect → /sales', () => {
  beforeEach(() => jest.clearAllMocks());
  it('renders a Redirect to /sales', () => {
    const { default: ContactsRedirect } = require('../../../../app/(tabs)/contacts/index');
    render(React.createElement(ContactsRedirect));
    expect(screen.getByTestId('redirect-href').props.children).toBe('/sales');
  });
});
