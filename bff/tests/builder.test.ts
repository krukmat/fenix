import request from 'supertest';

jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 10 }),
}));

import app from '../src/app';
import { createGoClient } from '../src/services/goClient';

const mockCreateGoClient = createGoClient as jest.MockedFunction<typeof createGoClient>;

describe('GET /bff/builder', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('returns the HTMX builder shell HTML', async () => {
    const res = await request(app).get('/bff/builder');

    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/text\/html/);
    expect(res.text).toContain('<title>FenixCRM Builder</title>');
    expect(res.text).toContain('https://unpkg.com/htmx.org@2.0.4');
    expect(res.text).toContain('hx-post="/bff/builder/preview"');
    expect(res.text).toContain('id="builder-preview-status"');
    expect(res.text).toContain('id="builder-editor"');
    expect(res.text).toContain('name="source"');
    expect(res.text).toContain('id="builder-spec-source"');
    expect(res.text).toContain('name="spec_source"');
    expect(res.text).toContain('aria-describedby="builder-diagnostics"');
    expect(res.text).toContain('id="builder-diagnostics"');
    expect(res.text).toContain('Validation diagnostics');
    expect(res.text).toContain('No diagnostics have been run for this draft.');
    expect(res.text).toContain('id="builder-graph"');
    expect(res.text).toContain('data-projection-source="fixture"');
    expect(res.text).toContain('id="builder-workflow-id"');
    expect(res.text).toContain('workflowId: sales_followup');
    expect(res.text).toContain('id="builder-bound-workflow"');
    expect(res.text).toContain('Bound workflow: sales_followup');
    expect(res.text).toContain('Back to workflows');
    expect(res.text).not.toContain('Back to workflow detail');
    expect(res.text).toContain('Builder opened without a bound admin workflow.');
    expect(res.text).toContain('Text save unavailable in standalone mode');
    expect(res.text).toContain('data-workflow-id="sales_followup"');
    expect(res.text).toContain('data-save-action="/bff/builder/visual-authoring/sales_followup"');
    expect(res.text).toContain('Read-only workflow graph preview');
    expect(res.text).toContain('class="graph-edge"');
    expect(res.text).toContain('class="graph-node action"');
    expect(res.text).toContain('class="graph-node governance"');
    expect(res.text).toContain('Live backend refresh is reserved for CLSF-66.');
    expect(res.text).toContain('id="builder-inspector"');
    expect(res.text).toContain('Selected node');
    expect(res.text).toContain('Workflow / sales_followup');
    expect(res.text).toContain('Conformance');
    expect(res.text).toContain('safe fixture');
    expect(res.text).toContain('No graph diagnostics for fixture projection.');
    expect(res.text).toContain("localStorage.getItem('fenix.builder.bearerToken')");
    expect(res.text).toContain("event.detail.headers.Authorization = 'Bearer ' + token");
  });

  it('binds a workflowId from querystring into the shell context and save target', async () => {
    const get = jest.fn()
      .mockResolvedValueOnce({
        data: {
          data: {
            id: 'wf-admin-42',
            name: 'admin_followup',
            dsl_source: 'WORKFLOW admin_followup\nON case.created',
            spec_source: 'CARTA admin_followup\nAGENT ops_assistant',
          },
        },
      })
      .mockResolvedValueOnce({
        data: {
          data: {
            visual_graph: {
              workflow_name: 'admin_followup',
              conformance: { profile: 'safe' },
              nodes: [
                { id: 'wf', kind: 'workflow', label: 'admin_followup', position: { x: 40, y: 40 } },
                { id: 'act', kind: 'action', label: 'notify owner', position: { x: 250, y: 40 } },
              ],
              edges: [
                { id: 'edge-1', from: 'wf', to: 'act', connection_type: 'execution' },
              ],
            },
          },
        },
      });
    mockCreateGoClient.mockReturnValue({ get } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .get('/bff/builder?workflowId=wf-admin-42')
      .set('Authorization', 'Bearer builder-token');

    expect(res.status).toBe(200);
    expect(res.text).toContain('workflowId: wf-admin-42');
    expect(res.text).toContain('Editor: admin_followup');
    expect(res.text).toContain('Bound workflow: wf-admin-42');
    expect(res.text).toContain('/bff/admin/workflows/wf-admin-42');
    expect(res.text).toContain('Editing workflow');
    expect(res.text).toContain('Save text or graph changes here, then return to workflow detail');
    expect(res.text).toContain('hx-post="/bff/builder/save/wf-admin-42"');
    expect(res.text).toContain('Save draft');
    expect(res.text).toContain('data-projection-source="api"');
    expect(res.text).toContain('data-workflow-id="wf-admin-42"');
    expect(res.text).toContain('data-workflow-name="admin_followup"');
    expect(res.text).toContain('data-save-action="/bff/builder/visual-authoring/wf-admin-42"');
    expect(res.text).toContain('WORKFLOW admin_followup');
    expect(res.text).toContain('CARTA admin_followup');
    expect(res.text).toContain('Live workflow projection loaded for the bound workflow.');
    expect(res.text).toContain('Projection loaded from workflow context.');
    expect(res.text).toContain('notify owner');
    expect(res.text).toContain('id="builder-initial-projection"');
    expect(res.text).toContain('&quot;workflow_name&quot;:&quot;admin_followup&quot;');
    expect(mockCreateGoClient).toHaveBeenCalledWith('Bearer builder-token');
    expect(get).toHaveBeenCalledWith('/api/v1/workflows/wf-admin-42');
    expect(get).toHaveBeenCalledWith('/api/v1/workflows/wf-admin-42/graph', { params: { format: 'visual' } });
  });

  it('keeps the standalone shell when workflowId is not provided', async () => {
    const res = await request(app).get('/bff/builder');

    expect(res.status).toBe(200);
    expect(mockCreateGoClient).not.toHaveBeenCalled();
    expect(res.text).toContain('Back to workflows');
    expect(res.text).not.toContain('Back to workflow detail');
  });

  it('renders an inline unauthorized load state when workflow fetch returns 401', async () => {
    const get = jest.fn().mockRejectedValue(Object.assign(new Error('Unauthorized'), {
      response: { status: 401, data: { message: 'Unauthorized' } },
    }));
    mockCreateGoClient.mockReturnValue({ get } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .get('/bff/builder?workflowId=wf-protected')
      .set('Authorization', 'Bearer expired-token');

    expect(res.status).toBe(200);
    expect(res.text).toContain('workflowId: wf-protected');
    expect(res.text).toContain('unauthorized');
    expect(res.text).toContain('Update the token and retry.');
    expect(res.text).toContain('/bff/admin/workflows/wf-protected');
    expect(res.text).toContain('Fix the load issue or return to the admin detail');
  });

  it('renders a not-found load state when workflow fetch returns 404', async () => {
    const get = jest.fn().mockRejectedValue(Object.assign(new Error('Missing'), {
      response: { status: 404, data: { message: 'missing workflow' } },
    }));
    mockCreateGoClient.mockReturnValue({ get } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app).get('/bff/builder?workflowId=wf-missing');

    expect(res.status).toBe(200);
    expect(res.text).toContain('Workflow wf-missing was not found.');
    expect(res.text).toContain('Check the admin link or create the draft again.');
    expect(res.text).toContain('/bff/admin/workflows/wf-missing');
  });

  it('renders a transport failure state when workflow fetch fails unexpectedly', async () => {
    const get = jest.fn().mockRejectedValue(new Error('backend unavailable'));
    mockCreateGoClient.mockReturnValue({ get } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app).get('/bff/builder?workflowId=wf-error');

    expect(res.status).toBe(200);
    expect(res.text).toContain('Workflow wf-error could not be loaded: backend unavailable');
  });

  it('keeps the real editor context and shows a graph warning when visual projection fetch fails', async () => {
    const get = jest.fn()
      .mockResolvedValueOnce({
        data: {
          data: {
            id: 'wf-graph-fail',
            name: 'graph_fail_workflow',
            dsl_source: 'WORKFLOW graph_fail_workflow\nON case.created',
            spec_source: null,
          },
        },
      })
      .mockRejectedValueOnce(new Error('graph unavailable'));
    mockCreateGoClient.mockReturnValue({ get } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app).get('/bff/builder?workflowId=wf-graph-fail');

    expect(res.status).toBe(200);
    expect(res.text).toContain('Editor: graph_fail_workflow');
    expect(res.text).toContain('WORKFLOW graph_fail_workflow');
    expect(res.text).toContain('Visual projection for wf-graph-fail is unavailable: graph unavailable');
    expect(res.text).toContain('data-projection-source="api"');
    expect(res.text).toContain('No graph nodes returned for current draft.');
  });

  it('relays editor source to the Go preview API and returns HTMX fragments', async () => {
    const post = jest.fn().mockResolvedValue({
      data: {
        data: {
          passed: true,
          diagnostics: {},
          conformance: { profile: 'safe' },
          visual_graph: {
            nodes: [
              { id: 'node-workflow', kind: 'workflow', label: 'sales_followup', color: '#2563eb', position: { x: 0, y: 0 } },
            ],
            edges: [],
          },
        },
      },
    });
    mockCreateGoClient.mockReturnValue({ post } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/builder/preview')
      .set('Authorization', 'Bearer builder-token')
      .type('form')
      .send({ source: 'WORKFLOW sales_followup\nON deal.updated', spec_source: 'CARTA sales_followup' });

    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/text\/html/);
    expect(res.text).toContain('Preview synced');
    expect(res.text).toContain('hx-swap-oob="true"');
    expect(res.text).toContain('data-projection-source="api"');
    expect(res.text).toContain('sales_followup');
    expect(res.text).toContain('safe');
    expect(mockCreateGoClient).toHaveBeenCalledWith('Bearer builder-token');
    expect(post).toHaveBeenCalledWith(
      '/api/v1/workflows/preview',
      { dsl_source: 'WORKFLOW sales_followup\nON deal.updated', spec_source: 'CARTA sales_followup' },
      expect.objectContaining({ validateStatus: expect.any(Function) }),
    );
  });

  it('renders diagnostics and graph branches for action and governance nodes', async () => {
    const post = jest.fn().mockResolvedValue({
      data: {
        data: {
          passed: false,
          diagnostics: {
            violations: [{ location: 'line 1', message: 'invalid <tag>' }, { description: 'fallback diagnostic' }],
            warnings: [{ code: 'warn_code', description: 'warn detail' }],
          },
          conformance: {
            profile: 'needs_attention',
            details: [{ message: 'conformance detail' }],
          },
          visual_graph: {
            nodes: [
              { id: 'workflow', kind: 'workflow', label: 'sales_followup', color: '#2563eb', position: { x: 0, y: 0 } },
              { id: 'action', kind: 'action', label: 'notify owner', color: '#16a34a', position: { x: 260, y: 0 } },
              { id: 'grounds', kind: 'grounds', label: 'min_sources', color: '#d97706', position: { x: 520, y: 0 } },
            ],
            edges: [{ from: 'workflow', to: 'action' }, { from: 'missing', to: 'grounds' }],
          },
        },
      },
    });
    mockCreateGoClient.mockReturnValue({ post } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/builder/preview')
      .type('form')
      .send({ source: 'WORKFLOW x\nON y', spec_source: 'CARTA x' });

    expect(res.status).toBe(200);
    expect(res.text).toContain('Preview has diagnostics');
    expect(res.text).toContain('id="builder-canvas-root"');
    expect(res.text).toContain('data-projection-payload=');
    expect(res.text).toContain('data-workflow-name=""');
    expect(res.text).toContain('<strong>line 1</strong>');
    expect(res.text).toContain('&lt;tag&gt;');
    expect(res.text).toContain('<strong>diagnostic</strong>');
    expect(res.text).toContain('<strong>warn_code</strong>');
    expect(res.text).toContain('4 current finding(s).');
  });

  it('falls back when preview envelope is empty and validates status gate function', async () => {
    const post = jest.fn().mockResolvedValue({ data: {} });
    mockCreateGoClient.mockReturnValue({ post } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/builder/preview')
      .set('Content-Type', 'text/plain')
      .send('WORKFLOW plain');

    expect(res.status).toBe(200);
    expect(res.text).toContain('Preview has diagnostics');
    expect(res.text).toContain('data-projection-payload="{&quot;workflow_name&quot;:&quot;&quot;,&quot;nodes&quot;:[],&quot;edges&quot;:[]}"');
    expect(res.text).toContain('No node selected');
    expect(res.text).toContain('No validation diagnostics for current draft.');
    expect(res.text).toContain('unknown');
    expect(post).toHaveBeenCalledWith(
      '/api/v1/workflows/preview',
      { dsl_source: '', spec_source: '' },
      expect.objectContaining({ validateStatus: expect.any(Function) }),
    );
    const options = post.mock.calls[0][2] as { validateStatus: (status: number) => boolean };
    expect(options.validateStatus(499)).toBe(true);
    expect(options.validateStatus(500)).toBe(false);
  });

  it('returns preview error fragments for Error and non-Error failures', async () => {
    const post = jest.fn().mockRejectedValueOnce(new Error('preview exploded')).mockRejectedValueOnce('boom');
    mockCreateGoClient.mockReturnValue({ post } as unknown as ReturnType<typeof createGoClient>);

    const first = await request(app)
      .post('/bff/builder/preview')
      .type('form')
      .send({ source: 'WORKFLOW a\nON b' });
    expect(first.status).toBe(502);
    expect(first.text).toContain('Preview unavailable');
    expect(first.text).toContain('preview_error');
    expect(first.text).toContain('preview exploded');

    const second = await request(app)
      .post('/bff/builder/preview')
      .type('form')
      .send({ source: 'WORKFLOW c\nON d' });
    expect(second.status).toBe(502);
    expect(second.text).toContain('Preview request failed');
  });

  it('saves text edits to the real workflow and resets diagnostics on success', async () => {
    const put = jest.fn().mockResolvedValue({ status: 200, data: { data: { id: 'wf-save' } } });
    mockCreateGoClient.mockReturnValue({ put } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/builder/save/wf-save')
      .set('Authorization', 'Bearer builder-token')
      .type('form')
      .send({ source: 'WORKFLOW saved\nON case.created', spec_source: 'CARTA saved' });

    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/text\/html/);
    expect(res.text).toContain('Draft saved');
    expect(res.text).toContain('No validation diagnostics for current draft.');
    expect(mockCreateGoClient).toHaveBeenCalledWith('Bearer builder-token');
    expect(put).toHaveBeenCalledWith(
      '/api/v1/workflows/wf-save',
      { dsl_source: 'WORKFLOW saved\nON case.created', spec_source: 'CARTA saved' },
      expect.objectContaining({ validateStatus: expect.any(Function) }),
    );
  });

  it('renders inline conflict diagnostics when text save returns 409', async () => {
    const put = jest.fn().mockResolvedValue({
      status: 409,
      data: { message: 'workflow is not editable in active state' },
    });
    mockCreateGoClient.mockReturnValue({ put } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/builder/save/wf-locked')
      .type('form')
      .send({ source: 'WORKFLOW locked\nON case.created', spec_source: '' });

    expect(res.status).toBe(200);
    expect(res.text).toContain('Save failed');
    expect(res.text).toContain('Conflict: workflow is not editable in active state.');
  });

  it('renders inline unauthorized diagnostics when text save returns 401', async () => {
    const put = jest.fn().mockResolvedValue({
      status: 401,
      data: { message: 'Unauthorized' },
    });
    mockCreateGoClient.mockReturnValue({ put } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/builder/save/wf-auth')
      .type('form')
      .send({ source: 'WORKFLOW auth\nON case.created', spec_source: '' });

    expect(res.status).toBe(200);
    expect(res.text).toContain('Save failed');
    expect(res.text).toContain('Unauthorized: Unauthorized. Update the bearer token and retry.');
  });

  it('returns save-unavailable fragments for text save transport failures', async () => {
    const put = jest.fn().mockRejectedValueOnce(new Error('save exploded')).mockRejectedValueOnce('boom');
    mockCreateGoClient.mockReturnValue({ put } as unknown as ReturnType<typeof createGoClient>);

    const first = await request(app)
      .post('/bff/builder/save/wf-error')
      .type('form')
      .send({ source: 'WORKFLOW err\nON case.created', spec_source: '' });
    expect(first.status).toBe(502);
    expect(first.text).toContain('Save unavailable');
    expect(first.text).toContain('save exploded');

    const second = await request(app)
      .post('/bff/builder/save/wf-error')
      .type('form')
      .send({ source: 'WORKFLOW err\nON case.created', spec_source: '' });
    expect(second.status).toBe(502);
    expect(second.text).toContain('Text save request failed');
  });

  it('saves a visual graph and returns HTMX graph fragments', async () => {
    const post = jest.fn().mockResolvedValue({ status: 200, data: { data: { id: 'wf-1' } } });
    const get = jest.fn().mockResolvedValue({
      data: {
        data: {
          visual_graph: {
            conformance: { profile: 'safe' },
            nodes: [
              { id: 'wf', kind: 'workflow', label: 'sales <followup>', color: '#2563eb', position: { x: 0, y: 0 } },
              { id: 'act', kind: 'action', label: 'notify owner', color: '#16a34a', position: { x: 220, y: 0 } },
              { id: 'permit', kind: 'permit', label: 'send_reply', color: '#d97706', position: { x: 440, y: 0 } },
            ],
            edges: [
              { id: 'edge-ok', from: 'wf', to: 'act', connection_type: 'execution' },
              { id: 'edge-missing', from: 'missing', to: 'permit', connection_type: 'requirement' },
            ],
          },
        },
      },
    });
    mockCreateGoClient.mockReturnValue({ post, get } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/builder/visual-authoring/wf-1')
      .set('Authorization', 'Bearer builder-token')
      .send({ graph: { workflow_name: 'sales_followup', nodes: [], edges: [] } });

    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/text\/html/);
    expect(res.text).toContain('Graph saved');
    expect(res.text).toContain('hx-swap-oob="true"');
    expect(res.text).toContain('data-projection-source="api"');
    expect(res.text).toContain('data-workflow-id="wf-1"');
    expect(res.text).toContain('data-workflow-name="wf-1"');
    expect(res.text).toContain('data-projection-payload=');
    expect(res.text).toContain('Live workflow projection loaded for the bound workflow.');
    expect(res.text).toContain('Projection loaded from workflow context.');
    expect(mockCreateGoClient).toHaveBeenCalledWith('Bearer builder-token');
    expect(post).toHaveBeenCalledWith(
      '/api/v1/workflows/wf-1/visual-authoring',
      { graph: { workflow_name: 'sales_followup', nodes: [], edges: [] } },
      expect.objectContaining({ validateStatus: expect.any(Function) }),
    );
    expect(get).toHaveBeenCalledWith('/api/v1/workflows/wf-1/graph', { params: { format: 'visual' } });
    const options = post.mock.calls[0][2] as { validateStatus: (status: number) => boolean };
    expect(options.validateStatus(499)).toBe(true);
    expect(options.validateStatus(500)).toBe(false);
  });

  it('returns JSON for visual authoring saves when requested', async () => {
    const post = jest.fn().mockResolvedValue({ status: 200, data: { data: { id: 'wf-json' } } });
    const get = jest.fn().mockResolvedValue({ data: { data: { visual_graph: {} } } });
    mockCreateGoClient.mockReturnValue({ post, get } as unknown as ReturnType<typeof createGoClient>);

    const res = await request(app)
      .post('/bff/builder/visual-authoring/wf-json')
      .set('Accept', 'application/json')
      .send({ graph: { workflow_name: 'wf-json', nodes: [], edges: [] } });

    expect(res.status).toBe(200);
    expect(res.body).toEqual({ status: 'saved', projection: {} });
  });

  it('renders visual authoring validation diagnostics for HTML and JSON clients', async () => {
    const post = jest.fn()
      .mockResolvedValueOnce({
        status: 422,
        data: {
          data: {
            diagnostics: {
              violations: [{ code: 'bad_node', description: 'bad <node>' }],
              warnings: [{ message: 'warning detail' }],
            },
            conformance: { details: [{ code: 'extended', description: 'extended detail' }] },
          },
        },
      })
      .mockResolvedValueOnce({
        status: 422,
        data: { data: { diagnostics: { violations: [{ code: 'json_bad', message: 'json bad' }] } } },
      });
    mockCreateGoClient.mockReturnValue({ post } as unknown as ReturnType<typeof createGoClient>);

    const html = await request(app)
      .post('/bff/builder/visual-authoring/wf-bad')
      .send({ graph: { workflow_name: 'wf-bad', nodes: [], edges: [] } });
    expect(html.status).toBe(422);
    expect(html.text).toContain('Save failed');
    expect(html.text).toContain('<strong>bad_node</strong>: bad &lt;node&gt;');
    expect(html.text).toContain('<strong>diagnostic</strong>: warning detail');
    expect(html.text).toContain('<strong>extended</strong>: extended detail');

    const json = await request(app)
      .post('/bff/builder/visual-authoring/wf-bad')
      .set('Accept', 'application/json')
      .send({ graph: { workflow_name: 'wf-bad', nodes: [], edges: [] } });
    expect(json.status).toBe(422);
    expect(json.body).toEqual({
      status: 'validation_error',
      diagnostics: [{ code: 'json_bad', message: 'json bad' }],
    });
  });

  it('passes through non-validation upstream responses and handles graph refresh fallback', async () => {
    const post = jest.fn()
      .mockResolvedValueOnce({ status: 403, data: { error: 'forbidden' } })
      .mockResolvedValueOnce({ status: 200, data: { data: { id: 'wf-empty' } } });
    const get = jest.fn().mockRejectedValue(new Error('graph unavailable'));
    mockCreateGoClient.mockReturnValue({ post, get } as unknown as ReturnType<typeof createGoClient>);

    const forbidden = await request(app)
      .post('/bff/builder/visual-authoring/wf-forbidden')
      .send({ graph: { workflow_name: 'wf-forbidden', nodes: [], edges: [] } });
    expect(forbidden.status).toBe(403);
    expect(forbidden.body).toEqual({ error: 'forbidden' });

    const empty = await request(app)
      .post('/bff/builder/visual-authoring/wf-empty')
      .send({ graph: { workflow_name: 'wf-empty', nodes: [], edges: [] } });
    expect(empty.status).toBe(200);
    expect(empty.text).toContain('data-projection-payload="{}"');
    expect(empty.text).toContain('No node selected');
    expect(empty.text).toContain('unknown');
  });

  it('returns JSON relay errors for visual authoring transport failures', async () => {
    const post = jest.fn().mockRejectedValueOnce(new Error('visual exploded')).mockRejectedValueOnce('boom');
    mockCreateGoClient.mockReturnValue({ post } as unknown as ReturnType<typeof createGoClient>);

    const first = await request(app)
      .post('/bff/builder/visual-authoring/wf-error')
      .send({ graph: { workflow_name: 'wf-error', nodes: [], edges: [] } });
    expect(first.status).toBe(502);
    expect(first.body).toEqual({ error: 'visual exploded' });

    const second = await request(app)
      .post('/bff/builder/visual-authoring/wf-error')
      .send({ graph: { workflow_name: 'wf-error', nodes: [], edges: [] } });
    expect(second.status).toBe(502);
    expect(second.body).toEqual({ error: 'Visual authoring request failed' });
  });
});
