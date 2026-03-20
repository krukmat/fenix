// UC-S1 — Sales Copilot: smoke E2E for Copilot access from Account and Deal detail screens.
import { device, element, by, expect as detoxExpect, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';
import { ensureMobileP2Seed } from './helpers/seed.helper';

describe('UC-S1 — Sales Copilot: account and deal entry points', () => {
  const seeded = ensureMobileP2Seed();

  beforeAll(async () => {
    await loginAsTestUser();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  // --- Account → Copilot ---

  it('opens account detail screen', async () => {
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-accounts-tab')).tap();
    await waitFor(element(by.id(`accounts-list-item-${seeded.account.id}`)))
      .toBeVisible()
      .withTimeout(10000);
    await element(by.id(`accounts-list-item-${seeded.account.id}`)).tap();
    await waitFor(element(by.id('account-detail-screen')))
      .toBeVisible()
      .withTimeout(10000);
  });

  it('sees Copilot button on account detail', async () => {
    await waitFor(element(by.id('account-copilot-open-button')))
      .toBeVisible()
      .withTimeout(5000);
  });

  it('opens Copilot from account detail with entity context', async () => {
    await element(by.id('account-copilot-open-button')).tap();
    await waitFor(element(by.id('copilot-panel')))
      .toBeVisible()
      .withTimeout(10000);
  });

  // --- Deal → Copilot ---

  it('goes back and opens deal detail screen', async () => {
    await device.pressBack();
    await device.pressBack();
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-deals-tab')).tap();
    await waitFor(element(by.id(`deals-list-item-${seeded.deal.id}`)))
      .toBeVisible()
      .withTimeout(10000);
    await element(by.id(`deals-list-item-${seeded.deal.id}`)).tap();
    await waitFor(element(by.id('deal-detail-screen')))
      .toBeVisible()
      .withTimeout(10000);
  });

  it('sees Copilot button on deal detail', async () => {
    await waitFor(element(by.id('deal-copilot-open-button')))
      .toBeVisible()
      .withTimeout(5000);
  });

  it('opens Copilot from deal detail with entity context', async () => {
    await element(by.id('deal-copilot-open-button')).tap();
    await waitFor(element(by.id('copilot-panel')))
      .toBeVisible()
      .withTimeout(10000);
  });
});
