import React from 'react';
import { describe, it, expect, jest } from '@jest/globals';
import { render, fireEvent, waitFor } from '@testing-library/react-native';
import { Dialog, PaperProvider } from 'react-native-paper';

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
    const { getByTestId, UNSAFE_getByType } = wrap(
      <ActionButton action={action} onExecute={onExecute} testIDPrefix="ab" />,
    );

    fireEvent.press(getByTestId('ab-btn'));
    expect(UNSAFE_getByType(Dialog).props.visible).toBe(true);

    fireEvent.press(getByTestId('ab-cancel'));
    await waitFor(() => expect(UNSAFE_getByType(Dialog).props.visible).toBe(false));
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
