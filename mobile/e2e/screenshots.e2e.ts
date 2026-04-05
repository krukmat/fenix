import { by, device, element, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';
import { ensureMobileP2Seed } from './helpers/seed.helper';
import { ensureFirstActiveSignalId } from './helpers/screenshots.helper';

const seeded = ensureMobileP2Seed();
async function waitForVisible(testID: string, timeout = 30000) {
  await waitFor(element(by.id(testID))).toBeVisible().withTimeout(timeout);
}
async function waitForAnyVisible(testIDs: string[], timeout = 30000) {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeout) {
    for (const testID of testIDs) {
      if (await isVisible(testID, 250)) return testID;
    }
  }
  throw new Error(`Timed out waiting for any of: ${testIDs.join(', ')}`);
}
async function dismissKeyboard() { await device.pressBack(); }
async function isVisible(testID: string, timeout = 1000) {
  try {
    await waitForVisible(testID, timeout);
    return true;
  } catch {
    return false;
  }
}
async function relaunchAuthenticatedSession() {
  await loginAsTestUser();
}
async function swipeOpenDrawer() {
  const candidates = [
    'accounts-list',
    'accounts-flatlist',
    'home-feed',
    'home-feed-flatlist',
    'crm-hub',
    'contacts-list',
    'contacts-flatlist',
    'deals-list',
    'deals-flatlist',
    'cases-list',
    'cases-flatlist',
    'workflows-list',
    'workflows-flatlist',
    'agent-runs-list-screen',
    'copilot-panel',
  ];
  for (const testID of candidates) {
    if (await isVisible(testID, 300)) {
      await element(by.id(testID)).swipe('right', 'fast', 0.2);
      return;
    }
  }
  throw new Error('Unable to find a visible screen container to open the drawer');
}
async function openDrawer() {
  try {
    await waitForVisible('drawer-content', 1000);
    return;
  } catch {
    if (await isVisible('drawer-open-button', 500)) {
      await element(by.id('drawer-open-button')).tap();
    } else {
      await swipeOpenDrawer();
    }
    await waitForVisible('drawer-content', 3000);
  }
}
async function closeDrawerIfOpen() {
  if (await isVisible('drawer-content', 500)) {
    await device.pressBack();
  }
}
async function ensureCrmSubmenuVisible() {
  await openDrawer();
  try {
    await waitForVisible('drawer-crm-submenu', 1000);
    return;
  } catch {
    await element(by.id('drawer-crm-tab')).tap();
    await waitForVisible('drawer-crm-submenu', 3000);
  }
}
async function navigateToDrawerTab(tabTestID: string, targetTestIDs: string[], timeout = 30000) {
  await openDrawer();
  await element(by.id(tabTestID)).tap();
  await closeDrawerIfOpen();
  await waitForAnyVisible(targetTestIDs, timeout);
}
async function navigateToCrmSection(itemTestID: string, targetTestIDs: string[], timeout = 30000) {
  await ensureCrmSubmenuVisible();
  await element(by.id(itemTestID)).tap();
  await closeDrawerIfOpen();
  await waitForAnyVisible(targetTestIDs, timeout);
}
async function scrollToVisible(testID: string, scrollableTestID: string) {
  await waitFor(element(by.id(testID)))
    .toBeVisible()
    .whileElement(by.id(scrollableTestID))
    .scroll(200, 'down');
}
async function showHome() {
  await navigateToDrawerTab('drawer-home-tab', ['home-feed', 'home-feed-flatlist', 'home-feed-empty']);
}
async function showAccountsList() {
  await navigateToCrmSection('drawer-crm-accounts', ['accounts-list', 'accounts-flatlist', 'accounts-empty']);
}
async function showContactsList() {
  await navigateToCrmSection('drawer-crm-contacts', ['contacts-list', 'contacts-flatlist', 'contacts-empty']);
}
async function showDealsList() {
  await navigateToCrmSection('drawer-crm-deals', ['deals-list', 'deals-flatlist', 'deals-empty']);
}
async function showCasesList() {
  await navigateToCrmSection('drawer-crm-cases', ['cases-list', 'cases-flatlist', 'cases-empty']);
}
async function showCopilotPanel() {
  await navigateToDrawerTab('drawer-copilot-tab', ['copilot-panel']);
}
async function showWorkflowsList() {
  await navigateToDrawerTab('drawer-workflows-tab', ['workflows-list', 'workflows-flatlist', 'workflows-empty']);
}
async function showAgentsList() {
  await navigateToDrawerTab('drawer-activity-tab', ['agent-runs-list-screen']);
}
async function openHomeSignalDetail() {
  await showHome();
  await waitForAnyVisible(['home-feed-flatlist', 'home-feed-empty']);
  await element(by.id('home-feed-chip-signals')).tap();
  if (await isVisible('home-feed-empty', 5000)) return false;
  const signalId = ensureFirstActiveSignalId();
  if (signalId) {
    await scrollToVisible(`home-feed-signal-${signalId}`, 'home-feed-flatlist');
    await element(by.id(`home-feed-signal-${signalId}`)).tap();
  } else {
    await element(by.id('home-feed-flatlist')).tapAtPoint({ x: 540, y: 260 });
  }
  await waitForVisible('signal-detail');
  return true;
}
function registerHomeAndAccountScreens() {
  it('03_home_feed', async () => {
    await relaunchAuthenticatedSession();
    await showHome();
    await device.takeScreenshot('03_home_feed');
  });
  it('04_home_signal_detail', async () => {
    await relaunchAuthenticatedSession();
    if (!(await openHomeSignalDetail())) {
      await device.takeScreenshot('04_home_signal_detail_EMPTY');
      return;
    }
    await device.takeScreenshot('04_home_signal_detail');
    await device.pressBack();
  });
  it('05_crm_hub', async () => {
    await relaunchAuthenticatedSession();
    await ensureCrmSubmenuVisible();
    await device.takeScreenshot('05_crm_hub');
  });
  it('06_crm_accounts_list', async () => {
    await relaunchAuthenticatedSession();
    await showAccountsList();
    await device.takeScreenshot('06_crm_accounts_list');
  });
  it('07_crm_account_detail', async () => {
    await relaunchAuthenticatedSession();
    await showAccountsList();
    await waitForVisible('accounts-list-item-0');
    await element(by.id('accounts-list-item-0')).tap();
    await waitForVisible('account-detail-screen');
    await device.takeScreenshot('07_crm_account_detail');
    await device.pressBack();
  });
  it('08_crm_account_new', async () => {
    await relaunchAuthenticatedSession();
    await waitForVisible('create-account-fab');
    await element(by.id('create-account-fab')).tap();
    await waitForVisible('account-form-screen');
    await device.takeScreenshot('08_crm_account_new');
    await device.pressBack();
  });
}
function registerContactDealAndCaseScreens() {
  it('09_crm_contacts_list', async () => {
    await relaunchAuthenticatedSession();
    await showContactsList();
    await device.takeScreenshot('09_crm_contacts_list');
  });
  it('10_crm_contact_detail', async () => {
    await relaunchAuthenticatedSession();
    await showContactsList();
    await element(by.id('contacts-search')).replaceText(seeded.contact.email);
    await dismissKeyboard();
    await waitForVisible(`contact-item-${seeded.contact.id}`);
    await element(by.id(`contact-item-${seeded.contact.id}`)).tap();
    await waitForVisible('contact-detail-header');
    await device.takeScreenshot('10_crm_contact_detail');
    await device.pressBack();
  });
  it('11_crm_deals_list', async () => {
    await relaunchAuthenticatedSession();
    await showDealsList();
    await device.takeScreenshot('11_crm_deals_list');
  });
  it('12_crm_deal_detail', async () => {
    await relaunchAuthenticatedSession();
    await showDealsList();
    await scrollToVisible(`deal-item-${seeded.deal.id}`, 'deals-flatlist');
    await element(by.id(`deal-item-${seeded.deal.id}`)).tap();
    await waitForVisible('deal-detail-screen');
    await device.takeScreenshot('12_crm_deal_detail');
    await device.pressBack();
  });
  it('13_crm_deal_new', async () => {
    await relaunchAuthenticatedSession();
    await showDealsList();
    await waitForVisible('create-deal-fab');
    await element(by.id('create-deal-fab')).tap();
    await waitForVisible('deal-new-screen');
    await device.takeScreenshot('13_crm_deal_new');
    await device.pressBack();
  });
  it('14_crm_deal_edit', async () => {
    await relaunchAuthenticatedSession();
    await showDealsList();
    await scrollToVisible(`deal-item-${seeded.deal.id}`, 'deals-flatlist');
    await element(by.id(`deal-item-${seeded.deal.id}`)).tap();
    await waitForVisible('deal-detail-screen');
    await element(by.id('deal-edit-button')).tap();
    await waitForVisible('deal-edit-screen');
    await device.takeScreenshot('14_crm_deal_edit');
    await device.pressBack();
    await device.pressBack();
  });
  it('15_crm_cases_list', async () => {
    await relaunchAuthenticatedSession();
    await showCasesList();
    await device.takeScreenshot('15_crm_cases_list');
  });
  it('16_crm_case_detail', async () => {
    await relaunchAuthenticatedSession();
    await showCasesList();
    await element(by.id('cases-search')).replaceText(seeded.case.subject);
    await dismissKeyboard();
    await waitForVisible('cases-list-item-0');
    await element(by.id('cases-list-item-0')).tap();
    await waitForVisible('case-detail-screen');
    await device.takeScreenshot('16_crm_case_detail');
    await device.pressBack();
  });
  it('17_crm_case_new', async () => {
    await relaunchAuthenticatedSession();
    await showCasesList();
    await waitForVisible('create-case-fab');
    await element(by.id('create-case-fab')).tap();
    await waitForVisible('case-new-screen');
    await device.takeScreenshot('17_crm_case_new');
    await device.pressBack();
  });
  it('18_crm_case_edit', async () => {
    await relaunchAuthenticatedSession();
    await showCasesList();
    await element(by.id('cases-search')).replaceText(seeded.case.subject);
    await dismissKeyboard();
    await waitForVisible('cases-list-item-0');
    await element(by.id('cases-list-item-0')).tap();
    await waitForVisible('case-detail-screen');
    await element(by.id('case-edit-button')).tap();
    await waitForVisible('case-edit-screen');
    await device.takeScreenshot('18_crm_case_edit');
    await device.pressBack();
    await device.pressBack();
  });
}
function registerWorkflowAndAgentScreens() {
  it('19_copilot_panel', async () => {
    await relaunchAuthenticatedSession();
    await showCopilotPanel();
    await device.takeScreenshot('19_copilot_panel');
  });
  it('20_workflows_list', async () => {
    await relaunchAuthenticatedSession();
    await showWorkflowsList();
    await device.takeScreenshot('20_workflows_list');
  });
  it('21_workflow_new', async () => {
    await relaunchAuthenticatedSession();
    await showWorkflowsList();
    await element(by.id('workflows-new-btn')).tap();
    await waitForVisible('workflow-new-screen');
    await device.takeScreenshot('21_workflow_new');
    await device.pressBack();
  });
  it('22_workflow_detail', async () => {
    await relaunchAuthenticatedSession();
    await showWorkflowsList();
    await waitFor(element(by.id(`workflow-${seeded.workflows.activeId}`)))
      .toBeVisible()
      .whileElement(by.id('workflows-flatlist'))
      .scroll(200, 'down');
    await element(by.id(`workflow-${seeded.workflows.activeId}`)).tap();
    await waitForVisible('workflow-detail');
    await device.takeScreenshot('22_workflow_detail');
  });
  it('23_workflow_edit', async () => {
    await relaunchAuthenticatedSession();
    await showWorkflowsList();
    await waitFor(element(by.id(`workflow-${seeded.workflows.activeId}`)))
      .toBeVisible()
      .whileElement(by.id('workflows-flatlist'))
      .scroll(200, 'down');
    await element(by.id(`workflow-${seeded.workflows.activeId}`)).tap();
    await waitForVisible('workflow-detail');
    await element(by.id('workflow-edit-btn')).tap();
    await waitForVisible('workflow-edit-screen');
    await device.takeScreenshot('23_workflow_edit');
    await device.pressBack();
    await device.pressBack();
  });
  it('24_agents_list', async () => {
    await relaunchAuthenticatedSession();
    await showAgentsList();
    await device.takeScreenshot('24_agents_list');
  });
  it('25_agent_run_detail', async () => {
    await relaunchAuthenticatedSession();
    await showAgentsList();
    await waitForVisible(`agent-run-item-${seeded.agentRuns.rejectedId}`);
    await element(by.id(`agent-run-item-${seeded.agentRuns.rejectedId}`)).tap();
    await waitForVisible('agent-run-detail-screen');
    await device.takeScreenshot('25_agent_run_detail');
    await device.pressBack();
  });
  it('26_drawer_open', async () => {
    await relaunchAuthenticatedSession();
    await showAgentsList();
    await openDrawer();
    await waitForVisible('drawer-content');
    await device.takeScreenshot('26_drawer_open');
  });
}
describe('Visual Audit', () => {
  beforeAll(async () => {
    await device.launchApp({ newInstance: true });
    await device.disableSynchronization();
  });
  afterAll(async () => {
    await device.terminateApp();
  });
  it('01_auth_login', async () => {
    await waitForVisible('login-screen');
    await dismissKeyboard();
    await device.takeScreenshot('01_auth_login');
  });
  it('02_auth_register', async () => {
    await element(by.id('go-to-register-link')).tap();
    await waitForVisible('register-screen');
    await device.takeScreenshot('02_auth_register');
  });
  registerHomeAndAccountScreens();
  registerContactDealAndCaseScreens();
  registerWorkflowAndAgentScreens();
});
