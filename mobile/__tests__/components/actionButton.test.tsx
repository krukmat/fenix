import React from 'react';
import { describe, it, expect, jest } from '@jest/globals';
import { render, fireEvent, waitFor } from '@testing-library/react-native';
import { PaperProvider } from 'react-native-paper';

import { ActionButton } from '../../src/components/copilot/ActionButton';

describe('ActionButton', () => {
  const action = { label: 'Create task', tool: 'create_task', params: { title: 'Follow up' } };

  const wrap = (ui: React.ReactElement) => render(<PaperProvider>{ui}</PaperProvider>);

  it('renders action label', () => {
    const onExecute = jest.fn<() => Promise<void>>().mockResolvedValue(undefined);
    const { getByText } = wrap(<ActionButton action={action} onExecute={onExecute} testIDPrefix="ab" />);

    expect(getByText('Create task')).toBeTruthy();
  });

  it('opens dialog on press and cancel closes without execute', async () => {
    const onExecute = jest.fn<() => Promise<void>>().mockResolvedValue(undefined);
    const { getByTestId, queryByText } = wrap(<ActionButton action={action} onExecute={onExecute} testIDPrefix="ab" />);

    fireEvent.press(getByTestId('ab-btn'));
    expect(queryByText('Execute: Create task?')).toBeTruthy();

    fireEvent.press(getByTestId('ab-cancel'));
    await waitFor(() => expect(queryByText('Execute: Create task?')).toBeNull());
    expect(onExecute).not.toHaveBeenCalled();
  });

  it('confirm calls onExecute', async () => {
    const onExecute = jest.fn<() => Promise<void>>().mockResolvedValue(undefined);
    const { getByTestId } = wrap(<ActionButton action={action} onExecute={onExecute} testIDPrefix="ab" />);

    fireEvent.press(getByTestId('ab-btn'));
    fireEvent.press(getByTestId('ab-confirm'));

    await waitFor(() => expect(onExecute).toHaveBeenCalledWith(action));
  });
});
