import type { Config } from 'jest';

const config: Config = {
  preset: 'jest-expo',
  cacheDirectory: '<rootDir>/.jest-cache',
  setupFilesAfterEnv: ['<rootDir>/jest.logic.setup.js'],
  transformIgnorePatterns: [
    'node_modules/(?!((jest-)?react-native|@react-native(-community)?)|expo(nent)?|@expo(nent)?/.*|@expo-google-fonts/.*|react-navigation|@react-navigation/.*|@unimodules/.*|unimodules|sentry-expo|native-base|react-native-svg|react-native-paper|react-native-reanimated|react-native-gesture-handler|react-native-screens|@tanstack/react-query|zustand)',
  ],
  testMatch: [
    '<rootDir>/__tests__/services/**/*.test.ts',
    '<rootDir>/__tests__/stores/**/*.test.ts',
    '<rootDir>/__tests__/hooks/**/*.test.ts',
    '<rootDir>/__tests__/components/**/*.test.ts',
    '<rootDir>/__tests__/components/**/*.test.tsx',
    '<rootDir>/__tests__/navigation.test.tsx',
  ],
  testPathIgnorePatterns: [
    '<rootDir>/__tests__/components/ui.test.tsx',
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
