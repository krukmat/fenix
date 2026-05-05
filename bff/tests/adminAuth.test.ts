// BAL-01: BFF admin session auth — login page render, credential relay, session management
import request from 'supertest';
import nock from 'nock';
import { makeProxyStub } from './helpers/proxyStub';
import { getAdminSessionCookie } from './helpers/adminSession';

const proxyStub = makeProxyStub();
jest.mock('http-proxy-middleware', () => ({
  createProxyMiddleware: jest.fn(() => proxyStub),
}));

const mockGoClient = { get: jest.fn(), post: jest.fn(), put: jest.fn() };
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(() => mockGoClient),
  pingGoBackend: jest.fn().mockResolvedValue({ reachable: true, latencyMs: 5 }),
}));

import app from '../src/app';

const GO_BACKEND = process.env['BACKEND_URL'] ?? 'http://localhost:8080';

describe('BAL-01 — GET /bff/admin/login', () => {
  it('returns 200 with HTML content-type', async () => {
    const res = await request(app).get('/bff/admin/login');
    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/text\/html/);
  });

  it('renders an email input', async () => {
    const res = await request(app).get('/bff/admin/login');
    expect(res.text).toContain('name="email"');
    expect(res.text).toContain('type="email"');
  });

  it('renders a password input', async () => {
    const res = await request(app).get('/bff/admin/login');
    expect(res.text).toContain('name="password"');
    expect(res.text).toContain('type="password"');
  });

  it('renders a Sign in submit button', async () => {
    const res = await request(app).get('/bff/admin/login');
    expect(res.text).toContain('Sign in');
  });

  it('does not contain fenix.admin.bearerToken', async () => {
    const res = await request(app).get('/bff/admin/login');
    expect(res.text).not.toContain('fenix.admin.bearerToken');
  });

  it('does not contain a bearer token input', async () => {
    const res = await request(app).get('/bff/admin/login');
    expect(res.text).not.toContain('Bearer token');
  });

  it('posts to /bff/admin/login', async () => {
    const res = await request(app).get('/bff/admin/login');
    expect(res.text).toContain('action="/bff/admin/login"');
  });
});

describe('BAL-01 — POST /bff/admin/login success', () => {
  beforeEach(() => {
    nock.cleanAll();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  it('redirects to /bff/admin on valid credentials', async () => {
    nock(GO_BACKEND)
      .post('/auth/login', { email: 'admin@fenix.com', password: 'secret' })
      .reply(200, { token: 'jwt-token-abc' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'admin@fenix.com', password: 'secret' });

    expect(res.status).toBe(302);
    expect(res.headers['location']).toBe('/bff/admin');
  });

  it('sets a session cookie on successful login', async () => {
    nock(GO_BACKEND)
      .post('/auth/login', { email: 'admin@fenix.com', password: 'secret' })
      .reply(200, { token: 'jwt-token-abc' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'admin@fenix.com', password: 'secret' });

    const cookies = res.headers['set-cookie'] as string[] | string | undefined;
    const cookieArray = Array.isArray(cookies) ? cookies : cookies ? [cookies] : [];
    const hasSession = cookieArray.some((c: string) => c.startsWith('fenix.admin.sid'));
    expect(hasSession).toBe(true);
  });

  it('session cookie is HttpOnly', async () => {
    nock(GO_BACKEND)
      .post('/auth/login', { email: 'admin@fenix.com', password: 'secret' })
      .reply(200, { token: 'jwt-token-abc' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'admin@fenix.com', password: 'secret' });

    const cookies = res.headers['set-cookie'] as string[] | string | undefined;
    const cookieArray = Array.isArray(cookies) ? cookies : cookies ? [cookies] : [];
    const sessionCookie = cookieArray.find((c: string) => c.startsWith('fenix.admin.sid'));
    expect(sessionCookie).toContain('HttpOnly');
  });

  it('session cookie is SameSite=Lax', async () => {
    nock(GO_BACKEND)
      .post('/auth/login', { email: 'admin@fenix.com', password: 'secret' })
      .reply(200, { token: 'jwt-token-abc' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'admin@fenix.com', password: 'secret' });

    const cookies = res.headers['set-cookie'] as string[] | string | undefined;
    const cookieArray = Array.isArray(cookies) ? cookies : cookies ? [cookies] : [];
    const sessionCookie = cookieArray.find((c: string) => c.startsWith('fenix.admin.sid'));
    expect(sessionCookie).toContain('SameSite=Lax');
  });
});

describe('BAL-01 — POST /bff/admin/login failure', () => {
  beforeEach(() => {
    nock.cleanAll();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  it('returns 200 and re-renders the login form on 401 from Go', async () => {
    nock(GO_BACKEND)
      .post('/auth/login')
      .reply(401, { message: 'invalid credentials' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'bad@fenix.com', password: 'wrong' });

    expect(res.status).toBe(200);
    expect(res.headers['content-type']).toMatch(/text\/html/);
  });

  it('renders an inline error message on invalid credentials', async () => {
    nock(GO_BACKEND)
      .post('/auth/login')
      .reply(401, { message: 'invalid credentials' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'bad@fenix.com', password: 'wrong' });

    expect(res.text).toContain('Invalid email or password');
  });

  it('still renders the login form inputs after failure', async () => {
    nock(GO_BACKEND)
      .post('/auth/login')
      .reply(401, { message: 'invalid credentials' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'bad@fenix.com', password: 'wrong' });

    expect(res.text).toContain('name="email"');
    expect(res.text).toContain('name="password"');
  });

  it('returns 200 and renders service unavailable error when Go auth returns 503', async () => {
    nock(GO_BACKEND)
      .post('/auth/login')
      .reply(503, { message: 'service unavailable' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'admin@fenix.com', password: 'secret' });

    expect(res.status).toBe(200);
    expect(res.text).toContain('Auth service unavailable');
  });

  it('does not set a session cookie on failed login', async () => {
    nock(GO_BACKEND)
      .post('/auth/login')
      .reply(401, { message: 'invalid credentials' });

    const res = await request(app)
      .post('/bff/admin/login')
      .type('form')
      .send({ email: 'bad@fenix.com', password: 'wrong' });

    const cookies = res.headers['set-cookie'] as string[] | string | undefined;
    const cookieArray = Array.isArray(cookies) ? cookies : cookies ? [cookies] : [];
    const hasSession = cookieArray.some(
      (c: string) => c.startsWith('fenix.admin.sid') && !c.includes('Expires=Thu, 01 Jan 1970'),
    );
    expect(hasSession).toBe(false);
  });
});

// BAL-02: session guard — unauthenticated access redirects to login
describe('BAL-02 — unauthenticated admin access redirects to login', () => {
  it('GET /bff/admin redirects to /bff/admin/login when no session', async () => {
    const res = await request(app).get('/bff/admin');
    expect(res.status).toBe(302);
    expect(res.headers['location']).toBe('/bff/admin/login');
  });

  it('GET /bff/admin/dashboard redirects to /bff/admin/login when no session', async () => {
    const res = await request(app).get('/bff/admin/dashboard');
    expect(res.status).toBe(302);
    expect(res.headers['location']).toBe('/bff/admin/login');
  });

  it('GET /bff/admin/workflows redirects to /bff/admin/login when no session', async () => {
    const res = await request(app).get('/bff/admin/workflows');
    expect(res.status).toBe(302);
    expect(res.headers['location']).toBe('/bff/admin/login');
  });

  it('GET /bff/admin/login is accessible without a session', async () => {
    const res = await request(app).get('/bff/admin/login');
    expect(res.status).toBe(200);
  });
});

// BAL-02: session guard — authenticated access passes through
describe('BAL-02 — authenticated admin access serves the shell', () => {
  beforeEach(() => {
    nock.cleanAll();
    jest.clearAllMocks();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  it('GET /bff/admin returns 200 when session is present', async () => {
    const cookie = await getAdminSessionCookie(app);
    const res = await request(app)
      .get('/bff/admin')
      .set('Cookie', cookie);
    expect(res.status).toBe(200);
  });

  it('admin shell does not contain Bearer token input when authenticated', async () => {
    const cookie = await getAdminSessionCookie(app);
    const res = await request(app)
      .get('/bff/admin')
      .set('Cookie', cookie);
    expect(res.text).not.toContain('Bearer token');
    expect(res.text).not.toContain('fenix.admin.bearerToken');
  });

  it('admin shell contains a sign out button when authenticated', async () => {
    const cookie = await getAdminSessionCookie(app);
    const res = await request(app)
      .get('/bff/admin')
      .set('Cookie', cookie);
    expect(res.text).toContain('Sign out');
  });
});

describe('BAL-03 — logout and session expiry handling', () => {
  beforeEach(() => {
    nock.cleanAll();
    jest.clearAllMocks();
  });

  afterEach(() => {
    nock.cleanAll();
  });

  it('POST /bff/admin/logout redirects to /bff/admin/login', async () => {
    const cookie = await getAdminSessionCookie(app);

    const res = await request(app)
      .post('/bff/admin/logout')
      .set('Cookie', cookie);

    expect(res.status).toBe(302);
    expect(res.headers['location']).toBe('/bff/admin/login');
  });

  it('POST /bff/admin/logout clears the session cookie', async () => {
    const cookie = await getAdminSessionCookie(app);

    const res = await request(app)
      .post('/bff/admin/logout')
      .set('Cookie', cookie);

    const cookies = res.headers['set-cookie'] as string[] | string | undefined;
    const cookieArray = Array.isArray(cookies) ? cookies : cookies ? [cookies] : [];
    const clearedSession = cookieArray.find((c: string) => c.startsWith('fenix.admin.sid='));
    expect(clearedSession).toContain('Expires=Thu, 01 Jan 1970');
  });

  it('logged-out cookie can no longer access protected admin routes', async () => {
    const cookie = await getAdminSessionCookie(app);

    await request(app)
      .post('/bff/admin/logout')
      .set('Cookie', cookie);

    const res = await request(app)
      .get('/bff/admin')
      .set('Cookie', cookie);

    expect(res.status).toBe(302);
    expect(res.headers['location']).toBe('/bff/admin/login');
  });

  it('upstream 401 invalidates the session and redirects to /bff/admin/login', async () => {
    const cookie = await getAdminSessionCookie(app);
    const err = Object.assign(new Error('Unauthorized'), {
      isAxiosError: true,
      response: { status: 401, data: { message: 'Unauthorized' } },
    });
    mockGoClient.get.mockRejectedValue(err);

    const res = await request(app)
      .get('/bff/admin/workflows')
      .set('Cookie', cookie);

    expect(res.status).toBe(302);
    expect(res.headers['location']).toBe('/bff/admin/login');

    const cookies = res.headers['set-cookie'] as string[] | string | undefined;
    const cookieArray = Array.isArray(cookies) ? cookies : cookies ? [cookies] : [];
    const clearedSession = cookieArray.find((c: string) => c.startsWith('fenix.admin.sid='));
    expect(clearedSession).toContain('Expires=Thu, 01 Jan 1970');
  });

  it('expired-session redirect returns a cleared cookie that stays unauthenticated on the next request', async () => {
    const cookie = await getAdminSessionCookie(app);
    const err = Object.assign(new Error('Unauthorized'), {
      isAxiosError: true,
      response: { status: 401, data: { message: 'Unauthorized' } },
    });
    mockGoClient.get.mockRejectedValue(err);

    const expiredRes = await request(app)
      .get('/bff/admin/workflows')
      .set('Cookie', cookie);

    const cookies = expiredRes.headers['set-cookie'] as string[] | string | undefined;
    const cookieArray = Array.isArray(cookies) ? cookies : cookies ? [cookies] : [];
    const clearedSession = cookieArray.find((c: string) => c.startsWith('fenix.admin.sid='));
    const clearedCookie = clearedSession ? clearedSession.split(';')[0] ?? '' : '';

    const nextRes = await request(app)
      .get('/bff/admin/dashboard')
      .set('Cookie', clearedCookie);

    expect(nextRes.status).toBe(302);
    expect(nextRes.headers['location']).toBe('/bff/admin/login');
  });
});
