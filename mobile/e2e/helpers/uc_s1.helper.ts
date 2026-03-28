import { by, device, element, expect as detoxExpect, waitFor } from 'detox';

import { loginAsTestUser } from './auth.helper';
import { ensureMobileP2Seed } from './seed.helper';

const seeded = ensureMobileP2Seed();

export async function startUCS1Session(): Promise<void> {
  await loginAsTestUser();
}

export async function finishUCS1Session(): Promise<void> {
  await device.terminateApp();
}

export async function openAccountDetailForUCS1(): Promise<void> {
  await element(by.id('drawer-open-button')).tap();
  await element(by.id('drawer-accounts-tab')).tap();
  await waitFor(element(by.id(`accounts-list-item-${seeded.account.id}`)))
    .toBeVisible()
    .withTimeout(10000);
  await element(by.id(`accounts-list-item-${seeded.account.id}`)).tap();
  await waitFor(element(by.id('account-detail-screen')))
    .toBeVisible()
    .withTimeout(10000);
}

export async function openCopilotFromAccountDetail(): Promise<void> {
  await waitFor(element(by.id('account-copilot-open-button')))
    .toBeVisible()
    .withTimeout(5000);
  await element(by.id('account-copilot-open-button')).tap();
  await waitFor(element(by.id('copilot-panel')))
    .toBeVisible()
    .withTimeout(10000);
}

export async function openDealDetailForUCS1(): Promise<void> {
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
}

export async function openCopilotFromDealDetail(): Promise<void> {
  await waitFor(element(by.id('deal-copilot-open-button')))
    .toBeVisible()
    .withTimeout(5000);
  await element(by.id('deal-copilot-open-button')).tap();
  await waitFor(element(by.id('copilot-panel')))
    .toBeVisible()
    .withTimeout(10000);
}

export async function assertCopilotPanelVisible(): Promise<void> {
  await detoxExpect(element(by.id('copilot-panel'))).toBeVisible();
}

export async function assertCopilotResponseSurfaces(): Promise<void> {
  await waitFor(element(by.id('copilot-response-text')))
    .toBeVisible()
    .withTimeout(10000);
}
