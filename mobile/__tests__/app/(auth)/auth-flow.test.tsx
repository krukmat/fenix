import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import React from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react-native';

const mockLoginApi = jest.fn();
const mockRegisterApi = jest.fn();
const mockStoreLogin = jest.fn();

jest.mock('expo-router', () => ({
  __esModule: true,
  useRouter: () => ({ push: jest.fn() }),
}));

jest.mock('../../../src/services/api', () => ({
  authApi: {
    login: (...args: unknown[]) => mockLoginApi(...args),
    register: (...args: unknown[]) => mockRegisterApi(...args),
  },
}));

jest.mock('../../../src/stores/authStore', () => ({
  useAuthStore: (selector: (state: { login: typeof mockStoreLogin }) => unknown) =>
    selector({ login: mockStoreLogin }),
}));

function renderLoginScreen() {
  const { default: Screen } = require('../../../app/(auth)/login');
  render(React.createElement(Screen));
}

function renderRegisterScreen() {
  const { default: Screen } = require('../../../app/(auth)/register');
  render(React.createElement(Screen));
}

describe('auth screens', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockStoreLogin.mockResolvedValue(undefined);
  });

  it('logs in through auth store without imperative navigation', async () => {
    mockLoginApi.mockResolvedValue({
      token: 'token-1',
      userId: 'user-1',
      workspaceId: 'ws-1',
    });

    renderLoginScreen();

    fireEvent.changeText(screen.getByTestId('login-email-input'), 'e2e@fenixcrm.test');
    fireEvent.changeText(screen.getByTestId('login-password-input'), 'e2eTestPass123!');
    fireEvent.press(screen.getByTestId('login-submit-button'));

    await waitFor(() => {
      expect(mockLoginApi).toHaveBeenCalledWith('e2e@fenixcrm.test', 'e2eTestPass123!');
      expect(mockStoreLogin).toHaveBeenCalledWith({
        token: 'token-1',
        userId: 'user-1',
        workspaceId: 'ws-1',
      });
    });
  });

  it('registers through auth store without imperative navigation', async () => {
    mockRegisterApi.mockResolvedValue({
      token: 'token-2',
      userId: 'user-2',
      workspaceId: 'ws-2',
    });

    renderRegisterScreen();

    fireEvent.changeText(screen.getByTestId('register-name-input'), 'E2E User');
    fireEvent.changeText(screen.getByTestId('register-email-input'), 'e2e@fenixcrm.test');
    fireEvent.changeText(screen.getByTestId('register-workspace-input'), 'E2E Workspace');
    fireEvent.changeText(screen.getByTestId('register-password-input'), 'e2eTestPass123!');
    fireEvent.press(screen.getByTestId('register-submit-button'));

    await waitFor(() => {
      expect(mockRegisterApi).toHaveBeenCalledWith(
        'E2E User',
        'e2e@fenixcrm.test',
        'e2eTestPass123!',
        'E2E Workspace',
      );
      expect(mockStoreLogin).toHaveBeenCalledWith({
        token: 'token-2',
        userId: 'user-2',
        workspaceId: 'ws-2',
      });
    });
  });
});
