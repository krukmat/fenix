import request from 'supertest';
import app from '../src/app';

describe('GET /bff/metrics', () => {
  it('returns prometheus text format', async () => {
    const res = await request(app).get('/bff/metrics');
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/text\/plain/);
    expect(res.text).toContain('bff_requests_total');
    expect(res.text).toContain('bff_uptime_seconds');
  });

  it('includes counter type declaration', async () => {
    const res = await request(app).get('/bff/metrics');
    expect(res.text).toContain('# TYPE bff_requests_total counter');
    expect(res.text).toContain('# TYPE bff_request_errors_total counter');
  });
});