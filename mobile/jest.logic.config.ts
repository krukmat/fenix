import type { Config } from 'jest';

const config: Config = {
  cacheDirectory: '<rootDir>/.jest-cache',
  testEnvironment: 'node',
  setupFilesAfterEnv: ['<rootDir>/jest.logic.setup.js'],
  transform: {
    '^.+\\.(ts|tsx)$': [
      'ts-jest',
      {
        tsconfig: '<rootDir>/tsconfig.app.json',
      },
    ],
  },
  testMatch: [
    '<rootDir>/__tests__/services/**/*.test.ts',
    '<rootDir>/__tests__/stores/**/*.test.ts',
    '<rootDir>/__tests__/hooks/**/*.test.ts',
    '<rootDir>/__tests__/navigation.test.tsx',
  ],
  collectCoverageFrom: [
    'src/hooks/**/*.{ts,tsx}',
    'src/services/**/*.{ts,tsx}',
    'src/stores/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/**/index.{ts,tsx}',
  ],
  coverageProvider: 'v8',
  coverageThreshold: {
    global: {
      branches: 25,
      functions: 20,
      lines: 35,
      statements: 35,
    },
  },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/$1',
  },
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
};

export default config;
