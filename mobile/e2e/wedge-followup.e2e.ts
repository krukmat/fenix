import { by, device, element, expect as detoxExpect, waitFor } from 'detox';
import { loginAsTestUser } from './helpers/auth.helper';
import { waitForElement } from './helpers/navigation.helper';
import { ensureWedgeSeed } from './helpers/seed.helper';

async function openWedgePath(path: string) {
  const normalizedPath = path.replace(/^\/+/, '');
  await device.openURL({ url: `fenixcrm:///${normalizedPath}` });
}

describe('Wedge follow-up functional smoke', () => {
  const seeded = ensureWedgeSeed();

  beforeAll(async () => {
    await loginAsTestUser();
  });

  afterAll(async () => {
    await device.terminateApp();
  });

  it('lands on Inbox and resolves the seeded support approval inline', async () => {
    await openWedgePath('inbox');
    await waitFor(element(by.id(`inbox-approval-${seeded.inbox.approvalId}`)))
      .toBeVisible()
      .withTimeout(30000);

    await element(by.id(`inbox-approval-${seeded.inbox.approvalId}-reject`)).tap();
    await waitFor(element(by.id(`inbox-approval-${seeded.inbox.approvalId}-reject-dialog`)))
      .toBeVisible()
      .withTimeout(5000);
    await element(by.id(`inbox-approval-${seeded.inbox.approvalId}-reject-reason-input`))
      .replaceText('Detox follow-up rejection');
    await element(by.id(`inbox-approval-${seeded.inbox.approvalId}-reject-submit`)).tap();

    await waitFor(element(by.id(`inbox-approval-${seeded.inbox.approvalId}`)))
      .not.toBeVisible()
      .withTimeout(15000);
  });

  it('routes a handed off support run from Activity to the support case detail', async () => {
    await openWedgePath(`activity/${seeded.agentRuns.handoffId}`);
    await waitForElement('activity-run-detail-screen', 15000);
    await waitForElement('activity-detail-handoff-accept', 15000);
    await element(by.id('activity-detail-handoff-accept')).tap();

    await waitForElement('support-case-detail-screen', 15000);
    await detoxExpect(element(by.id('support-trigger-agent-button'))).toBeVisible();
  });

  it('shows an abstained Sales Brief for the seeded account context', async () => {
    await openWedgePath(`sales/${seeded.account.id}/brief?entity_type=account&entity_id=${seeded.account.id}`);
    await waitForElement('sales-brief-screen', 30000);
    await detoxExpect(element(by.id('sales-brief-outcome'))).toBeVisible();
    await detoxExpect(element(by.text('abstained'))).toBeVisible();
    await detoxExpect(element(by.id('sales-brief-abstention-reason'))).toBeVisible();
    await detoxExpect(element(by.id('sales-brief-evidence-pack'))).toBeVisible();
  });

  it('shows a completed Sales Brief for the seeded deal context', async () => {
    await openWedgePath(`sales/deal-${seeded.deal.id}/brief?entity_type=deal&entity_id=${seeded.deal.id}`);
    await waitForElement('sales-brief-screen', 30000);
    await detoxExpect(element(by.id('sales-brief-outcome'))).toBeVisible();
    await detoxExpect(element(by.text('completed'))).toBeVisible();
    await detoxExpect(element(by.id('sales-brief-next-best-actions'))).toBeVisible();
    await detoxExpect(element(by.id('sales-brief-evidence-pack'))).toBeVisible();
  });

  it('shows the denied activity detail and governance trace surfaces', async () => {
    await openWedgePath(`activity/${seeded.agentRuns.deniedByPolicyId}`);
    await waitForElement('activity-run-detail-screen', 15000);
    await detoxExpect(element(by.id('activity-detail-rejection-reason'))).toBeVisible();

    await openWedgePath('governance');
    await waitForElement('governance-screen', 15000);
    await detoxExpect(element(by.id('governance-recent-usage'))).toBeVisible();
  });
});
