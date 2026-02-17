import type { Config } from 'jest';

const config: Config = {
  preset: 'jest-expo',
  cacheDirectory: '<rootDir>/.jest-cache',
  setupFilesAfterEnv: ['<rootDir>/jest.setup.ts'],
  transformIgnorePatterns: [
    'node_modules/(?!((jest-)?react-native|@react-native(-community)?)|expo(nent)?|@expo(nent)?/.*|@expo-google-fonts/.*|react-navigation|@react-navigation/.*|@unimodules/.*|unimodules|sentry-expo|native-base|react-native-svg|react-native-paper|react-native-reanimated|react-native-gesture-handler|react-native-screens|@tanstack/react-query|zustand)',
  ],
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/**/index.{ts,tsx}',
  ],
  coverageThreshold: {
    global: {
      // Baseline gate for progressive adoption; raise incrementally each sprint.
      branches: 15,
      functions: 9,
      lines: 20,
      statements: 20,
    },
  },
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/$1',
  },
  // testEnvironment se detecta automáticamente con jest-expo
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
};

export default config;
