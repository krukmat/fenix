// W2-T2: renderDealItem moved to src/components/crm/DealItem.tsx
import { describe, it, expect } from '@jest/globals';
import { renderDealItem } from '../../src/components/crm/DealItem';

const mockColors = {
  background: '#fff',
  surface: '#f5f5f5',
  surfaceVariant: '#eee',
  primary: '#6200EE',
  onPrimary: '#fff',
  onSurface: '#000',
  onSurfaceVariant: '#666',
  error: '#B00020',
  outline: '#ccc',
};

const mockRouter = {
  push: () => undefined,
} as any;

describe('Deals list — status chip', () => {
  function collectTestIDs(node: unknown, acc: string[] = []): string[] {
    if (!node || typeof node !== 'object') return acc;

    const n = node as { props?: { testID?: string; children?: unknown } };
    if (n.props?.testID) acc.push(n.props.testID);

    const children = n.props?.children;
    if (Array.isArray(children)) {
      children.forEach((c) => collectTestIDs(c, acc));
    } else if (children) {
      collectTestIDs(children, acc);
    }

    return acc;
  }

  it('renders chip for won deal', () => {
    const deal = { id: '1', name: 'Deal A', status: 'won' as const, value: 1000, accountName: 'Acme' };
    const element = renderDealItem({ item: deal }, mockColors, mockRouter);
    expect(collectTestIDs(element)).toContain('deal-status-won');
  });

  it('renders chip for lost deal', () => {
    const deal = { id: '2', name: 'Deal B', status: 'lost' as const, value: 500, accountName: 'Corp' };
    const element = renderDealItem({ item: deal }, mockColors, mockRouter);
    expect(collectTestIDs(element)).toContain('deal-status-lost');
  });

  it('renders chip for open deal', () => {
    const deal = { id: '3', name: 'Deal C', status: 'open' as const, value: 2000, accountName: 'XYZ' };
    const element = renderDealItem({ item: deal }, mockColors, mockRouter);
    expect(collectTestIDs(element)).toContain('deal-status-open');
  });
});