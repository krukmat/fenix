// Traces: FR-001
// validateNewDealForm/validateNewCaseForm moved to src/utils/formValidation.ts
import { describe, expect, it } from '@jest/globals';
import { validateNewDealForm, validateNewCaseForm } from '../../src/utils/formValidation';

describe('Deals/Cases forms validation', () => {
  it('validates required fields for new deal', () => {
    const errors = validateNewDealForm({
      accountId: '',
      pipelineId: '',
      stageId: '',
      ownerId: '',
      title: '',
    });

    expect(errors).toEqual({
      accountId: true,
      pipelineId: true,
      stageId: true,
      ownerId: true,
      title: true,
    });
  });

  it('accepts complete required fields for new deal', () => {
    const errors = validateNewDealForm({
      accountId: 'acc-1',
      pipelineId: 'pipe-1',
      stageId: 'stage-1',
      ownerId: 'owner-1',
      title: 'Deal title',
    });

    expect(errors).toEqual({
      accountId: false,
      pipelineId: false,
      stageId: false,
      ownerId: false,
      title: false,
    });
  });

  it('validates required fields for new case', () => {
    const errors = validateNewCaseForm({
      ownerId: '',
      subject: '',
      description: '',
    });

    expect(errors).toEqual({
      ownerId: true,
      subject: true,
    });
  });

  it('accepts complete required fields for new case', () => {
    const errors = validateNewCaseForm({
      ownerId: 'owner-1',
      subject: 'Broken checkout',
      description: 'some context',
    });

    expect(errors).toEqual({
      ownerId: false,
      subject: false,
    });
  });
});
