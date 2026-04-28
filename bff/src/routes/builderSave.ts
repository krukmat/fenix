import { Router, Request, Response } from 'express';
import { createGoClient } from '../services/goClient';
import { upstreamMessage } from './adminAuth';

type BffRequest = Request & { bearerToken?: string };

const router = Router({ mergeParams: true });

router.post('/:id', async (req: BffRequest, res: Response): Promise<void> => {
  const raw = req.params['id'];
  const id = Array.isArray(raw) ? raw[0] : raw;
  const client = createGoClient(req.bearerToken);

  try {
    const upstream = await client.put(
      `/api/v1/workflows/${id}`,
      {
        dsl_source: formValue(req.body, 'source'),
        spec_source: formValue(req.body, 'spec_source'),
      },
      { validateStatus: (status) => status < 500 },
    );

    if (upstream.status === 200) {
      res.type('html').status(200).send(renderSaveResult('Draft saved', []));
      return;
    }

    const message = buildFailureMessage(upstream.status, upstreamMessage({ response: upstream }));
    res.type('html').status(200).send(renderSaveResult('Save failed', [message]));
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Text save request failed';
    res.type('html').status(502).send(renderSaveResult('Save unavailable', [message]));
  }
});

function buildFailureMessage(status: number | undefined, message: string): string {
  if (status === 401) return `Unauthorized: ${message}. Update the bearer token and retry.`;
  if (status === 403) return `Forbidden: ${message}.`;
  if (status === 409) return `Conflict: ${message}.`;
  if (status === 422) return `Validation: ${message}.`;
  return message;
}

function renderSaveResult(statusLabel: string, diagnostics: string[]): string {
  return [
    `<span class="preview-status" id="builder-save-status">${escapeHtml(statusLabel)}</span>`,
    renderDiagnostics(diagnostics),
  ].join('');
}

function renderDiagnostics(items: string[]): string {
  const content = items.length === 0
    ? '<li class="diagnostic-empty">No validation diagnostics for current draft.</li>'
    : items.map((item) => `<li><strong>save</strong>: ${escapeHtml(item)}</li>`).join('');
  return `<ul class="diagnostics-list" id="builder-diagnostics" aria-live="polite" hx-swap-oob="true">${content}</ul>`;
}

function formValue(body: unknown, key: string): string {
  if (typeof body !== 'object' || body === null) return '';
  const value = (body as Record<string, unknown>)[key];
  return typeof value === 'string' ? value : '';
}

function escapeHtml(value: string): string {
  return value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#39;');
}

export default router;
