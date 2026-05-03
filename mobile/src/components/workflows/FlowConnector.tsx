// CLSF-82: mobile read-only graph connector (line between nodes, no SVG)
import React from 'react';
import { View, StyleSheet } from 'react-native';

import { type FlowConnectorSegment } from '../../lib/flowLayout';
import { brandColors } from '../../theme/colors';

// Workflow connector colors are DSL-domain exceptions, parallel to FlowNode kind
// colors. They distinguish execution flow from requirement dependencies.
const CONNECTION_COLOR: Record<string, string> = {
  execution: '#6B7280',
  requirement: '#7C3AED',
};

const DEFAULT_COLOR = brandColors.onSurfaceVariant;

type Props = {
  connector: FlowConnectorSegment;
};

export function FlowConnector({ connector }: Props): React.ReactElement {
  const dx = connector.end.x - connector.start.x;
  const dy = connector.end.y - connector.start.y;
  const length = Math.sqrt(dx * dx + dy * dy);
  const angle = Math.atan2(dy, dx);
  const color = CONNECTION_COLOR[connector.connectionType] ?? DEFAULT_COLOR;

  if (length < 1) {
    return <View testID={`flow-connector-${connector.id}`} />;
  }

  // WFG-T1b: midpoint-centering so RN's default center pivot aligns with segment midpoint
  const midX = (connector.start.x + connector.end.x) / 2;
  const midY = (connector.start.y + connector.end.y) / 2;

  return (
    <View
      testID={`flow-connector-${connector.id}`}
      style={[
        styles.line,
        {
          left: midX - length / 2,
          top: midY - 1,
          width: length,
          backgroundColor: color,
          transform: [{ rotate: `${angle}rad` }],
        },
      ]}
    />
  );
}

const styles = StyleSheet.create({
  line: {
    position: 'absolute',
    height: 2,
  },
});
