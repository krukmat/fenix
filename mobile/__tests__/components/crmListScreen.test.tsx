// mobile/__tests__/components/crmListScreen.test.tsx
// Task 4.3.td — CRMListScreen component unit tests

import { describe, it, expect, jest } from '@jest/globals';
import React from 'react';
import { Text } from 'react-native';
import { render, fireEvent } from '@testing-library/react-native';
import { CRMListScreen } from '../../src/components/crm/CRMListScreen';
import { PaperProvider } from 'react-native-paper';

interface TestItem {
  id: string;
  name: string;
}

const mockFn = jest.fn;

// FIX: React Native requires strings inside <Text>, not bare fragments
const renderItem = ({ item }: { item: TestItem }) => <Text>{item.name}</Text>;

const defaultProps = {
  data: [] as TestItem[],
  loading: false,
  error: null,
  onRefresh: mockFn(),
  searchValue: '',
  onSearchChange: mockFn(),
  renderItem,
  emptyTitle: 'No items',
  testIDPrefix: 'test',
};

function wrap(ui: React.ReactElement) {
  return render(<PaperProvider>{ui}</PaperProvider>);
}

describe('CRMListScreen', () => {
  it('renders list items when data is non-empty', () => {
    const data: TestItem[] = [
      { id: '1', name: 'Alpha' },
      { id: '2', name: 'Beta' },
    ];
    const { getByText } = wrap(
      <CRMListScreen {...defaultProps} data={data} hasData />
    );
    expect(getByText('Alpha')).toBeTruthy();
    expect(getByText('Beta')).toBeTruthy();
  });

  it('shows loading state when loading=true and data is empty', () => {
    const { getByTestId } = wrap(
      <CRMListScreen {...defaultProps} loading={true} />
    );
    expect(getByTestId('test-loading')).toBeTruthy();
  });

  it('shows empty state when not loading, no error, and data is empty', () => {
    const { getByText } = wrap(<CRMListScreen {...defaultProps} />);
    expect(getByText('No items')).toBeTruthy();
  });

  it('shows search bar and no-results message when hasData=true but filtered data is empty', () => {
    const { getByTestId, queryByText } = wrap(
      <CRMListScreen
        {...defaultProps}
        data={[]}
        hasData={true}
        searchValue="xyz"
      />
    );
    // Search bar must be visible — user can clear query
    expect(getByTestId('test-search')).toBeTruthy();
    // Full-screen empty state must NOT be shown
    expect(queryByText('No items')).toBeNull();
  });

  it('calls onEndReached when end of list is reached', () => {
    const onEndReached = jest.fn();
    const data: TestItem[] = Array.from({ length: 20 }, (_, i) => ({
      id: String(i),
      name: `Item ${i}`,
    }));
    const { getByTestId } = wrap(
      <CRMListScreen
        {...defaultProps}
        data={data}
        hasData
        hasMore
        onEndReached={onEndReached}
      />
    );
    fireEvent(getByTestId('test-flatlist'), 'endReached');
    expect(onEndReached).toHaveBeenCalledTimes(1);
  });

  it('calls onRefresh on pull-to-refresh', () => {
    const onRefresh = jest.fn();
    const data: TestItem[] = [{ id: '1', name: 'Alpha' }];
    const { getByTestId } = wrap(
      <CRMListScreen
        {...defaultProps}
        data={data}
        hasData
        onRefresh={onRefresh}
      />
    );
    fireEvent(getByTestId('test-flatlist'), 'refresh');
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });
});
