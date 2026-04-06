// Task 4.8 — E2E auth helper
import { device, element, by, waitFor } from 'detox';
import { ensureMobileP2Seed } from './seed.helper';

const TEST_EMAIL = 'e2e@fenixcrm.test';
const TEST_PASSWORD = 'e2eTestPass123!';
const TEST_NAME = 'E2E Test User';

async function waitForPostLoginLanding(timeout = 30000) {
  const targets = [
    'authenticated-root',
    'drawer-open-button',
    'home-feed',
    'home-feed-flatlist',
    'home-feed-empty',
  ];

  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    for (const testID of targets) {
      try {
        await waitFor(element(by.id(testID))).toBeVisible().withTimeout(1000);
        return;
      } catch {
        // Try the next landing marker until timeout expires.
      }
    }
  }

  throw new Error(`Timed out waiting for post-login landing: ${targets.join(', ')}`);
}

async function waitForAnyVisible(testIDs: string[], timeout = 30000) {
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    for (const testID of testIDs) {
      try {
        await waitFor(element(by.id(testID))).toBeVisible().withTimeout(1000);
        return testID;
      } catch {
        // Keep probing until timeout expires.
      }
    }
  }

  throw new Error(`Timed out waiting for any of: ${testIDs.join(', ')}`);
}

/**
 * Registers a new user and lands on the authenticated home route.
 * Call once at the beginning of an E2E suite.
 */
export async function registerAndLogin(): Promise<void> {
  await device.launchApp({
    newInstance: true,
    launchArgs: { detoxEnableSynchronization: '0' },
  });
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
  await element(by.id('register-submit-button')).tap();

  await waitForPostLoginLanding(10000);
}

/**
 * Logs in with test credentials. App must be at login screen.
 */
export async function loginAsTestUser(): Promise<void> {
  const seeded = ensureMobileP2Seed();
  try {
    await waitForPostLoginLanding(3000);
    return;
  } catch {
    // Not already authenticated in the current session.
  }

  try {
    await waitFor(element(by.id('login-screen'))).toBeVisible().withTimeout(30000);
  } catch {
    for (let i = 0; i < 3; i += 1) {
      await device.pressBack();
      try {
        await waitForAnyVisible([
          'login-screen',
          'authenticated-root',
          'drawer-open-button',
          'home-feed',
          'home-feed-flatlist',
          'home-feed-empty',
        ], 3000);
        break;
      } catch {
        // Keep unwinding the current screen state.
      }
    }
  }

  try {
    await waitForPostLoginLanding(3000);
    return;
  } catch {
    // Not authenticated yet; continue to login form.
  }

  await element(by.id('login-email-input')).replaceText(seeded.credentials.email || TEST_EMAIL);
  await element(by.id('login-password-input')).replaceText(seeded.credentials.password || TEST_PASSWORD);
  await element(by.id('login-submit-button')).tap();

  // Screenshot suite: login + navigation can be slow on emulator with ANR noise.
  await waitForPostLoginLanding(60000);
}

/**
 * Logs out the current user.
 */
export async function logout(): Promise<void> {
  await element(by.id('drawer-open-button')).tap();
  await element(by.id('drawer-logout-button')).tap();
  await waitFor(element(by.id('login-screen'))).toBeVisible().withTimeout(5000);
}
