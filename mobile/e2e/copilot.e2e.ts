// Task 4.8 — E2E: Copilot panel (Case detail → ask question → SSE response + evidence)
import { device, element, by, expect as detoxExpect, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';

describe('Copilot panel', () => {
  beforeAll(async () => {
    await loginAsTestUser();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  it('navigates to cases list', async () => {
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-cases-tab')).tap();
    await detoxExpect(element(by.id('cases-list'))).toBeVisible(5000);
  });

  it('opens first case and sees Copilot button', async () => {
    // Tap first case in list
    await element(by.id('cases-list-item-0')).tap();
    await detoxExpect(element(by.id('case-detail-screen'))).toBeVisible(5000);

    // Copilot button must be visible
    await detoxExpect(element(by.id('copilot-open-button'))).toBeVisible();
  });

  it('opens Copilot panel, asks a question, sees streaming response', async () => {
    await element(by.id('copilot-open-button')).tap();
    await detoxExpect(element(by.id('copilot-panel'))).toBeVisible(3000);

    // Type and submit a question
    await element(by.id('copilot-input')).typeText('What is the status of this case?');
    await element(by.id('copilot-send-button')).tap();

    // Wait for at least one SSE token to appear (streaming in progress or complete)
    await waitFor(element(by.id('copilot-response-text')))
      .toBeVisible()
      .withTimeout(15000);
  });

  it('sees at least one evidence card after response', async () => {
    // Evidence cards appear after response completes
    await waitFor(element(by.id('evidence-card-0')))
      .toBeVisible()
      .withTimeout(20000);
  });
});