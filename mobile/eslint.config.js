// https://docs.expo.dev/guides/using-eslint/
const { defineConfig } = require('eslint/config');
const expoConfig = require('eslint-config-expo/flat');
const sonarjs = require('eslint-plugin-sonarjs');

module.exports = defineConfig([
  expoConfig,
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
      'max-lines': ['warn', { max: 300, skipBlankLines: true, skipComments: true }],

      // Maintainability gates
      '@typescript-eslint/no-explicit-any': 'error',
      'import/no-cycle': 'error',
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
    },
  },
]);
