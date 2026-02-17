import type { Config } from 'jest';

const config: Config = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/tests'],
  testMatch: ['**/*.test.ts'],
  transform: {
    '^.+\\.ts$': 'ts-jest',
  },
  moduleFileExtensions: ['ts', 'js', 'json'],
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/server.ts',    // entry point excluded — no lógica testeable
    '!src/config.ts',    // env-var config excluded — branches depend on process.env
  ],
  coverageThreshold: {
    global: {
      // branches: 75% — proxy and aggregated have catch/network error branches
      // reachable only with live infrastructure (excluded via istanbul ignore)
      branches: 75,
      functions: 80,
      lines: 80,
      statements: 80,
    },
  },
  coverageReporters: ['text', 'lcov'],
  testTimeout: 10000,
};

export default config;
