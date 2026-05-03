// WFG-T1a: failing geometry tests for FlowConnector midpoint-centering fix
import { describe, it, expect } from '@jest/globals';
import { render } from '@testing-library/react-native';
import { StyleSheet } from 'react-native';

import { FlowConnector } from '../../../src/components/workflows/FlowConnector';
import { type FlowConnectorSegment } from '../../../src/lib/flowLayout';

function getStyle(
  element: ReturnType<ReturnType<typeof render>['getByTestId']>,
): Record<string, unknown> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const raw = (element as any).props?.style;
  return StyleSheet.flatten(raw) ?? {};
}

describe('FlowConnector — midpoint-centering geometry (WFG-T1a)', () => {
  it('Case 1: slightly-angled connector — top and left must use midpoint, not start coords', () => {
    // start=(50,40), end=(250,60): nearly horizontal but start.y ≠ end.y
    // length = sqrt(200²+20²) ≈ 200.998
    // buggy:   left = start.x = 50,  top = start.y - 1 = 39
    // correct: left = (50+250)/2 - length/2 ≈ 49.5,  top = (40+60)/2 - 1 = 49
    // top diverges (39 vs 49) and left diverges (50 vs 49.5) — test catches the bug
    const connector: FlowConnectorSegment = {
      id: 'c1',
      from: 'n1',
      to: 'n2',
      start: { x: 50, y: 40 },
      end: { x: 250, y: 60 },
      connectionType: 'execution',
    };
    const length = Math.sqrt(200 * 200 + 20 * 20);
    const angle = Math.atan2(20, 200);
    const expectedLeft = (50 + 250) / 2 - length / 2;
    const expectedTop = (40 + 60) / 2 - 1;

    const { getByTestId } = render(<FlowConnector connector={connector} />);
    const style = getStyle(getByTestId('flow-connector-c1'));

    expect(style.left as number).toBeCloseTo(expectedLeft, 2);
    expect(style.top as number).toBeCloseTo(expectedTop, 2);
    expect(style.width as number).toBeCloseTo(length, 2);
    const rotateVal = (style.transform as { rotate: string }[])?.[0]?.rotate ?? '';
    expect(parseFloat(rotateVal.replace('rad', ''))).toBeCloseTo(angle, 4);
  });

  it('Case 2: 45° diagonal connector — line center must align with segment midpoint', () => {
    // start=(0,0), end=(100,100)
    // length ≈ 141.42, angle ≈ 0.7854 rad
    // buggy:   left = 0,  top = -1
    // correct: left = 50 - 141.42/2 ≈ -20.71,  top = 50 - 1 = 49
    const connector: FlowConnectorSegment = {
      id: 'c2',
      from: 'n1',
      to: 'n2',
      start: { x: 0, y: 0 },
      end: { x: 100, y: 100 },
      connectionType: 'execution',
    };
    const length = Math.sqrt(100 * 100 + 100 * 100);
    const angle = Math.atan2(100, 100);
    const expectedLeft = 50 - length / 2;
    const expectedTop = 49;

    const { getByTestId } = render(<FlowConnector connector={connector} />);
    const style = getStyle(getByTestId('flow-connector-c2'));

    expect(style.left as number).toBeCloseTo(expectedLeft, 2);
    expect(style.top as number).toBeCloseTo(expectedTop, 2);
    expect(style.width as number).toBeCloseTo(length, 2);
    const rotateVal = (style.transform as { rotate: string }[])?.[0]?.rotate ?? '';
    expect(parseFloat(rotateVal.replace('rad', ''))).toBeCloseTo(angle, 4);
  });

  it('Case 3: zero-length connector renders view with testID but no width or transform style', () => {
    // start == end → length < 1 → early-return guard path
    const connector: FlowConnectorSegment = {
      id: 'c3',
      from: 'n1',
      to: 'n2',
      start: { x: 50, y: 50 },
      end: { x: 50, y: 50 },
      connectionType: 'execution',
    };
    const { getByTestId } = render(<FlowConnector connector={connector} />);
    const style = getStyle(getByTestId('flow-connector-c3'));

    expect(style.width).toBeUndefined();
    expect(style.transform).toBeUndefined();
  });
});
