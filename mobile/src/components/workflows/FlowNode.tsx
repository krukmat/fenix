// CLSF-82: mobile read-only graph node component
// Task T8.1 - Workflow node colors documented as domain-specific exceptions
import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

import { type FlowNodeBox } from '../../lib/flowLayout';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';

// Workflow node kinds are DSL-specific concepts. Colors are domain-specific
// exceptions to the Command Center theme. These are not CRM statuses but rather
// semantic node types in the workflow DSL that users need to visually distinguish.
//
// Note: These colors may be replaced with theme tokens if they become too
// disparate from the design contract. Current rationale:
// - These are visual node types in workflow DSL (trigger, action, decision, etc.)
// - Distinct colors aid scanning in complex workflow graphs
// - The graph is read-only - no edit capability to change node types on mobile
//
// See: mobile/src/services/api.workflows.ts - WorkflowGraphNode kind field
const KIND_COLOR: Record<string, string> = {
  workflow: '#4F46E5', // purple - workflow container node
  trigger: '#0891B2',  // cyan - trigger event node
  action: '#059669',   // green - action execution node
  decision: '#D97706', // orange - decision/branch node
  grounds: '#7C3AED',  // violet - grounds/justification node
  permit: '#BE185D',   // pink - permission/permit node
  delegate: '#B45309', // brown - delegation node
  invariant: '#6B7280', // gray - invariant/check node
  budget: '#DC2626',   // red - budget/cost node
};

const DEFAULT_COLOR = brandColors.outline;

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
    borderRadius: radius.sm,
    backgroundColor: brandColors.surface,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.sm,
    justifyContent: 'center',
  },
  kind: {
    ...typography.eyebrow,
    fontSize: 10,
    textTransform: 'uppercase',
    color: brandColors.onSurface,
  },
  label: {
    ...typography.labelMD,
    color: brandColors.onSurface,
    marginTop: 2,
  },
});
