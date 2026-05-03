// Task Mobile P1.4 — T2: Workflow API methods
import { apiClient } from './api.client';
import type { Workflow, WorkflowStatus } from './api';

export type WorkflowGraphNode = {
  id: string;
  kind: string;
  label: string;
};

export type WorkflowGraphEdge = {
  id?: string;
  from: string;
  to: string;
  connection_type?: string;
};

export type WorkflowGraph = {
  workflow_id?: string;
  workflow_name?: string;
  conformance: {
    profile: string;
    details: { code?: string; description?: string; message?: string }[];
  };
  nodes: WorkflowGraphNode[];
  edges: WorkflowGraphEdge[];
};

export const workflowApi = {
  list: async (params?: { status?: WorkflowStatus; page?: number; limit?: number }) => {
    const response = await apiClient.get('/bff/api/v1/workflows', { params });
    return response.data as { data: Workflow[]; total: number };
  },

  get: async (id: string) => {
    const response = await apiClient.get(`/bff/api/v1/workflows/${id}`);
    return response.data as Workflow;
  },

  create: async (body: { name: string; description?: string; dsl_source: string }) => {
    const response = await apiClient.post('/bff/api/v1/workflows', body);
    return response.data as Workflow;
  },

  update: async (id: string, body: { description?: string; dsl_source: string }) => {
    const response = await apiClient.put(`/bff/api/v1/workflows/${id}`, body);
    return response.data as Workflow;
  },

  activate: async (id: string) => {
    const response = await apiClient.put(`/bff/api/v1/workflows/${id}/activate`);
    return response.data as Workflow;
  },

  execute: async (id: string) => {
    const response = await apiClient.post(`/bff/api/v1/workflows/${id}/execute`);
    return response.data as { run_id: string };
  },

  listVersions: async (id: string) => {
    const response = await apiClient.get(`/bff/api/v1/workflows/${id}/versions`);
    const payload = response.data as { data?: Workflow[] } | Workflow[];
    return Array.isArray(payload) ? payload : Array.isArray(payload.data) ? payload.data : [];
  },

  newVersion: async (id: string) => {
    const response = await apiClient.post(`/bff/api/v1/workflows/${id}/new-version`);
    return response.data as Workflow;
  },

  rollback: async (id: string) => {
    const response = await apiClient.put(`/bff/api/v1/workflows/${id}/rollback`);
    return response.data as Workflow;
  },

  verifyWorkflow: async (id: string) => {
    const response = await apiClient.post(`/bff/api/v1/workflows/${id}/verify`);
    return response.data as { valid: boolean; errors?: string[] };
  },

  getGraph: async (id: string): Promise<WorkflowGraph> => {
    const response = await apiClient.get(`/bff/api/v1/workflows/${id}/graph`, { params: { format: 'visual' } });
    const payload = response.data as { data?: WorkflowGraph } | WorkflowGraph;
    return 'data' in payload && payload.data ? payload.data : payload as WorkflowGraph;
  },
};
