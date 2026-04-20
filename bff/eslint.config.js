// ESLint flat config for the BFF Express.js layer.
// The BFF is a thin proxy — zero business logic, zero DB access.
// Rules enforce production hygiene: no debug artifacts, strict types, no silence on errors.
const tsPlugin = require('@typescript-eslint/eslint-plugin');
const tsParser = require('@typescript-eslint/parser');
const sonarjs = require('eslint-plugin-sonarjs');

/** @type {import('eslint').Linter.FlatConfig[]} */
module.exports = [
  {
    ignores: ['dist/**', 'coverage/**', 'node_modules/**'],
  },
  {
    files: ['src/**/*.ts'],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        project: './tsconfig.json',
        tsconfigRootDir: __dirname,
      },
    },
    plugins: {
      '@typescript-eslint': tsPlugin,
      sonarjs,
    },
    rules: {
      // Production-grade: no debug artifacts
      'no-console': 'error',
      'no-debugger': 'error',

      // Production-grade: no implicit any — BFF routes must have typed req/res
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-non-null-assertion': 'error',

      // Production-grade: unused code
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', ignoreRestSiblings: true },
      ],

      // Production-grade: promise misuse — every async call must be awaited or chained
      '@typescript-eslint/no-floating-promises': 'error',
      // No misused promises as callbacks (express next(promise) footgun)
      '@typescript-eslint/no-misused-promises': 'error',

      // Production-grade: no return-await in try blocks (wraps resolved value in extra Promise)
      '@typescript-eslint/return-await': ['error', 'in-try-catch'],

      // Complexity gate — BFF handlers should be thin; extract if complex
      complexity: ['error', { max: 8 }],
      'sonarjs/cognitive-complexity': ['error', 10],
      'max-lines-per-function': ['error', { max: 60, skipBlankLines: true, skipComments: true }],
      'max-lines': ['error', { max: 200, skipBlankLines: true, skipComments: true }],

      // SonarJS smell detection
      'sonarjs/no-redundant-jump': 'error',
      'sonarjs/prefer-immediate-return': 'error',
      'sonarjs/no-ignored-return': 'error',
      'sonarjs/no-duplicate-string': ['error', { threshold: 3 }],
    },
  },
  {
    files: ['**/*.test.ts', 'tests/**/*.ts'],
    rules: {
      'no-console': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/no-floating-promises': 'off',
      complexity: 'off',
      'sonarjs/cognitive-complexity': 'off',
      'max-lines-per-function': 'off',
      'max-lines': 'off',
    },
  },
];
