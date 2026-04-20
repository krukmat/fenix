// Task 4.2 — FR-300: CRM and Auth API methods
import { apiClient } from './api.client';
import { crmEndpointApi } from './api.crm.endpoints';

const CRM_BASE = '/bff/api/v1';

export const authApi = {
  login: async (email: string, password: string) => {
    const response = await apiClient.post('/bff/auth/login', { email, password });
    return response.data;
  },

  register: async (displayName: string, email: string, password: string, workspaceName: string) => {
    const response = await apiClient.post('/bff/auth/register', {
      displayName,
      email,
      password,
      workspaceName,
    });
    return response.data;
  },
};

async function getOrNull(path: string, params?: Record<string, string | number | undefined>) {
  try {
    const response = await apiClient.get(path, params ? { params } : undefined);
    return response.data;
  } catch {
    return null;
  }
}

function extractId(obj: Record<string, unknown> | null, camel: string, snake: string): string | undefined {
  return (obj?.[camel] as string | undefined) ?? (obj?.[snake] as string | undefined);
}

function extractSignalCount(obj: Record<string, unknown> | null): number {
  return typeof obj?.active_signal_count === 'number' ? (obj.active_signal_count as number) : 0;
}

export const crmApi = {
  ...crmEndpointApi,

  getAccounts: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get(`${CRM_BASE}/accounts`, {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  createAccount: async (data: {
    name: string;
    industry?: string;
    website?: string;
    phone?: string;
    email?: string;
    description?: string;
    ownerId?: string;
  }) => {
    const response = await apiClient.post(`${CRM_BASE}/accounts`, data);
    return response.data;
  },

  getContacts: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get(`${CRM_BASE}/contacts`, {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  getDeals: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get(`${CRM_BASE}/deals`, {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  getLeads: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const page = pagination?.page ?? 1;
    const limit = pagination?.limit ?? 50;
    const offset = (page - 1) * limit;
    const response = await apiClient.get(`${CRM_BASE}/leads`, {
      params: { workspace_id: workspaceId, limit, offset },
    });
    return response.data;
  },

  getCases: async (workspaceId: string, pagination?: { page?: number; limit?: number }) => {
    const response = await apiClient.get(`${CRM_BASE}/cases`, {
      params: { workspace_id: workspaceId, page: pagination?.page ?? 1, limit: pagination?.limit ?? 50 },
    });
    return response.data;
  },

  createDeal: async (data: {
    accountId: string;
    contactId?: string;
    pipelineId: string;
    stageId: string;
    ownerId: string;
    title: string;
    amount?: number;
    currency?: string;
    expectedClose?: string;
    status?: string;
    metadata?: string;
  }) => {
    const response = await apiClient.post(`${CRM_BASE}/deals`, data);
    return response.data;
  },

  updateDeal: async (id: string, data: {
    accountId?: string;
    contactId?: string;
    pipelineId?: string;
    stageId?: string;
    ownerId?: string;
    title?: string;
    amount?: number;
    currency?: string;
    expectedClose?: string;
    status?: string;
    metadata?: string;
  }) => {
    const response = await apiClient.put(`${CRM_BASE}/deals/${id}`, data);
    return response.data;
  },

  createCase: async (data: {
    accountId?: string;
    contactId?: string;
    pipelineId?: string;
    stageId?: string;
    ownerId: string;
    subject: string;
    description?: string;
    priority?: string;
    status?: string;
    channel?: string;
    slaConfig?: string;
    slaDeadline?: string;
    metadata?: string;
  }) => {
    const response = await apiClient.post(`${CRM_BASE}/cases`, data);
    return response.data;
  },

  updateCase: async (id: string, data: {
    accountId?: string;
    contactId?: string;
    pipelineId?: string;
    stageId?: string;
    ownerId?: string;
    subject?: string;
    description?: string;
    priority?: string;
    status?: string;
    channel?: string;
    slaConfig?: string;
    slaDeadline?: string;
    metadata?: string;
  }) => {
    const response = await apiClient.put(`${CRM_BASE}/cases/${id}`, data);
    return response.data;
  },

  getAccountFull: async (id: string) => {
    const [account, contacts, deals, timeline] = await Promise.all([
      apiClient.get(`${CRM_BASE}/accounts/${id}`).then((response) => response.data),
      getOrNull(`${CRM_BASE}/accounts/${id}/contacts`),
      getOrNull(`${CRM_BASE}/deals`, { account_id: id, limit: 50 }),
      getOrNull(`${CRM_BASE}/timeline/account/${id}`),
    ]);
    return {
      account,
      contacts,
      deals,
      timeline,
      active_signal_count:
        typeof account?.active_signal_count === 'number' ? account.active_signal_count : 0,
    };
  },

  getDealFull: async (id: string) => {
    const deal = await apiClient.get(`${CRM_BASE}/deals/${id}`).then((response) => response.data);
    const d = deal as Record<string, unknown> | null;
    const accountId = extractId(d, 'accountId', 'account_id');
    const contactId = extractId(d, 'contactId', 'contact_id');
    const [account, contact, activities] = await Promise.all([
      accountId ? getOrNull(`${CRM_BASE}/accounts/${accountId}`) : Promise.resolve(null),
      contactId ? getOrNull(`${CRM_BASE}/contacts/${contactId}`) : Promise.resolve(null),
      getOrNull(`${CRM_BASE}/activities`, { deal_id: id, limit: 50 }),
    ]);
    return { deal, account, contact, activities, active_signal_count: extractSignalCount(d) };
  },

  getCaseFull: async (id: string) => {
    const caseData = await apiClient.get(`${CRM_BASE}/cases/${id}`).then((response) => response.data);
    const c = caseData as Record<string, unknown> | null;
    const accountId = extractId(c, 'accountId', 'account_id');
    const contactId = extractId(c, 'contactId', 'contact_id');
    const handoffStatus = extractId(c, 'handoffStatus', 'handoff_status');
    const [account, contact, activities] = await Promise.all([
      accountId ? getOrNull(`${CRM_BASE}/accounts/${accountId}`) : Promise.resolve(null),
      contactId ? getOrNull(`${CRM_BASE}/contacts/${contactId}`) : Promise.resolve(null),
      getOrNull(`${CRM_BASE}/activities`, { case_id: id, limit: 50 }),
    ]);
    return {
      case: caseData,
      account,
      contact,
      activities,
      handoff: handoffStatus ? { status: handoffStatus } : null,
      active_signal_count: extractSignalCount(c),
    };
  },

  getContact: async (id: string) => {
    const response = await apiClient.get(`${CRM_BASE}/contacts/${id}`);
    return response.data;
  },

  getLead: async (id: string) => {
    const response = await apiClient.get(`${CRM_BASE}/leads/${id}`);
    return response.data;
  },
};
