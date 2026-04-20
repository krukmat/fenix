// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require('eslint/config');
const expoConfig = require('eslint-config-expo/flat');
const sonarjs = require('eslint-plugin-sonarjs');

module.exports = defineConfig([
  expoConfig,
  {
    linterOptions: {
      noInlineConfig: true,
      reportUnusedDisableDirectives: 'error',
    },
  },
  {
    plugins: {
      sonarjs,
    },
    ignores: ['dist/*', 'coverage/*'],
    rules: {
      // Complexity gates
      complexity: ['error', { max: 10 }],
      'sonarjs/cognitive-complexity': ['error', 15],
      'max-lines-per-function': ['error', { max: 80, skipBlankLines: true, skipComments: true }],
      'max-lines': ['error', { max: 300, skipBlankLines: true, skipComments: true }],

      // Maintainability gates
      '@typescript-eslint/no-explicit-any': 'error',
      'import/no-cycle': 'error',

      // Production-grade: no debug artifacts in shipped code
      'no-console': 'error',
      'no-debugger': 'error',

      // Production-grade: unused code is dead weight and confuses readers
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', ignoreRestSiblings: true },
      ],

      // Production-grade: React hooks correctness
      // Missing deps cause stale closures — silent bugs that are hard to reproduce in prod
      'react-hooks/exhaustive-deps': 'error',
      // Hook call order must be stable across renders
      'react-hooks/rules-of-hooks': 'error',

      // Production-grade: no non-null assertions — replace with proper guards
      '@typescript-eslint/no-non-null-assertion': 'error',

      // Production-grade: sonarjs smell detection
      'sonarjs/no-redundant-jump': 'error',
      'sonarjs/prefer-immediate-return': 'error',
      'sonarjs/no-ignored-return': 'error',   // ignoring return values of pure functions
      'sonarjs/no-duplicate-string': ['error', { threshold: 3 }],
    },
  },
  {
    files: ['**/*.e2e.{ts,tsx}'],
    rules: {
      complexity: 'off',
      'sonarjs/cognitive-complexity': 'off',
      'max-lines-per-function': 'off',
      'max-lines': 'off',
      'no-console': 'off',
    },
  },
  {
    files: ['**/__tests__/**/*.{ts,tsx}', '**/*.test.{ts,tsx}'],
    rules: {
      // Tests can be more verbose/branchy
      complexity: 'off',
      'sonarjs/cognitive-complexity': 'off',
      'max-lines-per-function': 'off',
      'max-lines': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      'import/no-cycle': 'off',
      'no-console': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
    },
  },
]);
