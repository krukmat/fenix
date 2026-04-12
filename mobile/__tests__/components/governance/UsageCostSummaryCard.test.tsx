import React from 'react';
import { describe, expect, it } from '@jest/globals';
import { render } from '@testing-library/react-native';
import { Provider as PaperProvider } from 'react-native-paper';
import { UsageCostSummaryCard } from '../../../src/components/governance/UsageCostSummaryCard';

const summary = {
  totalCost: 0.01567,
  totalInputUnits: 1800,
  totalOutputUnits: 240,
  eventCount: 3,
};

describe('UsageCostSummaryCard', () => {
  it('renders total cost with euro formatting', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <UsageCostSummaryCard summary={summary} testIDPrefix="usc" />
      </PaperProvider>
    );
    expect(getByTestId('usc-total-cost').props.children).toBe('€0.01567');
  });

  it('renders event count, input units and output units', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <UsageCostSummaryCard summary={summary} testIDPrefix="usc" />
      </PaperProvider>
    );
    expect(getByTestId('usc-event-count').props.children).toBe(3);
    expect(getByTestId('usc-input-units').props.children).toBe(1800);
    expect(getByTestId('usc-output-units').props.children).toBe(240);
  });

  it('handles zero values without crashing', () => {
    const { getByTestId } = render(
      <PaperProvider>
        <UsageCostSummaryCard
          summary={{ totalCost: 0, totalInputUnits: 0, totalOutputUnits: 0, eventCount: 0 }}
          testIDPrefix="usc"
        />
      </PaperProvider>
    );
    expect(getByTestId('usc-total-cost').props.children).toBe('€0.00000');
  });
});
