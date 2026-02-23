jest.mock('@sentry/react-native', () => ({
  init: jest.fn(),
  wrap: jest.fn((c) => c),
}));

import * as Sentry from '@sentry/react-native';

it('Sentry.init is called on app load', () => {
  require('../app/_layout');
  expect(Sentry.init).toHaveBeenCalledWith(
    expect.objectContaining({ tracesSampleRate: expect.any(Number) })
  );
});