// Task 4.8 — E2E: Auth flow (Register → Login → accounts list)
import { device, element, by, expect as detoxExpect } from 'detox';

describe('Auth flow', () => {
  beforeAll(async () => {
    // React Query polling + Reanimated animations keep the event queue busy.
    // Launch without sync then disable it immediately to avoid the 45s idle timeout.
    await device.launchApp({ newInstance: true });
    await device.disableSynchronization();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  it('shows login screen on first launch', async () => {
    await detoxExpect(element(by.id('login-screen'))).toBeVisible();
  });

  it('navigates to register screen', async () => {
    await element(by.id('go-to-register-link')).tap();
    await detoxExpect(element(by.id('register-screen'))).toBeVisible();
  });

  it('registers a new user and lands on accounts list', async () => {
    await element(by.id('register-name-input')).typeText('E2E User');
    await element(by.id('register-email-input')).typeText('e2e+auth@fenixcrm.test');
    await element(by.id('register-password-input')).typeText('TestPass123!');
    await element(by.id('register-submit-button')).tap();

    await detoxExpect(element(by.id('accounts-list'))).toBeVisible(15000);
  });

  it('logs out and can log back in', async () => {
    // Open drawer → logout
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-logout-button')).tap();

    // Should return to login screen
    await detoxExpect(element(by.id('login-screen'))).toBeVisible(5000);

    // Log in again
    await element(by.id('login-email-input')).typeText('e2e+auth@fenixcrm.test');
    await element(by.id('login-password-input')).typeText('TestPass123!');
    await element(by.id('login-submit-button')).tap();

    await detoxExpect(element(by.id('accounts-list'))).toBeVisible(15000);
  });
});