// CLSF-82: mobile read-only graph canvas — composes FlowNode + FlowConnector
import React from 'react';
import { ScrollView, View, StyleSheet } from 'react-native';

import { type FlowLayoutResult } from '../../lib/flowLayout';
import { FlowNode } from './FlowNode';
import { FlowConnector } from './FlowConnector';

type Props = {
  layout: FlowLayoutResult;
};

export function FlowCanvas({ layout }: Props): React.ReactElement {
  return (
    <ScrollView horizontal style={styles.scroll} contentContainerStyle={styles.scrollContent}>
      <ScrollView style={styles.scroll} contentContainerStyle={styles.scrollContent}>
        <View
          style={[styles.canvas, { width: layout.bounds.width, height: layout.bounds.height }]}
        >
          {layout.connectors.map((connector) => (
            <FlowConnector key={connector.id} connector={connector} />
          ))}
          {layout.nodes.map((node) => (
            <FlowNode key={node.id} node={node} />
          ))}
        </View>
      </ScrollView>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  scroll: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
  },
  canvas: {
    position: 'relative',
    backgroundColor: '#F9FAFB',
  },
});
