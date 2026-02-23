// Task 4.8 — E2E: Accounts (list → detail → timeline)
import { device, element, by, expect as detoxExpect } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';

describe('Accounts flow', () => {
  beforeAll(async () => {
    await loginAsTestUser();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  it('shows accounts list', async () => {
    await detoxExpect(element(by.id('accounts-list'))).toBeVisible();
  });

  it('opens first account detail', async () => {
    // Tap first account in list
    await element(by.id('accounts-list-item-0')).tap();
    await detoxExpect(element(by.id('account-detail-screen'))).toBeVisible(5000);
  });

  it('shows account timeline', async () => {
    // Timeline section must exist
    await detoxExpect(element(by.id('account-timeline-list'))).toBeVisible(5000);
  });
});