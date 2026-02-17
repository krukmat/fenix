// Task 4.2 — FR-300: Mocks globales para Jest

// Mock reanimated (no puede correr en Jest)
jest.mock('react-native-reanimated', () => {
  const Reanimated = require('react-native-reanimated/mock');
  Reanimated.default.call = () => {};
  return Reanimated;
});

// Mock expo-secure-store (usa Android Keystore — no disponible en Jest)
jest.mock('expo-secure-store', () => ({
  setItemAsync: jest.fn().mockResolvedValue(undefined),
  getItemAsync: jest.fn().mockResolvedValue(null),
  deleteItemAsync: jest.fn().mockResolvedValue(undefined),
}));

// Mock expo-splash-screen
jest.mock('expo-splash-screen', () => ({
  preventAutoHideAsync: jest.fn().mockResolvedValue(undefined),
  hideAsync: jest.fn().mockResolvedValue(undefined),
}));

// Mock react-native-gesture-handler
jest.mock('react-native-gesture-handler', () => {
  const React = require('react');
  const View = ({ children }: { children?: React.ReactNode }) => children ?? null;
  return {
    Swipeable: View,
    DrawerLayout: View,
    State: {},
    ScrollView: View,
    Slider: View,
    Switch: View,
    TextInput: View,
    ToolbarAndroid: View,
    ViewPagerAndroid: View,
    DrawerLayoutAndroid: View,
    WebView: View,
    NativeViewGestureHandler: View,
    TapGestureHandler: View,
    FlingGestureHandler: View,
    ForceTouchGestureHandler: View,
    LongPressGestureHandler: View,
    PanGestureHandler: View,
    PinchGestureHandler: View,
    RotationGestureHandler: View,
    RawButton: View,
    BaseButton: View,
    RectButton: View,
    BorderlessButton: View,
    FlatList: View,
    gestureHandlerRootHOC: jest.fn(),
    Directions: {},
  };
});

// Mock react-native-screens
jest.mock('react-native-screens', () => {
  const React = require('react');
  const mockView = ({ children }: { children?: React.ReactNode }) => children ?? null;
  return {
    enableScreens: jest.fn(),
    Screen: mockView,
    ScreenContainer: mockView,
    NativeScreen: mockView,
    NativeScreenContainer: mockView,
    ScreenStack: mockView,
  };
});

// Mock @react-navigation/native
jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({
    navigate: jest.fn(),
    goBack: jest.fn(),
  }),
  useRoute: () => ({
    params: {},
  }),
  NavigationContainer: ({ children }: { children: React.ReactNode }) => children,
}));

// Mock expo-router
jest.mock('expo-router', () => ({
  useRouter: () => ({
    push: jest.fn(),
    replace: jest.fn(),
    back: jest.fn(),
  }),
  useLocalSearchParams: () => ({}),
  Link: ({ children }: { children: React.ReactNode }) => children,
  Redirect: ({ href }: { href: string }) => null,
  Tabs: ({ children }: { children: React.ReactNode }) => children,
  Stack: ({ children }: { children: React.ReactNode }) => children,
  Slot: () => null,
}));

// Silence console warnings during tests
global.console = {
  ...console,
  warn: jest.fn(),
  error: jest.fn(),
};
