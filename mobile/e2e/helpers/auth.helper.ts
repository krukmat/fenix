// Task 4.8 — E2E auth helper
import { device, element, by, waitFor } from 'detox';
import { ensureMobileP2Seed } from './seed.helper';

const TEST_EMAIL = 'e2e@fenixcrm.test';
const TEST_PASSWORD = 'e2eTestPass123!';
const TEST_NAME = 'E2E Test User';

async function dismissKeyboard() {
  await device.pressBack();
}

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
  // Task 4.8 — workspace field is required by RegisterForm
  await element(by.id('register-workspace-input')).typeText('E2E Test Workspace');
  await element(by.id('register-password-input')).typeText(TEST_PASSWORD);
  await dismissKeyboard();
  await element(by.id('register-submit-button')).tap();

  // Wait for accounts list to appear (authentication succeeded)
  await waitFor(element(by.id('accounts-list'))).toBeVisible().withTimeout(10000);
}

/**
 * Logs in with test credentials. App must be at login screen.
 */
export async function loginAsTestUser(): Promise<void> {
  const seeded = ensureMobileP2Seed();
  await device.launchApp({ newInstance: true });
  await device.disableSynchronization();

  try {
    await waitFor(element(by.id('login-screen'))).toBeVisible().withTimeout(30000);
  } catch {
    try {
      await waitFor(element(by.id('accounts-list'))).toBeVisible().withTimeout(30000);
      return;
    } catch {
      // fall through to relaunch and retry login
    }
    await device.terminateApp();
    await device.launchApp({ newInstance: true });
    await device.disableSynchronization();
    try {
      await waitFor(element(by.id('accounts-list'))).toBeVisible().withTimeout(30000);
      return;
    } catch {
      // fall through to login form
    }
  }

  await element(by.id('login-email-input')).replaceText(seeded.credentials.email || TEST_EMAIL);
  await element(by.id('login-password-input')).replaceText(seeded.credentials.password || TEST_PASSWORD);
  await dismissKeyboard();
  await element(by.id('login-submit-button')).tap();

  // Screenshot suite: login + navigation to accounts-list can be slow on emulator with ANR noise
  await waitFor(element(by.id('accounts-list'))).toBeVisible().withTimeout(60000);
}

/**
 * Logs out the current user.
 */
export async function logout(): Promise<void> {
  await element(by.id('drawer-open-button')).tap();
  await element(by.id('drawer-logout-button')).tap();
  await waitFor(element(by.id('login-screen'))).toBeVisible().withTimeout(5000);
}
