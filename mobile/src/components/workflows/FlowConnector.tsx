// CLSF-82: mobile read-only graph connector (line between nodes, no SVG)
import React from 'react';
import { View, StyleSheet } from 'react-native';

import { type FlowConnectorSegment } from '../../lib/flowLayout';

const CONNECTION_COLOR: Record<string, string> = {
  execution: '#6B7280',
  requirement: '#7C3AED',
};

const DEFAULT_COLOR = '#9CA3AF';

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

  return (
    <View
      testID={`flow-connector-${connector.id}`}
      style={[
        styles.line,
        {
          left: connector.start.x,
          top: connector.start.y - 1,
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
    transformOrigin: 'left center',
  },
});
