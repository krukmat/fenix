// Task 4.8 — E2E: Accounts (list → detail → timeline)
// UC-S1: Account detail → Open Copilot with context
import { device, element, by, expect as detoxExpect, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';
import { ensureMobileP2Seed } from './helpers/seed.helper';

describe('Accounts flow', () => {
  const seeded = ensureMobileP2Seed();

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

  it('shows Agent Activity for a seeded account and navigates to run detail', async () => {
    await device.pressBack();
    await detoxExpect(element(by.id('accounts-list'))).toBeVisible(5000);

    await element(by.id(`accounts-list-item-${seeded.account.id}`)).tap();
    await detoxExpect(element(by.id('account-detail-screen'))).toBeVisible(5000);
    await detoxExpect(element(by.id('agent-activity-section'))).toBeVisible(5000);

    await element(by.id(`agent-activity-item-${seeded.agentRuns.rejectedId}`)).tap();
    await detoxExpect(element(by.id('agent-run-detail-screen'))).toBeVisible(5000);
  });

  // UC-S1: Account detail → Open Copilot with context
  it('opens Copilot from account detail with account context', async () => {
    await device.pressBack();
    await waitFor(element(by.id('accounts-list')))
      .toBeVisible()
      .withTimeout(10000);

    await element(by.id(`accounts-list-item-${seeded.account.id}`)).tap();
    await waitFor(element(by.id('account-detail-screen')))
      .toBeVisible()
      .withTimeout(10000);

    await waitFor(element(by.id('account-copilot-open-button')))
      .toBeVisible()
      .withTimeout(5000);
    await element(by.id('account-copilot-open-button')).tap();

    await waitFor(element(by.id('copilot-panel')))
      .toBeVisible()
      .withTimeout(5000);
  });
});
