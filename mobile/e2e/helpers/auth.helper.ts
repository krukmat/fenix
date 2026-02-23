// Task 4.8 — E2E auth helper
import { device, element, by, expect as detoxExpect, waitFor } from 'detox';

const TEST_EMAIL = 'e2e@fenixcrm.test';
const TEST_PASSWORD = 'e2eTestPass123!';
const TEST_NAME = 'E2E Test User';

/**
 * Registers a new user and lands on the accounts tab.
 * Call once at the beginning of an E2E suite.
 */
export async function registerAndLogin(): Promise<void> {
  await device.launchApp({ newInstance: true });
  await device.disableSynchronization();

  // Navigate to register if not already there — use waitFor with short timeout
  try {
    await waitFor(element(by.id('go-to-register-link'))).toBeVisible().withTimeout(3000);
    await element(by.id('go-to-register-link')).tap();
  } catch {
    // Already on register screen or not on login — continue
  }

  await element(by.id('register-name-input')).typeText(TEST_NAME);
  await element(by.id('register-email-input')).typeText(TEST_EMAIL);
  await element(by.id('register-password-input')).typeText(TEST_PASSWORD);
  await element(by.id('register-submit-button')).tap();

  // Wait for accounts list to appear (authentication succeeded)
  await detoxExpect(element(by.id('accounts-list'))).toBeVisible(10000);
}

/**
 * Logs in with test credentials. App must be at login screen.
 */
export async function loginAsTestUser(): Promise<void> {
  await device.launchApp({ newInstance: true });
  await device.disableSynchronization();

  await element(by.id('login-email-input')).typeText(TEST_EMAIL);
  await element(by.id('login-password-input')).typeText(TEST_PASSWORD);
  await element(by.id('login-submit-button')).tap();

  await detoxExpect(element(by.id('accounts-list'))).toBeVisible(10000);
}

/**
 * Logs out the current user.
 */
export async function logout(): Promise<void> {
  await element(by.id('drawer-open-button')).tap();
  await element(by.id('drawer-logout-button')).tap();
  await detoxExpect(element(by.id('login-screen'))).toBeVisible(5000);
}