import React from 'react';
import { describe, it, expect } from '@jest/globals';
import { render, fireEvent } from '@testing-library/react-native';
import { PaperProvider } from 'react-native-paper';

import { EvidenceCard } from '../../src/components/copilot/EvidenceCard';

describe('EvidenceCard', () => {
  const source = {
    id: 'ev-1',
    snippet:
      'Este es un snippet muy largo para validar truncado en collapsed mode y comportamiento expand/collapse.',
    score: 0.95,
    timestamp: '2026-02-01T10:00:00Z',
    title: 'Documento de soporte',
  };

  const wrap = (ui: React.ReactElement) => render(<PaperProvider>{ui}</PaperProvider>);

  it('renders [index] and collapsed snippet/title with score chip', () => {
    const { getByText, getByTestId } = wrap(<EvidenceCard source={source} index={1} testIDPrefix="ev" />);

    expect(getByText(/\[1\]/)).toBeTruthy();
    expect(getByTestId('ev-score')).toBeTruthy();
    expect(getByText('0.95')).toBeTruthy();
  });

  it('expands on tap and shows full snippet', () => {
    const { getByTestId, getByText } = wrap(<EvidenceCard source={source} index={1} testIDPrefix="ev" />);

    fireEvent.press(getByTestId('ev-card'));
    expect(getByText(source.snippet)).toBeTruthy();
  });

  it('collapses on second tap', () => {
    const { getByTestId, queryByText } = wrap(<EvidenceCard source={source} index={1} testIDPrefix="ev" />);

    fireEvent.press(getByTestId('ev-card'));
    expect(queryByText(source.snippet)).toBeTruthy();

    fireEvent.press(getByTestId('ev-card'));
    expect(queryByText(source.snippet)).toBeNull();
  });
});
