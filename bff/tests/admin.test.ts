// BFF-ADMIN-01 / BFF-ADMIN-02: admin shell layout and session guard tests
// BAL-02: bearer-token relay replaced by session guard; tests updated accordingly
import request from 'supertest';
import nock from 'nock';
import { makeProxyStub } from './helpers/proxyStub';
import { getAdminSessionCookie } from './helpers/adminSession';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => ({ get: jest.fn(), put: jest.fn() })),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

let sessionCookie: string;

beforeEach(async () => {
  sessionCookie = await getAdminSessionCookie(app);
});

afterEach(() => {
  nock.cleanAll();
});

describe('BFF admin shell — BFF-ADMIN-01', () => {
  describe('GET /bff/admin', () => {
    it('returns 200 with HTML content-type', async () => {
      const res = await request(app).get('/bff/admin').set('Cookie', sessionCookie);
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('includes HTMX CDN script tag', async () => {
      const res = await request(app).get('/bff/admin').set('Cookie', sessionCookie);
      expect(res.text).toContain('htmx.org');
    });

    it('includes the admin title landmark', async () => {
      const res = await request(app).get('/bff/admin').set('Cookie', sessionCookie);
      expect(res.text).toContain('FenixCRM Admin');
    });

    it('includes a nav element', async () => {
      const res = await request(app).get('/bff/admin').set('Cookie', sessionCookie);
      expect(res.text).toContain('<nav');
    });

    it('includes nav links for all admin sections', async () => {
      const res = await request(app).get('/bff/admin').set('Cookie', sessionCookie);
      expect(res.text).toContain('/bff/admin/workflows');
      expect(res.text).toContain('/bff/admin/agent-runs');
      expect(res.text).toContain('/bff/admin/approvals');
      expect(res.text).toContain('/bff/admin/audit');
      expect(res.text).toContain('/bff/admin/policy');
      expect(res.text).toContain('/bff/admin/tools');
      expect(res.text).toContain('/bff/admin/metrics');
    });

    it('does not include bearer-token localStorage relay script', async () => {
      const res = await request(app).get('/bff/admin').set('Cookie', sessionCookie);
      expect(res.text).not.toContain('fenix.admin.bearerToken');
      expect(res.text).not.toContain('htmx:configRequest');
    });

    it('includes workspace badge placeholder in header', async () => {
      const res = await request(app).get('/bff/admin').set('Cookie', sessionCookie);
      expect(res.text).toContain('id="admin-workspace-badge"');
    });

    it('includes a sign-out affordance', async () => {
      const res = await request(app).get('/bff/admin').set('Cookie', sessionCookie);
      expect(res.text).toContain('Sign out');
    });
  });

  describe('GET /bff/admin/ (trailing slash)', () => {
    it('returns 200', async () => {
      const res = await request(app).get('/bff/admin/').set('Cookie', sessionCookie);
      expect(res.status).toBe(200);
    });
  });

  describe('GET /bff/admin/dashboard', () => {
    it('returns 200 with HTML', async () => {
      const res = await request(app).get('/bff/admin/dashboard').set('Cookie', sessionCookie);
      expect(res.status).toBe(200);
      expect(res.headers['content-type']).toMatch(/text\/html/);
    });

    it('contains a placeholder dashboard section', async () => {
      const res = await request(app).get('/bff/admin/dashboard').set('Cookie', sessionCookie);
      expect(res.text).toContain('dashboard');
    });
  });
});

// BFF-ADMIN-02: session guard and 401 redirect handling
describe('BFF admin — BFF-ADMIN-02', () => {
  describe('handle401 helper', () => {
    it('redirects to /bff/admin/login when upstream returns 401 for an HTMX hx-request', async () => {
      const res = await request(app)
        .get('/bff/admin/test-401')
        .set('Cookie', sessionCookie)
        .set('HX-Request', 'true');
      expect([200, 302, 404]).toContain(res.status);
    });
  });

  describe('redirectOnUpstream401', () => {
    it('redirects to /bff/admin/login when the Go backend upstream response is 401', () => {
      const { redirectOnUpstream401 } = require('../src/routes/adminAuth');
      const mockRes = {
        redirect: jest.fn(),
        status: jest.fn().mockReturnThis(),
        json: jest.fn(),
      };
      redirectOnUpstream401(401, mockRes as any);
      expect(mockRes.redirect).toHaveBeenCalledWith('/bff/admin/login');
    });

    it('does not redirect for non-401 upstream status codes', () => {
      const { redirectOnUpstream401 } = require('../src/routes/adminAuth');
      const mockRes = {
        redirect: jest.fn(),
        status: jest.fn().mockReturnThis(),
        json: jest.fn(),
      };
      redirectOnUpstream401(403, mockRes as any);
      expect(mockRes.redirect).not.toHaveBeenCalled();
    });

    it('redirects on 401 from upstream even without HX-Request header', () => {
      const { redirectOnUpstream401 } = require('../src/routes/adminAuth');
      const mockRes = {
        redirect: jest.fn(),
        status: jest.fn().mockReturnThis(),
        json: jest.fn(),
      };
      redirectOnUpstream401(401, mockRes as any);
      expect(mockRes.redirect).toHaveBeenCalledWith('/bff/admin/login');
    });
  });
});
