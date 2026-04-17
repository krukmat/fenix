// CopilotScreen — route params to initialContext mapping, backward compat
// FR-200 (Copilot embedded), UC-A5: signal-aware context from route params

import React from 'react';
import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import { render } from '@testing-library/react-native';

const mockUseLocalSearchParams = jest.fn();
const mockCopilotPanel = jest.fn();

jest.mock('expo-router', () => ({
  Stack: { Screen: () => null },
  useLocalSearchParams: () => mockUseLocalSearchParams(),
}));

jest.mock('../../src/components/copilot', () => ({
  CopilotPanel: (props: unknown) => {
    mockCopilotPanel(props);
    return null;
  },
}));

describe('CopilotScreen — route params to context mapping', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('passes undefined initialContext when no params are present', () => {
    mockUseLocalSearchParams.mockReturnValue({});
    let Screen: any;
    jest.isolateModules(() => {
      Screen = require('../../app/(tabs)/copilot/index').default;
    });
    render(<Screen />);
    expect(mockCopilotPanel).toHaveBeenCalledWith(
      expect.objectContaining({ initialContext: undefined }),
    );
  });

  it('builds initialContext with all four fields when all params present', () => {
    mockUseLocalSearchParams.mockReturnValue({
      entity_type: 'deal',
      entity_id: 'd-1',
      signal_id: 'sig-1',
      signal_type: 'churn_risk',
    });
    let Screen: any;
    jest.isolateModules(() => {
      Screen = require('../../app/(tabs)/copilot/index').default;
    });
    render(<Screen />);
    expect(mockCopilotPanel).toHaveBeenCalledWith(
      expect.objectContaining({
        initialContext: {
          entityType: 'deal',
          entityId: 'd-1',
          signalId: 'sig-1',
          signalType: 'churn_risk',
        },
      }),
    );
  });

  it('builds initialContext with only entity context when signal params are absent', () => {
    mockUseLocalSearchParams.mockReturnValue({
      entity_type: 'account',
      entity_id: 'a-99',
    });
    let Screen: any;
    jest.isolateModules(() => {
      Screen = require('../../app/(tabs)/copilot/index').default;
    });
    render(<Screen />);
    expect(mockCopilotPanel).toHaveBeenCalledWith(
      expect.objectContaining({
        initialContext: expect.objectContaining({
          entityType: 'account',
          entityId: 'a-99',
        }),
      }),
    );
  });

  it('builds initialContext when only signal_id is present', () => {
    mockUseLocalSearchParams.mockReturnValue({ signal_id: 'sig-42' });
    let Screen: any;
    jest.isolateModules(() => {
      Screen = require('../../app/(tabs)/copilot/index').default;
    });
    render(<Screen />);
    expect(mockCopilotPanel).toHaveBeenCalledWith(
      expect.objectContaining({
        initialContext: expect.objectContaining({ signalId: 'sig-42' }),
      }),
    );
  });
});
