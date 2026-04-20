// Task 4.1 — FR-301: goClient unit tests — exercises real createGoClient and pingGoBackend
import axios from 'axios';
import { createGoClient, pingGoBackend } from '../src/services/goClient';

jest.mock('axios', () => {
  const mockAxiosInstance = { get: jest.fn(), post: jest.fn() };
  const mockAxios = {
    create: jest.fn(() => mockAxiosInstance),
    get: jest.fn(),
  };
  return { ...mockAxios, default: mockAxios };
});

const mockedAxios = axios as jest.Mocked<typeof axios>;

describe('createGoClient', () => {
  beforeEach(() => jest.clearAllMocks());

  it('creates an axios instance with Authorization header when token provided', () => {
    createGoClient('Bearer test-token');
    expect(mockedAxios.create).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: 'Bearer test-token' }),
      }),
    );
  });

  it('creates an axios instance without Authorization header when no token', () => {
    createGoClient();
    const callArgs = (mockedAxios.create as jest.Mock).mock.calls[0][0];
    expect(callArgs.headers).not.toHaveProperty('Authorization');
  });
});

describe('pingGoBackend', () => {
  beforeEach(() => jest.clearAllMocks());

  it('returns reachable:true when Go /readyz responds', async () => {
    (mockedAxios.get as jest.Mock).mockResolvedValueOnce({ status: 200 });
    const result = await pingGoBackend();
    expect(result.reachable).toBe(true);
    expect(typeof result.latencyMs).toBe('number');
  });

  it('returns reachable:false when Go /readyz times out or errors', async () => {
    (mockedAxios.get as jest.Mock).mockRejectedValueOnce(new Error('ETIMEDOUT'));
    const result = await pingGoBackend();
    expect(result.reachable).toBe(false);
    expect(typeof result.latencyMs).toBe('number');
  });
});
