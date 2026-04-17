// Task 4.2 — FR-200: Copilot API methods
import { apiClient } from './api.client';

export const copilotApi = {
  buildChatUrl: (): string => `${process.env.EXPO_PUBLIC_BFF_URL || 'http://10.0.2.2:3000'}/bff/copilot/chat`,

  suggestActions: async (entityType: string, entityId: string) => {
    const response = await apiClient.post('/bff/api/v1/copilot/suggest-actions', {
      entity_type: entityType,
      entity_id: entityId,
    });
    return response.data;
  },

  summarize: async (entityType: string, entityId: string) => {
    const response = await apiClient.post('/bff/api/v1/copilot/summarize', {
      entity_type: entityType,
      entity_id: entityId,
    });
    return response.data;
  },
};
