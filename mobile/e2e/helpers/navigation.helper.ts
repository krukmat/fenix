// Task 4.8 — E2E navigation helper
import { element, by, expect as detoxExpect } from 'detox';

/**
 * Opens the drawer menu and navigates to a named tab.
 * @param tabTestID - testID of the drawer item (e.g. 'drawer-cases-tab')
 */
export async function navigateTo(tabTestID: string): Promise<void> {
  const drawerButton = element(by.id('drawer-open-button'));
  await drawerButton.tap();
  await element(by.id(tabTestID)).tap();
}

/**
 * Waits for an element with the given testID to become visible.
 */
export async function waitForElement(testID: string, timeoutMs = 10000): Promise<void> {
  await detoxExpect(element(by.id(testID))).toBeVisible(timeoutMs);
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