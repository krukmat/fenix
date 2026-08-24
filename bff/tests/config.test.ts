describe('BFF production configuration', () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.resetModules();
    process.env = { ...originalEnv };
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  function loadConfig(): typeof import('../src/config') {
    // resetModules above ensures each assertion reads its own environment.
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    return require('../src/config') as typeof import('../src/config');
  }

  it('rejects a production start without an explicit session secret', () => {
    process.env['NODE_ENV'] = 'production';
    process.env['BACKEND_URL'] = 'http://backend:8080';
    process.env['BFF_CORS_ALLOWED_ORIGINS'] = 'https://app.example.test';
    delete process.env['SESSION_SECRET'];

    expect(() => loadConfig().validateConfig()).toThrow('Missing required environment variable: SESSION_SECRET');
  });

  it('rejects a production start without explicit CORS origins', () => {
    process.env['NODE_ENV'] = 'production';
    process.env['BACKEND_URL'] = 'http://backend:8080';
    process.env['SESSION_SECRET'] = 'test-session-secret';
    delete process.env['BFF_CORS_ALLOWED_ORIGINS'];

    expect(() => loadConfig().validateConfig()).toThrow('Missing required environment variable: BFF_CORS_ALLOWED_ORIGINS');
  });

  it('accepts explicit production configuration', () => {
    process.env['NODE_ENV'] = 'production';
    process.env['BACKEND_URL'] = 'http://backend:8080';
    process.env['SESSION_SECRET'] = 'test-session-secret';
    process.env['BFF_CORS_ALLOWED_ORIGINS'] = 'https://app.example.test';

    const { config, validateConfig } = loadConfig();

    expect(() => validateConfig()).not.toThrow();
    expect(config.corsAllowedOrigins).toEqual(['https://app.example.test']);
  });
});
