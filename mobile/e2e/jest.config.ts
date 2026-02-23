// Task 4.8 — Jest config for Detox E2E tests
import type { Config } from 'jest';

const config: Config = {
  rootDir: '..',
  testMatch: ['<rootDir>/e2e/**/*.e2e.ts'],
  testTimeout: 120000,
  maxWorkers: 1,
  globalSetup: 'detox/runners/jest/globalSetup',
  globalTeardown: 'detox/runners/jest/globalTeardown',
  reporters: ['detox/runners/jest/reporter'],
  testEnvironment: 'detox/runners/jest/testEnvironment',
  verbose: true,
  transform: {
    '^.+\\.(ts|tsx)$': [
      'ts-jest',
      { tsconfig: { jsx: 'react' } },
    ],
  },
};

export default config;