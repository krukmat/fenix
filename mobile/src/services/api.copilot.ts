// Task 4.2 — FR-200: Copilot API methods
import { apiClient, BFF_URL } from './api.client';

export const copilotApi = {
  buildChatUrl: (): string => `${BFF_URL}/bff/copilot/chat`,

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
