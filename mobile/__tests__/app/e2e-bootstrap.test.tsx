/**
 * docs/plans/maestro-screenshot-auth-bypass-plan.md — Task 2
 *
 * Governance guard test for mobile/app/e2e-bootstrap.tsx.
 *
 * Rationale:
 * The e2e-bootstrap route accepts a token/userId/workspaceId via query params
 * and calls authStore.login() to inject an authenticated session without
 * going through the login UI. This is required by the screenshot runner but
 * is a direct governance risk if a production build ever ships with the
 * route reachable: it would be an unconditional auth-injection surface,
 * which violates the "Tools, not mutations" principle from CLAUDE.md.
 *
 * The route MUST therefore be gated behind EXPO_PUBLIC_E2E_MODE=1. When the
 * flag is off, the route must render nothing useful and redirect to /login
 * without mutating auth state.
 *
 * These tests lock that contract.
 */

import { describe, it, expect, beforeEach, afterEach, jest } from '@jest/globals';
import React from 'react';
import { render } from '@testing-library/react-native';

// Jest disallows referencing out-of-scope vars inside jest.mock factories
// unless they're prefixed with `mock`. We expose a shared registry on
// globalThis so both the factory and the tests can mutate/observe it.
type BootstrapMocks = {
  replace: jest.Mock;
  login: jest.Mock;
  params: Record<string, string | string[] | undefined>;
};

(globalThis as unknown as { __bootstrapMocks: BootstrapMocks }).__bootstrapMocks = {
  replace: jest.fn(),
  login: jest.fn(async () => {}),
  params: {},
};

const getMocks = () =>
  (globalThis as unknown as { __bootstrapMocks: BootstrapMocks }).__bootstrapMocks;

jest.mock('expo-router', () => {
  const registry = (globalThis as unknown as { __bootstrapMocks: BootstrapMocks })
    .__bootstrapMocks;
  return {
    __esModule: true,
    useRouter: () => ({ replace: registry.replace, push: jest.fn() }),
    useLocalSearchParams: () => registry.params,
  };
});

jest.mock('../../src/stores/authStore', () => {
  const registry = (globalThis as unknown as { __bootstrapMocks: BootstrapMocks })
    .__bootstrapMocks;
  return {
    __esModule: true,
    useAuthStore: (selector: (state: { login: jest.Mock }) => unknown) =>
      selector({ login: registry.login }),
  };
});

function setParams(params: Record<string, string | string[] | undefined>) {
  getMocks().params = params;
}

// Import the component AFTER the mocks are registered.
// eslint-disable-next-line @typescript-eslint/no-var-requires
const E2EBootstrapRoute = require('../../app/e2e-bootstrap').default;

describe('e2e-bootstrap governance gate', () => {
  const originalE2EFlag = process.env.EXPO_PUBLIC_E2E_MODE;

  beforeEach(() => {
    getMocks().replace.mockClear();
    getMocks().login.mockClear();
    setParams({});
  });

  afterEach(() => {
    process.env.EXPO_PUBLIC_E2E_MODE = originalE2EFlag;
  });

  it('redirects to /login without calling login() when EXPO_PUBLIC_E2E_MODE is off', async () => {
    process.env.EXPO_PUBLIC_E2E_MODE = '0';
    setParams({
      token: 'tok-abc.def.ghi',
      userId: 'user-xyz',
      workspaceId: 'ws-xyz',
      redirect: '/inbox',
    });

    render(<E2EBootstrapRoute />);

    // Let the useEffect bootstrap promise settle.
    await Promise.resolve();
    await Promise.resolve();

    expect(getMocks().login).not.toHaveBeenCalled();
    expect(getMocks().replace).toHaveBeenCalledWith('/login');
  });

  it('redirects to /login when EXPO_PUBLIC_E2E_MODE is unset', async () => {
    delete process.env.EXPO_PUBLIC_E2E_MODE;
    setParams({
      token: 'tok-abc.def.ghi',
      userId: 'user-xyz',
      workspaceId: 'ws-xyz',
    });

    render(<E2EBootstrapRoute />);

    await Promise.resolve();
    await Promise.resolve();

    expect(getMocks().login).not.toHaveBeenCalled();
    expect(getMocks().replace).toHaveBeenCalledWith('/login');
  });

  it('calls login() and redirects to the requested route when E2E mode is on', async () => {
    process.env.EXPO_PUBLIC_E2E_MODE = '1';
    setParams({
      token: 'tok-abc.def.ghi',
      userId: 'user-xyz',
      workspaceId: 'ws-xyz',
      redirect: '/inbox',
    });

    render(<E2EBootstrapRoute />);

    await Promise.resolve();
    await Promise.resolve();

    expect(getMocks().login).toHaveBeenCalledWith({
      token: 'tok-abc.def.ghi',
      userId: 'user-xyz',
      workspaceId: 'ws-xyz',
    });
    expect(getMocks().replace).toHaveBeenCalledWith('/inbox');
  });

  it('redirects to /login when any required param is missing in E2E mode', async () => {
    process.env.EXPO_PUBLIC_E2E_MODE = '1';
    setParams({
      token: 'tok-abc.def.ghi',
      // userId missing
      workspaceId: 'ws-xyz',
    });

    render(<E2EBootstrapRoute />);

    await Promise.resolve();
    await Promise.resolve();

    expect(getMocks().login).not.toHaveBeenCalled();
    expect(getMocks().replace).toHaveBeenCalledWith('/login');
  });
});
