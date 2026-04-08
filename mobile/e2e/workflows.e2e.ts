import { by, device, element, expect as detoxExpect, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';
import { ensureMobileP2Seed } from './helpers/seed.helper';

describe.skip('Workflows Mobile P2 smoke', () => {
  const seeded = ensureMobileP2Seed();
  const suffix = Date.now().toString().slice(-6);

  beforeAll(async () => {
    await loginAsTestUser();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  it('creates and edits a draft workflow', async () => {
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-workflows-tab')).tap();
    await detoxExpect(element(by.id('workflows-list'))).toBeVisible(5000);

    await element(by.id('workflows-new-btn')).tap();
    await detoxExpect(element(by.id('workflow-new-screen'))).toBeVisible(5000);

    await element(by.id('workflow-form-name-input')).replaceText(`e2e_mobile_p2_${suffix}`);
    await element(by.id('workflow-form-description-input')).replaceText('Created from Detox smoke');
    await element(by.id('workflow-form-dsl-input')).replaceText('WORKFLOW e2e_mobile_p2 ON case.created SET case.status = "open"');
    await element(by.id('workflow-new-submit')).tap();

    await waitFor(element(by.id('workflow-detail')))
      .toBeVisible()
      .withTimeout(10000);
    await detoxExpect(element(by.id('workflow-edit-btn'))).toBeVisible();

    await element(by.id('workflow-edit-btn')).tap();
    await detoxExpect(element(by.id('workflow-edit-screen'))).toBeVisible(5000);
    await element(by.id('workflow-form-description-input')).replaceText('Edited from Detox smoke');
    await element(by.id('workflow-form-dsl-input')).replaceText('WORKFLOW e2e_mobile_p2 ON case.updated SET case.status = "resolved"');
    await element(by.id('workflow-edit-submit')).tap();

    await waitFor(element(by.id('workflow-detail')))
      .toBeVisible()
      .withTimeout(10000);
  });

  it('shows version history and creates a new version from an active workflow', async () => {
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-workflows-tab')).tap();
    await waitFor(element(by.id(`workflow-${seeded.workflows.activeId}`)))
      .toBeVisible()
      .withTimeout(10000);

    await element(by.id(`workflow-${seeded.workflows.activeId}`)).tap();
    await waitFor(element(by.id('workflow-detail')))
      .toBeVisible()
      .withTimeout(10000);
    await detoxExpect(element(by.id('workflow-version-history'))).toBeVisible();
    await detoxExpect(element(by.id('workflow-new-version-btn'))).toBeVisible();

    await element(by.id('workflow-new-version-btn')).tap();

    await waitFor(element(by.id('workflow-detail')))
      .toBeVisible()
      .withTimeout(10000);
    await detoxExpect(element(by.id('workflow-edit-btn'))).toBeVisible();
  });

  it('rolls back an archived workflow fixture', async () => {
    await element(by.id('drawer-open-button')).tap();
    await element(by.id('drawer-workflows-tab')).tap();
    await waitFor(element(by.id(`workflow-${seeded.workflows.archivedId}`)))
      .toBeVisible()
      .withTimeout(10000);

    await element(by.id(`workflow-${seeded.workflows.archivedId}`)).tap();
    await waitFor(element(by.id('workflow-detail')))
      .toBeVisible()
      .withTimeout(10000);
    await detoxExpect(element(by.id('workflow-version-history'))).toBeVisible();
    await detoxExpect(element(by.id(`workflow-rollback-btn-${seeded.workflows.archivedId}`))).toBeVisible();

    await element(by.id(`workflow-rollback-btn-${seeded.workflows.archivedId}`)).tap();

    await waitFor(element(by.id('workflow-detail-status')))
      .toBeVisible()
      .withTimeout(10000);
  });
});
