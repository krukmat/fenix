// Task 4.1 — FR-301: Auth relay tests (TDD — written before implementation)
import request from 'supertest';

// Mock goClient to avoid real HTTP calls to Go backend
jest.mock('../src/services/goClient', () => ({
  createGoClient: jest.fn(),
  pingGoBackend: jest.fn(),
}));

import { createGoClient } from '../src/services/goClient';
import app from '../src/app';

const mockCreateGoClient = createGoClient as jest.MockedFunction<typeof createGoClient>;

describe('Auth relay routes', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('POST /bff/auth/login', () => {
    it('relays login credentials to Go and returns JWT token', async () => {
      const mockToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test';
      const mockResponse = {
        token: mockToken,
        user: { id: 'user-123', email: 'test@example.com' },
      };

      mockCreateGoClient.mockReturnValue({
        post: jest.fn().mockResolvedValue({ data: mockResponse, status: 200 }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .post('/bff/auth/login')
        .send({ email: 'test@example.com', password: 'password123' });

      expect(res.status).toBe(200);
      expect(res.body).toMatchObject({
        token: mockToken,
        user: { id: 'user-123' },
      });
    });

    it('returns 401 when Go backend rejects credentials', async () => {
      const axiosError = Object.assign(new Error('Unauthorized'), {
        response: {
          status: 401,
          data: { message: 'Invalid credentials' },
        },
        isAxiosError: true,
      });

      mockCreateGoClient.mockReturnValue({
        post: jest.fn().mockRejectedValue(axiosError),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .post('/bff/auth/login')
        .send({ email: 'wrong@example.com', password: 'wrong' });

      expect(res.status).toBe(401);
    });
  });

  describe('POST /bff/auth/register', () => {
    it('relays register data to Go and returns created user', async () => {
      const mockResponse = {
        token: 'new-token',
        user: { id: 'user-456', email: 'new@example.com' },
      };

      mockCreateGoClient.mockReturnValue({
        post: jest.fn().mockResolvedValue({ data: mockResponse, status: 201 }),
      } as unknown as ReturnType<typeof createGoClient>);

      const res = await request(app)
        .post('/bff/auth/register')
        .send({ email: 'new@example.com', password: 'securepass', name: 'Test User' });

      expect(res.status).toBe(201);
      expect(res.body.token).toBeDefined();
    });
  });
});
