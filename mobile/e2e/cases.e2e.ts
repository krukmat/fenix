// Task 4.8 — E2E: Case detail Agent Activity smoke
import { device, element, by, expect as detoxExpect, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';
import { ensureMobileP2Seed } from './helpers/seed.helper';

describe.skip('Case detail — Agent Activity', () => {
  const seeded = ensureMobileP2Seed();

  beforeAll(async () => {
    await loginAsTestUser();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  it('opens case detail screen', async () => {
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-cases-tab')).tap();
    await waitFor(element(by.id(`cases-list-item-${seeded.case.id}`)))
      .toBeVisible()
      .withTimeout(10000);
    await element(by.id(`cases-list-item-${seeded.case.id}`)).tap();
    await waitFor(element(by.id('case-detail-screen')))
      .toBeVisible()
      .withTimeout(10000);
  });

  it('shows Agent Activity section on case detail', async () => {
    await waitFor(element(by.id('case-agent-activity-section')))
      .toBeVisible()
      .withTimeout(10000);
  });

  it('navigates from case Agent Activity item to agent run detail', async () => {
    await element(by.id(`case-agent-activity-item-${seeded.agentRuns.caseRejectedId}`)).tap();
    await waitFor(element(by.id('agent-run-detail-screen')))
      .toBeVisible()
      .withTimeout(10000);
    await detoxExpect(element(by.id('run-status-chip'))).toBeVisible();
  });
});
