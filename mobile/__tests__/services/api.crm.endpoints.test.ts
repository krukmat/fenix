import { beforeEach, describe, expect, it, jest } from '@jest/globals';
import { crmApi } from '../../src/services/api';
import { apiClient } from '../../src/services/api.client';

describe('crmApi endpoint coverage', () => {
  beforeEach(() => {
    jest.restoreAllMocks();
  });

  it('covers Account and Contact CRUD endpoints', async () => {
    const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValue({ data: { ok: true } } as never);
    const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValue({ data: { ok: true } } as never);
    const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValue({ data: { ok: true } } as never);
    const deleteSpy = jest.spyOn(apiClient, 'delete').mockResolvedValue({ data: undefined } as never);

    await crmApi.getAccount('acc-1');
    await crmApi.updateAccount('acc-1', { name: 'Acme' });
    await crmApi.deleteAccount('acc-1');
    await crmApi.getContactsByAccount('acc-1');
    await crmApi.createContact({ accountId: 'acc-1', firstName: 'Ada' });
    await crmApi.updateContact('contact-1', { title: 'CTO' });
    await crmApi.deleteContact('contact-1');

    expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/accounts/acc-1', undefined);
    expect(putSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/accounts/acc-1', { name: 'Acme' });
    expect(deleteSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/accounts/acc-1');
    expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/accounts/acc-1/contacts', undefined);
    expect(postSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/contacts', { accountId: 'acc-1', firstName: 'Ada' });
    expect(putSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/contacts/contact-1', { title: 'CTO' });
    expect(deleteSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/contacts/contact-1');
  });

  it('covers Lead, Deal, and Case mutation endpoints', async () => {
    const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValue({ data: { ok: true } } as never);
    const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValue({ data: { ok: true } } as never);
    const deleteSpy = jest.spyOn(apiClient, 'delete').mockResolvedValue({ data: undefined } as never);

    await crmApi.createLead({ ownerId: 'user-1', source: 'web' });
    await crmApi.updateLead('lead-1', { status: 'qualified' });
    await crmApi.deleteLead('lead-1');
    await crmApi.deleteDeal('deal-1');
    await crmApi.deleteCase('case-1');

    expect(postSpy).toHaveBeenCalledWith('/bff/api/v1/leads', { ownerId: 'user-1', source: 'web' });
    expect(putSpy).toHaveBeenCalledWith('/bff/api/v1/leads/lead-1', { status: 'qualified' });
    expect(deleteSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/leads/lead-1');
    expect(deleteSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/deals/deal-1');
    expect(deleteSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/cases/case-1');
  });

  it('covers Pipeline and Stage endpoints', async () => {
    const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValue({ data: { ok: true } } as never);
    const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValue({ data: { ok: true } } as never);
    const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValue({ data: { ok: true } } as never);
    const deleteSpy = jest.spyOn(apiClient, 'delete').mockResolvedValue({ data: undefined } as never);

    await crmApi.getPipelines('ws-1', { page: 2, limit: 10 });
    await crmApi.getPipeline('pipe-1');
    await crmApi.createPipeline({ name: 'Sales', entityType: 'deal' });
    await crmApi.updatePipeline('pipe-1', { isDefault: true });
    await crmApi.deletePipeline('pipe-1');
    await crmApi.getPipelineStages('pipe-1');
    await crmApi.createPipelineStage('pipe-1', { name: 'Qualified', position: 2 });
    await crmApi.updatePipelineStage('stage-1', { probability: 50 });
    await crmApi.deletePipelineStage('stage-1');

    expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/pipelines', { params: { workspace_id: 'ws-1', page: 2, limit: 10 } });
    expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/pipelines/pipe-1', undefined);
    expect(postSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/pipelines', { name: 'Sales', entityType: 'deal' });
    expect(putSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/pipelines/pipe-1', { isDefault: true });
    expect(deleteSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/pipelines/pipe-1');
    expect(getSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/pipelines/pipe-1/stages', undefined);
    expect(postSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/pipelines/pipe-1/stages', { name: 'Qualified', position: 2 });
    expect(putSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/pipelines/stages/stage-1', { probability: 50 });
    expect(deleteSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/pipelines/stages/stage-1');
  });

  it('covers Activity, Note, Attachment, and Timeline endpoints', async () => {
    const getSpy = jest.spyOn(apiClient, 'get').mockResolvedValue({ data: { ok: true } } as never);
    const postSpy = jest.spyOn(apiClient, 'post').mockResolvedValue({ data: { ok: true } } as never);
    const putSpy = jest.spyOn(apiClient, 'put').mockResolvedValue({ data: { ok: true } } as never);
    const deleteSpy = jest.spyOn(apiClient, 'delete').mockResolvedValue({ data: undefined } as never);

    await crmApi.getActivities('ws-1', { limit: 5, offset: 10 });
    await crmApi.getActivity('act-1');
    await crmApi.createActivity({ entityType: 'case', entityId: 'case-1', type: 'call', subject: 'Call', description: 'Intro' });
    await crmApi.updateActivity('act-1', { status: 'completed', body: 'Done' });
    await crmApi.deleteActivity('act-1');
    await crmApi.getNotes('ws-1');
    await crmApi.getNote('note-1');
    await crmApi.createNote({ entityType: 'account', entityId: 'acc-1', content: 'memo' });
    await crmApi.updateNote('note-1', { content: 'updated' });
    await crmApi.deleteNote('note-1');
    await crmApi.getAttachments('ws-1');
    await crmApi.getAttachment('att-1');
    await crmApi.createAttachment({ entityType: 'case', entityId: 'case-1', fileName: 'log.txt' });
    await crmApi.deleteAttachment('att-1');
    await crmApi.getTimeline('ws-1', { limit: 2, offset: 4 });
    await crmApi.getTimelineByEntity('case', 'case-1');

    expect(getSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/activities', { params: { workspace_id: 'ws-1', limit: 5, offset: 10 } });
    expect(getSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/activities/act-1', undefined);
    expect(postSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/activities', { entityType: 'case', entityId: 'case-1', subject: 'Call', activityType: 'call', body: 'Intro' });
    expect(putSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/activities/act-1', { status: 'completed', body: 'Done' });
    expect(deleteSpy).toHaveBeenNthCalledWith(1, '/bff/api/v1/activities/act-1');
    expect(getSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/notes', { params: { workspace_id: 'ws-1', limit: 50, offset: 0 } });
    expect(getSpy).toHaveBeenNthCalledWith(4, '/bff/api/v1/notes/note-1', undefined);
    expect(postSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/notes', { entityType: 'account', entityId: 'acc-1', content: 'memo' });
    expect(putSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/notes/note-1', { content: 'updated' });
    expect(deleteSpy).toHaveBeenNthCalledWith(2, '/bff/api/v1/notes/note-1');
    expect(getSpy).toHaveBeenNthCalledWith(5, '/bff/api/v1/attachments', { params: { workspace_id: 'ws-1', limit: 50, offset: 0 } });
    expect(getSpy).toHaveBeenNthCalledWith(6, '/bff/api/v1/attachments/att-1', undefined);
    expect(postSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/attachments', { entityType: 'case', entityId: 'case-1', filename: 'log.txt' });
    expect(deleteSpy).toHaveBeenNthCalledWith(3, '/bff/api/v1/attachments/att-1');
    expect(getSpy).toHaveBeenNthCalledWith(7, '/bff/api/v1/timeline', { params: { workspace_id: 'ws-1', limit: 2, offset: 4 } });
    expect(getSpy).toHaveBeenNthCalledWith(8, '/bff/api/v1/timeline/case/case-1', undefined);
  });
});
