// BFF-ADMIN-10 / BFF-ADMIN-11 / BFF-ADMIN-12: workflows list, detail, activation
import { Router, Request, Response, NextFunction } from 'express';
import { createGoClient } from '../services/goClient';
import { adminLayout } from './adminLayout';
import { upstreamStatus, upstreamMessage } from './adminAuth';
import {
  escHtml,
  buildDetailBody,
  buildListBody,
  buildNewDraftBody,
  draftFormState,
  draftCreatePayload,
  type WorkflowDetail,
  type WorkflowRow,
  type WorkflowCreateResponse,
} from './adminWorkflowsFragments';

const ADMIN_ROOT = '/bff/admin';

function extractListParams(q: Record<string, unknown>): { statusFilter: string; nameFilter: string; params: Record<string, string> } {
  const statusFilter = typeof q['status'] === 'string' ? q['status'] : '';
  const nameFilter   = typeof q['name']   === 'string' ? q['name']   : '';
  const params: Record<string, string> = {};
  if (statusFilter) params['status'] = statusFilter;
  if (nameFilter)   params['name']   = nameFilter;
  return { statusFilter, nameFilter, params };
}

const router = Router();

router.get('/new', (_req: Request, res: Response): void => {
  res.type('html').status(200).send(adminLayout('Create workflow draft', buildNewDraftBody(draftFormState())));
});

router.get('/', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { statusFilter, nameFilter, params } = extractListParams(req.query);

  try {
    const client = createGoClient(token);
    // BFF-ADMIN-Task6: Go returns envelope { data: [...] } — extract the array
    const { data: body } = await client.get<{ data: WorkflowRow[] }>('/api/v1/workflows', { params });
    res.type('html').status(200).send(adminLayout('Workflows', buildListBody(body.data ?? [], statusFilter, nameFilter)));
  } catch (err: unknown) {
    const status = (err as { response?: { status?: number } }).response?.status;
    if (status === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

router.post('/', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const form = draftFormState(req.body as Record<string, unknown>);
  const client = createGoClient(token);

  try {
    const { data: resp } = await client.post<{ data: WorkflowCreateResponse }>('/api/v1/workflows', draftCreatePayload(form));
    res.redirect(`/bff/builder?workflowId=${encodeURIComponent(resp.data.id)}`);
  } catch (err: unknown) {
    const status = upstreamStatus(err);
    if (status === 401) { res.redirect(ADMIN_ROOT); return; }
    if (status !== undefined && status >= 400 && status < 500) {
      res.type('html').status(200).send(adminLayout('Create workflow draft', buildNewDraftBody(form, upstreamMessage(err))));
      return;
    }
    next(err);
  }
});

// BFF-ADMIN-11: workflow detail page (read-only)
router.get('/:id', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;

  try {
    const client = createGoClient(token);
    // BFF-ADMIN-Task6: Go returns envelope { data: {...} } — extract the workflow object
    const { data: resp } = await client.get<{ data: WorkflowDetail }>(`/api/v1/workflows/${id}`);
    res.type('html').status(200).send(adminLayout(`Workflow: ${resp.data.name}`, buildDetailBody(resp.data)));
  } catch (err: unknown) {
    if (upstreamStatus(err) === 401) { res.redirect(ADMIN_ROOT); return; }
    next(err);
  }
});

// BFF-ADMIN-12: activate form submission — POST-Redirect-GET on success, re-render with error on 4xx
router.post('/:id/activate', async (req: Request, res: Response, next: NextFunction): Promise<void> => {
  const token = res.locals['adminToken'] as string | undefined;
  const { id } = req.params;
  const client = createGoClient(token);

  try {
    await client.put(`/api/v1/workflows/${id}/activate`);
    res.redirect(`/bff/admin/workflows/${id}`);
  } catch (err: unknown) {
    const status = upstreamStatus(err);
    if (status === 401) { res.redirect(ADMIN_ROOT); return; }
    if (status !== undefined && status >= 400 && status < 500) {
      try {
        const { data: wfResp } = await client.get<{ data: WorkflowDetail }>(`/api/v1/workflows/${id}`);
        const wf = wfResp.data;
        const errorBanner = `
          <div style="margin-bottom:16px;padding:12px 16px;border:1px solid #fca5a5;background:#fef2f2;border-radius:6px;color:#991b1b;font-size:13px">
            <strong>Activation failed:</strong> ${escHtml(upstreamMessage(err))}
          </div>`;
        res.type('html').status(200).send(adminLayout(`Workflow: ${wf.name}`, errorBanner + buildDetailBody(wf)));
        return;
      } catch {
        // if the re-fetch also fails, fall through to next(err)
      }
    }
    next(err);
  }
});

export default router;
