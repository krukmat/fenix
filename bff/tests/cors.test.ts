import request from 'supertest';

jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 10 }),
}));

import app from '../src/app';

describe('BFF CORS allowlist', () => {
  it('sets CORS headers for a configured local builder origin', async () => {
    const res = await request(app)
      .options('/bff/health')
      .set('Origin', 'http://localhost:5173')
      .set('Access-Control-Request-Method', 'GET');

    expect(res.status).toBe(204);
    expect(res.headers['access-control-allow-origin']).toBe('http://localhost:5173');
  });

  it('does not set CORS headers for an unlisted browser origin', async () => {
    const res = await request(app)
      .options('/bff/health')
      .set('Origin', 'https://attacker.example.com')
      .set('Access-Control-Request-Method', 'GET');

    expect(res.status).toBe(200);
    expect(res.headers['access-control-allow-origin']).toBeUndefined();
  });

  it('allows same-origin and server-to-server requests without an Origin header', async () => {
    const res = await request(app).get('/bff/builder');

    expect(res.status).toBe(200);
    expect(res.headers['access-control-allow-origin']).toBeUndefined();
  });
});
