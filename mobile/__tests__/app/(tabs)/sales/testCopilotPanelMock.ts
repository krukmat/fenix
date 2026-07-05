// Shared jest mock factory for '../../../../src/components/copilot' across sales
// detail screen tests, avoiding a duplicated CopilotPanel stub per test file.
export function mockCopilotPanelModule(testID: string) {
  const React = require('react');
  const { View } = require('react-native');
  return {
    CopilotPanel: ({ initialContext }: { initialContext?: { entityType?: string; entityId?: string } }) =>
      React.createElement(View, {
        testID,
        accessibilityLabel: `${initialContext?.entityType ?? ''}:${initialContext?.entityId ?? ''}`,
      }),
  };
}
