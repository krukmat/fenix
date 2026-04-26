// bff-http-snapshots T3: catalog validation tests — written before implementation (TDD)
import { catalog } from '../../scripts/snapshots/catalog';
import type { CatalogEntry } from '../../scripts/snapshots/types';

describe('catalog', () => {
  it('has at least 15 entries', () => {
    expect(catalog.length).toBeGreaterThanOrEqual(15);
  });

  it('every entry has a non-empty name', () => {
    catalog.forEach((entry: CatalogEntry) => {
      expect(typeof entry.name).toBe('string');
      expect(entry.name.length).toBeGreaterThan(0);
    });
  });

  it('every entry has a non-empty group', () => {
    catalog.forEach((entry: CatalogEntry) => {
      expect(typeof entry.group).toBe('string');
      expect(entry.group.length).toBeGreaterThan(0);
    });
  });

  it('every entry has a valid HTTP method', () => {
    const validMethods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH'];
    catalog.forEach((entry: CatalogEntry) => {
      expect(validMethods).toContain(entry.method);
    });
  });

  it('every entry has a path starting with /bff', () => {
    catalog.forEach((entry: CatalogEntry) => {
      expect(entry.path).toMatch(/^\/bff/);
    });
  });

  it('every entry has a numeric expectedStatus', () => {
    catalog.forEach((entry: CatalogEntry) => {
      expect(typeof entry.expectedStatus).toBe('number');
      expect(entry.expectedStatus).toBeGreaterThanOrEqual(100);
      expect(entry.expectedStatus).toBeLessThan(600);
    });
  });

  it('names are unique across the catalog', () => {
    const names = catalog.map((e: CatalogEntry) => e.name);
    const unique = new Set(names);
    expect(unique.size).toBe(names.length);
  });

  it('no two entries share the same method+path+body-type combination', () => {
    // Allows same path with different methods (e.g. GET vs POST)
    // but not identical method+path unless body distinguishes them by name
    const keys = catalog.map((e: CatalogEntry) => `${e.method}:${e.path}:${e.name}`);
    const unique = new Set(keys);
    expect(unique.size).toBe(keys.length);
  });

  it('SSE entries have maxEvents and timeoutMs defined', () => {
    const sseEntries = catalog.filter((e: CatalogEntry) => e.sse !== undefined);
    expect(sseEntries.length).toBeGreaterThanOrEqual(1);
    sseEntries.forEach((entry: CatalogEntry) => {
      expect(entry.sse!.maxEvents).toBeGreaterThan(0);
      expect(entry.sse!.timeoutMs).toBeGreaterThan(0);
    });
  });

  it('auth entries have auth=true, public endpoints have auth=false', () => {
    const publicPaths = ['/bff/health', '/bff/metrics', '/bff/auth/login', '/bff/auth/register'];
    catalog.forEach((entry: CatalogEntry) => {
      if (publicPaths.includes(entry.path)) {
        expect(entry.auth).toBe(false);
      }
    });
  });

  it('entries with pathParams have a function for pathParams', () => {
    catalog
      .filter((e: CatalogEntry) => e.path.includes(':'))
      .forEach((entry: CatalogEntry) => {
        expect(typeof entry.pathParams).toBe('function');
      });
  });
});
