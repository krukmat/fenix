// F4-T3: Wedge-first Detox navigation helper
import { element, by, waitFor } from 'detox';

const TAB_LABELS = {
  inbox: 'Inbox',
  support: 'Support',
  sales: 'Sales',
  activity: 'Activity',
  governance: 'Governance',
} as const;

const TAB_READY_MATCHERS = {
  inbox: by.id('inbox-filter-chips'),
  support: by.id('support-cases-search'),
  sales: by.id('sales-account-item-0'),
  activity: by.id('filter-all'),
  governance: by.id('governance-recent-usage'),
} as const;

export type WedgeTab = keyof typeof TAB_LABELS;

/**
 * Navigates to a wedge bottom tab and waits for its root screen.
 */
export async function navigateToTab(tab: WedgeTab): Promise<void> {
  await element(by.text(TAB_LABELS[tab])).tap();
  await waitFor(element(TAB_READY_MATCHERS[tab])).toBeVisible().withTimeout(15000);
}

/**
 * Waits for an element with the given testID to become visible.
 */
export async function waitForElement(testID: string, timeoutMs = 10000): Promise<void> {
  await waitFor(element(by.id(testID))).toBeVisible().withTimeout(timeoutMs);
}

/**
 * Taps an element by testID.
 */
export async function tapElement(testID: string): Promise<void> {
  await element(by.id(testID)).tap();
}

/**
 * Types text into an input element by testID.
 */
export async function typeText(testID: string, text: string): Promise<void> {
  await element(by.id(testID)).typeText(text);
}
