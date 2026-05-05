// BAL-02: shared helper — obtains a valid admin session cookie for use in tests
// that hit protected /bff/admin/* routes after the session guard was added.
import request from 'supertest';
import nock from 'nock';
import type { Express } from 'express';

const GO_BACKEND = process.env['BACKEND_URL'] ?? 'http://localhost:8080';

/**
 * Performs a login POST against the BFF and returns the raw `fenix.admin.sid`
 * cookie string suitable for `.set('Cookie', cookie)` in supertest requests.
 *
 * Callers must ensure nock is not intercepting `/auth/login` before calling this.
 */
export async function getAdminSessionCookie(app: Express): Promise<string> {
  nock(GO_BACKEND)
    .post('/auth/login', { email: 'admin@fenix.com', password: 'secret' })
    .reply(200, { token: 'test-jwt-token' });

  const res = await request(app)
    .post('/bff/admin/login')
    .type('form')
    .send({ email: 'admin@fenix.com', password: 'secret' });

  const cookies = res.headers['set-cookie'] as string[] | string | undefined;
  const cookieArray = Array.isArray(cookies) ? cookies : cookies ? [cookies] : [];
  const sessionCookie = cookieArray.find((c: string) => c.startsWith('fenix.admin.sid'));
  return sessionCookie ? (sessionCookie.split(';')[0] ?? '') : '';
}
