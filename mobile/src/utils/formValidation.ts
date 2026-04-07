// Form validation utilities extracted from legacy create screens
// Tests: __tests__/components/dealsCasesForms.test.ts

export type DealCreateForm = {
  accountId: string;
  pipelineId: string;
  stageId: string;
  ownerId: string;
  title: string;
};

export type CaseCreateForm = {
  ownerId: string;
  subject: string;
  description: string;
};

export function validateNewDealForm(form: DealCreateForm) {
  return {
    accountId: !form.accountId.trim(),
    pipelineId: !form.pipelineId.trim(),
    stageId: !form.stageId.trim(),
    ownerId: !form.ownerId.trim(),
    title: !form.title.trim(),
  };
}

export function validateNewCaseForm(form: CaseCreateForm) {
  return {
    ownerId: !form.ownerId.trim(),
    subject: !form.subject.trim(),
  };
}
