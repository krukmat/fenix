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
    expect(res.text).toContain('graph-node action');
    expect(res.text).toContain('graph-node governance');
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
    expect(res.text).toContain('No graph nodes returned for current draft.');
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
    expect(res.text).toContain('graph-node action');
    expect(res.text).toContain('graph-node governance');
    expect(res.text).toContain('sales &lt;followup&gt;');
    expect(res.text).toContain('safe');
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
    expect(empty.text).toContain('No graph nodes returned for current draft.');
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
