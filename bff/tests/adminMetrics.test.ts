// BFF-ADMIN-70: metrics dashboard — proxy of Go GET /metrics (Prometheus text)
import request from 'supertest';
import { makeProxyStub } from './helpers/proxyStub';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

const mockGoClient = { get: jest.fn() };
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => mockGoClient),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

const PROMETHEUS_PAYLOAD = [
  '# HELP fenixcrm_requests_total Total HTTP requests',
  '# TYPE fenixcrm_requests_total counter',
  'fenixcrm_requests_total 42',
  '# HELP fenixcrm_request_errors_total Total HTTP errors (5xx)',
  '# TYPE fenixcrm_request_errors_total counter',
  'fenixcrm_request_errors_total 3',
  '# HELP fenixcrm_uptime_seconds Process uptime in seconds',
  '# TYPE fenixcrm_uptime_seconds gauge',
  'fenixcrm_uptime_seconds 1234.56',
].join('\n');

describe('BFF admin metrics — BFF-ADMIN-70', () => {
  beforeEach(() => { jest.clearAllMocks(); });

  describe('GET /bff/admin/metrics', () => {
    it('returns 200 with HTML content-type', async () => {
      mockGoClient.get.mockResolvedValue({ data: PROMETHEUS_PAYLOAD, status: 200 });

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('calls Go GET /metrics', async () => {
      mockGoClient.get.mockResolvedValue({ data: PROMETHEUS_PAYLOAD, status: 200 });

      await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(mockGoClient.get).toHaveBeenCalledWith('/metrics', expect.objectContaining({ responseType: 'text' }));
    });

    it('renders requests_total value', async () => {
      mockGoClient.get.mockResolvedValue({ data: PROMETHEUS_PAYLOAD, status: 200 });

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('42');
    });

    it('renders request_errors_total value', async () => {
      mockGoClient.get.mockResolvedValue({ data: PROMETHEUS_PAYLOAD, status: 200 });

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('3');
    });

    it('renders uptime_seconds value', async () => {
      mockGoClient.get.mockResolvedValue({ data: PROMETHEUS_PAYLOAD, status: 200 });

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('1234.56');
    });

    it('renders metric labels as section headings or card labels', async () => {
      mockGoClient.get.mockResolvedValue({ data: PROMETHEUS_PAYLOAD, status: 200 });

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('requests_total');
      expect(res.text).toContain('request_errors_total');
      expect(res.text).toContain('uptime_seconds');
    });

    it('renders link to governance page for quota data', async () => {
      mockGoClient.get.mockResolvedValue({ data: PROMETHEUS_PAYLOAD, status: 200 });

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(res.text).toContain('/bff/admin/policy');
    });

    it('renders gracefully when payload is empty', async () => {
      mockGoClient.get.mockResolvedValue({ data: '', status: 200 });

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(200);
      expect(res.text).toContain('Metrics');
    });

    it('redirects to /bff/admin when Go backend returns 401', async () => {
      const err = Object.assign(new Error('Unauthorized'), {
        isAxiosError: true,
        response: { status: 401, data: 'Unauthorized' },
      });
      mockGoClient.get.mockRejectedValue(err);

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer expired');

      expect(res.status).toBe(302);
      expect(res.headers['location']).toBe('/bff/admin');
    });

    it('returns 500 on unexpected upstream error', async () => {
      mockGoClient.get.mockRejectedValue(new Error('timeout'));

      const res = await request(app)
        .get('/bff/admin/metrics')
        .set('Authorization', 'Bearer test-token');

      expect(res.status).toBe(500);
    });
  });
});
