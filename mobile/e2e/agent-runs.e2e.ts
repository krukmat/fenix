// Task 4.8 — E2E: Agent Runs (list → trigger → detail)
import { device, element, by, expect as detoxExpect, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';
import { ensureMobileP2Seed } from './helpers/seed.helper';

describe('Agent Runs flow', () => {
  const seeded = ensureMobileP2Seed();

  beforeAll(async () => {
    await loginAsTestUser();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  it('navigates to agent runs list', async () => {
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-activity-tab')).tap();
    await detoxExpect(element(by.id('agent-runs-list-screen'))).toBeVisible(5000);
  });

  it('triggers a support agent run and navigates to detail', async () => {
    await element(by.id('trigger-agent-button')).tap();
    await detoxExpect(element(by.id('trigger-agent-modal'))).toBeVisible(3000);

    // Select support agent and confirm
    await element(by.id('agent-select-support')).tap();
    await element(by.id('trigger-confirm-button')).tap();

    // TriggerAgentButton navigates to /agents/${id} on success — verify detail screen
    await waitFor(element(by.id('agent-run-detail-screen')))
      .toBeVisible()
      .withTimeout(10000);
  });

  it('detail screen shows status chip', async () => {
    // Status chip must be visible (running, success, failed, or abstained)
    await detoxExpect(element(by.id('run-status-chip'))).toBeVisible();
  });

  it('opens a rejected run fixture and shows rejection reason', async () => {
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-activity-tab')).tap();
    await waitFor(element(by.id(`agent-run-item-${seeded.agentRuns.rejectedId}`)))
      .toBeVisible()
      .withTimeout(10000);

    await element(by.id(`agent-run-item-${seeded.agentRuns.rejectedId}`)).tap();

    await waitFor(element(by.id('agent-run-detail-screen')))
      .toBeVisible()
      .withTimeout(10000);
    await detoxExpect(element(by.id('run-status-chip'))).toBeVisible();
    await detoxExpect(element(by.id('agent-run-rejection-reason'))).toBeVisible();
  });

  it('can navigate back to runs list', async () => {
    // Navigate back and verify list is still accessible
    await device.pressBack();
    await detoxExpect(element(by.id('agent-runs-list-screen'))).toBeVisible(5000);
  });
});
