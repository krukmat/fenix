// Create and edit entry points redirect tests — non-wedge screens redirect to wedge destinations
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
    useLocalSearchParams: () => ({ id: 'test-id' }),
    Stack: { Screen: jest.fn(() => null) },
  };
});

function getHref(): string {
  return screen.getByTestId('redirect-href').props.children as string;
}

describe('cases/new redirect → /support', () => {
  beforeEach(() => jest.clearAllMocks());
  it('redirects to /support', () => {
    const { default: S } = require('../../../../app/(tabs)/cases/new');
    render(React.createElement(S));
    expect(getHref()).toBe('/support');
  });
});

describe('cases/edit/[id] redirect → /support', () => {
  beforeEach(() => jest.clearAllMocks());
  it('redirects to /support', () => {
    const { default: S } = require('../../../../app/(tabs)/cases/edit/[id]');
    render(React.createElement(S));
    expect(getHref()).toBe('/support');
  });
});

describe('deals/new redirect → /sales', () => {
  beforeEach(() => jest.clearAllMocks());
  it('redirects to /sales', () => {
    const { default: S } = require('../../../../app/(tabs)/deals/new');
    render(React.createElement(S));
    expect(getHref()).toBe('/sales');
  });
});

describe('deals/edit/[id] redirect → /sales', () => {
  beforeEach(() => jest.clearAllMocks());
  it('redirects to /sales', () => {
    const { default: S } = require('../../../../app/(tabs)/deals/edit/[id]');
    render(React.createElement(S));
    expect(getHref()).toBe('/sales');
  });
});

describe('accounts/new redirect → /sales', () => {
  beforeEach(() => jest.clearAllMocks());
  it('redirects to /sales', () => {
    const { default: S } = require('../../../../app/(tabs)/accounts/new');
    render(React.createElement(S));
    expect(getHref()).toBe('/sales');
  });
});

describe('workflows/new redirect → /inbox', () => {
  beforeEach(() => jest.clearAllMocks());
  it('redirects to /inbox', () => {
    const { default: S } = require('../../../../app/(tabs)/workflows/new');
    render(React.createElement(S));
    expect(getHref()).toBe('/inbox');
  });
});

describe('workflows/edit/[id] redirect → /inbox', () => {
  beforeEach(() => jest.clearAllMocks());
  it('redirects to /inbox', () => {
    const { default: S } = require('../../../../app/(tabs)/workflows/edit/[id]');
    render(React.createElement(S));
    expect(getHref()).toBe('/inbox');
  });
});
