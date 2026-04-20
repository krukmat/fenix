import { describe, expect, it } from '@jest/globals';
import {
  normalizeCRMAccount,
  normalizeCRMAttachment,
  normalizeCRMCase,
  normalizeCRMDeal,
  normalizeCRMLead,
  normalizeCRMList,
  normalizeCRMNote,
  normalizeCRMPipeline,
  normalizeCRMPipelineStage,
  normalizeCRMTimelineEvent,
} from '../../src/services/api';

describe('CRM API normalizers', () => {
  it('normalizes paginated account payloads from backend snake_case', () => {
    const result = normalizeCRMList(
      {
        data: [
          {
            id: 'acc-1',
            workspace_id: 'ws-1',
            owner_id: 'user-1',
            name: 'Acme',
            active_signal_count: 3,
            created_at: '2026-04-19T10:00:00Z',
          },
        ],
        meta: { total: 9, limit: 1, offset: 2 },
      },
      normalizeCRMAccount,
    );

    expect(result).toEqual({
      data: [
        expect.objectContaining({
          id: 'acc-1',
          workspaceId: 'ws-1',
          ownerId: 'user-1',
          name: 'Acme',
          activeSignalCount: 3,
          createdAt: '2026-04-19T10:00:00Z',
        }),
      ],
      meta: { total: 9, limit: 1, offset: 2 },
    });
  });

  it('normalizes camelCase and snake_case relation fields consistently', () => {
    expect(
      normalizeCRMDeal({
        id: 'deal-1',
        account_id: 'acc-1',
        contactId: 'contact-1',
        pipeline_id: 'pipe-1',
        stageId: 'stage-1',
        title: 'Expansion',
        expected_close: '2026-05-01',
        metadata: '{"source":"seed"}',
        account_name: 'Acme',
      }),
    ).toEqual(
      expect.objectContaining({
        accountId: 'acc-1',
        contactId: 'contact-1',
        pipelineId: 'pipe-1',
        stageId: 'stage-1',
        expectedClose: '2026-05-01',
        metadata: { source: 'seed' },
        accountName: 'Acme',
      }),
    );
  });

  it('normalizes standalone lead and case defaults without throwing', () => {
    expect(normalizeCRMLead({ id: 'lead-1', metadata: '{bad json' })).toEqual(
      expect.objectContaining({
        id: 'lead-1',
        metadata: {},
      }),
    );

    expect(normalizeCRMCase({ id: 'case-1', activeSignalCount: 2 })).toEqual(
      expect.objectContaining({
        id: 'case-1',
        subject: 'No Subject',
        activeSignalCount: 2,
      }),
    );
  });

  it('normalizes pipeline and stage records used by deal forms', () => {
    expect(normalizeCRMPipeline({ id: 'pipe-1', entity_type: 'deal', is_default: 1 })).toEqual(
      expect.objectContaining({
        id: 'pipe-1',
        name: 'Unnamed Pipeline',
        entityType: 'deal',
        isDefault: true,
      }),
    );

    expect(normalizeCRMPipelineStage({ id: 'stage-1', pipeline_id: 'pipe-1', position: 2 })).toEqual(
      expect.objectContaining({
        id: 'stage-1',
        pipelineId: 'pipe-1',
        name: 'Unnamed Stage',
        position: 2,
      }),
    );
  });

  it('normalizes polymorphic note, attachment, and timeline records', () => {
    expect(normalizeCRMNote({ id: 'note-1', entity_type: 'case', entity_id: 'case-1', is_internal: 1 })).toEqual(
      expect.objectContaining({
        id: 'note-1',
        entityType: 'case',
        entityId: 'case-1',
        isInternal: true,
      }),
    );

    expect(normalizeCRMAttachment({ id: 'att-1', entityType: 'case', entityId: 'case-1', size_bytes: 128 })).toEqual(
      expect.objectContaining({
        id: 'att-1',
        entityType: 'case',
        entityId: 'case-1',
        sizeBytes: 128,
      }),
    );

    expect(normalizeCRMTimelineEvent({ id: 'tl-1', entity_type: 'case', entity_id: 'case-1', event_type: 'created', created_at: '2026-04-19T11:00:00Z' })).toEqual(
      expect.objectContaining({
        id: 'tl-1',
        entityType: 'case',
        entityId: 'case-1',
        eventType: 'created',
        title: 'created',
        timestamp: '2026-04-19T11:00:00Z',
      }),
    );
  });
});
