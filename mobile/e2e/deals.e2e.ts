// Task 4.8 — E2E: Deal detail Agent Activity smoke
// UC-S1: Deal detail → Open Copilot with context
import { device, element, by, expect as detoxExpect, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';
import { ensureMobileP2Seed } from './helpers/seed.helper';

describe.skip('Deal detail — Agent Activity', () => {
  const seeded = ensureMobileP2Seed();

  beforeAll(async () => {
    await loginAsTestUser();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  it('opens deal detail screen', async () => {
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

  it('shows Agent Activity section on deal detail', async () => {
    await waitFor(element(by.id('deal-agent-activity-section')))
      .toBeVisible()
      .withTimeout(10000);
  });

  it('navigates from deal Agent Activity item to agent run detail', async () => {
    await element(by.id(`deal-agent-activity-item-${seeded.agentRuns.dealRejectedId}`)).tap();
    await waitFor(element(by.id('agent-run-detail-screen')))
      .toBeVisible()
      .withTimeout(10000);
    await detoxExpect(element(by.id('run-status-chip'))).toBeVisible();
  });

  // UC-S1: Deal detail → Open Copilot with context
  it('opens Copilot from deal detail with deal context', async () => {
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

    await waitFor(element(by.id('deal-copilot-open-button')))
      .toBeVisible()
      .withTimeout(5000);
    await element(by.id('deal-copilot-open-button')).tap();

    await waitFor(element(by.id('copilot-panel')))
      .toBeVisible()
      .withTimeout(5000);
  });
});
