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
    retrieval_method: 'hybrid_search',
    pii_redacted: true,
    knowledge_item_id: 'ki-42',
  };

  const wrap = (ui: React.ReactElement) => render(<PaperProvider>{ui}</PaperProvider>);

  it('renders [index] and collapsed snippet/title with score chip', () => {
    const { getByText, getByTestId, queryByTestId } = wrap(<EvidenceCard source={source} index={1} testIDPrefix="ev" />);

    expect(getByText(/\[1\]/)).toBeTruthy();
    expect(getByTestId('ev-score')).toBeTruthy();
    expect(getByText('0.95')).toBeTruthy();
    expect(queryByTestId('ev-meta')).toBeNull();
  });

  it('expands on tap and shows full snippet with trust fields', () => {
    const { getByTestId, getByText } = wrap(<EvidenceCard source={source} index={1} testIDPrefix="ev" />);

    fireEvent.press(getByTestId('ev-card'));
    expect(getByText(source.snippet)).toBeTruthy();
    expect(getByTestId('ev-method')).toBeTruthy();
    expect(getByText('Hybrid Search')).toBeTruthy();
    expect(getByTestId('ev-pii')).toBeTruthy();
    expect(getByText('PII redacted')).toBeTruthy();
    expect(getByTestId('ev-knowledge-item')).toBeTruthy();
    expect(getByText('Knowledge item: ki-42')).toBeTruthy();
  });

  it('collapses on second tap', () => {
    const { getByTestId, queryByText } = wrap(<EvidenceCard source={source} index={1} testIDPrefix="ev" />);

    fireEvent.press(getByTestId('ev-card'));
    expect(queryByText(source.snippet)).toBeTruthy();

    fireEvent.press(getByTestId('ev-card'));
    expect(queryByText(source.snippet)).toBeNull();
  });

  it('omits optional trust fields when the source does not provide them', () => {
    const minimalSource = {
      ...source,
      retrieval_method: undefined,
      pii_redacted: undefined,
      knowledge_item_id: undefined,
    };

    const { getByTestId, queryByTestId } = wrap(<EvidenceCard source={minimalSource} index={1} testIDPrefix="ev" />);

    fireEvent.press(getByTestId('ev-card'));

    expect(queryByTestId('ev-meta')).toBeNull();
    expect(queryByTestId('ev-method')).toBeNull();
    expect(queryByTestId('ev-pii')).toBeNull();
    expect(queryByTestId('ev-knowledge-item')).toBeNull();
  });
});
