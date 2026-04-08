import {
  assertCopilotPanelVisible,
  openAccountDetailForUCS1,
  openCopilotFromAccountDetail,
  openCopilotFromDealDetail,
  openDealDetailForUCS1,
  startUCS1Session,
  finishUCS1Session,
} from '../helpers/uc_s1.helper';

describe.skip('BDD UC-S1 Sales Copilot', () => {
  beforeAll(async () => {
    await startUCS1Session();
  });

  afterAll(async () => {
    await finishUCS1Session();
  });

  it('Launch Sales Copilot from account detail with grounded context', async () => {
    await openAccountDetailForUCS1();
    await openCopilotFromAccountDetail();
    await assertCopilotPanelVisible();
  });

  it('Launch Sales Copilot from deal detail with grounded context', async () => {
    await openDealDetailForUCS1();
    await openCopilotFromDealDetail();
    await assertCopilotPanelVisible();
  });
});
