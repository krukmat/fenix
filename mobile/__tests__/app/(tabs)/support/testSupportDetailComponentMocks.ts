export function mockSupportDetailCRMModule() {
  const React = require('react');
  const { View } = require('react-native');
  return {
    CRMDetailHeader: ({ testIDPrefix }: { testIDPrefix: string }) => React.createElement(View, { testID: `${testIDPrefix}-header` }),
    EntityTimeline: ({ testIDPrefix }: { testIDPrefix: string }) => React.createElement(View, { testID: `${testIDPrefix}-list` }),
  };
}

export function mockInlineSupportCopilotModule() {
  const React = require('react');
  const { View } = require('react-native');
  return { CopilotPanel: () => React.createElement(View, { testID: 'support-inline-copilot' }) };
}

export function mockSupportDetailPaperModule() {
  const React = require('react');
  const { TouchableOpacity, Text, View } = require('react-native');
  return {
    useTheme: () => ({ colors: { primary: '#E53935', surface: '#f5f5f5', onSurface: '#000', onSurfaceVariant: '#666', background: '#fff', error: '#B00020', surfaceVariant: '#ddd', outline: '#999', outlineVariant: '#ccc' } }),
    Button: ({ testID, onPress, children, disabled }: { testID: string; onPress: () => void; children: React.ReactNode; disabled?: boolean }) =>
      React.createElement(TouchableOpacity, { testID, onPress, accessibilityState: { disabled: !!disabled } }, React.createElement(Text, null, children)),
    ActivityIndicator: ({ testID }: { testID?: string }) => React.createElement(View, { testID }),
    Text: ({ children }: { children: React.ReactNode }) => React.createElement(Text, null, children),
  };
}
