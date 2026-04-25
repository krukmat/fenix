// CLSF-82: mobile read-only graph node component
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

import { type FlowNodeBox } from '../../lib/flowLayout';

const KIND_COLOR: Record<string, string> = {
  workflow: '#4F46E5',
  trigger: '#0891B2',
  action: '#059669',
  decision: '#D97706',
  grounds: '#7C3AED',
  permit: '#BE185D',
  delegate: '#B45309',
  invariant: '#6B7280',
  budget: '#DC2626',
};

const DEFAULT_COLOR = '#374151';

type Props = {
  node: FlowNodeBox;
};

export function FlowNode({ node }: Props): React.ReactElement {
  const color = KIND_COLOR[node.kind] ?? DEFAULT_COLOR;

  return (
    <View
      testID={`flow-node-${node.id}`}
      style={[
        styles.container,
        {
          left: node.x,
          top: node.y,
          width: node.width,
          height: node.height,
          borderColor: color,
        },
      ]}
    >
      <Text style={[styles.kind, { color }]} numberOfLines={1}>
        {node.kind}
      </Text>
      <Text style={styles.label} numberOfLines={2}>
        {node.label}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    position: 'absolute',
    borderWidth: 2,
    borderRadius: 8,
    backgroundColor: '#FFFFFF',
    paddingHorizontal: 8,
    paddingVertical: 6,
    justifyContent: 'center',
  },
  kind: {
    fontSize: 10,
    fontWeight: '600',
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  label: {
    fontSize: 12,
    color: '#111827',
    fontWeight: '500',
    marginTop: 2,
  },
});
