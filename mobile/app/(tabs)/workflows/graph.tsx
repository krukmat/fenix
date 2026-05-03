// CLSF-83: mobile read-only workflow graph review screen
import React, { useEffect, useState } from 'react';
import { View, StyleSheet } from 'react-native';
import { Text, Chip, useTheme } from 'react-native-paper';
import type { MD3Theme } from 'react-native-paper';
import { Stack, useLocalSearchParams } from 'expo-router';

import { workflowApi } from '../../../src/services/api.workflows';
import { layoutWorkflowGraph, type FlowLayoutResult } from '../../../src/lib/flowLayout';
import { FlowCanvas } from '../../../src/components/workflows/FlowCanvas';

type GraphResponse = Awaited<ReturnType<typeof workflowApi.getGraph>>;

type ScreenState =
  | { status: 'loading' }
  | { status: 'error'; message: string }
  | { status: 'empty' }
  | { status: 'ready'; layout: FlowLayoutResult; conformance: GraphResponse['conformance'] };

function makeStyles(theme: MD3Theme) {
  return StyleSheet.create({
    container: { flex: 1, backgroundColor: theme.colors.background },
    center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
    statusText: { color: theme.colors.onSurface, marginTop: 8, textAlign: 'center' },
    errorText: { color: theme.colors.error, marginTop: 8, textAlign: 'center' },
    // WFG-T3: static chip row — View+flexWrap instead of vertical ScrollView so canvas retains full height
    headerRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, padding: 12 },
    canvas: { flex: 1 },
  });
}

export default function WorkflowGraphScreen(): React.ReactElement {
  const { id } = useLocalSearchParams<{ id: string }>();
  const theme = useTheme();
  const styles = makeStyles(theme);
  const [state, setState] = useState<ScreenState>({ status: 'loading' });

  useEffect(() => {
    if (!id) {
      setState({ status: 'error', message: 'No workflow id provided.' });
      return;
    }
    setState({ status: 'loading' });
    workflowApi
      .getGraph(id)
      .then((graph) => {
        if (graph.nodes.length === 0) {
          setState({ status: 'empty' });
          return;
        }
        const layout = layoutWorkflowGraph({ nodes: graph.nodes, edges: graph.edges });
        setState({ status: 'ready', layout, conformance: graph.conformance });
      })
      .catch((err: unknown) => {
        const message = err instanceof Error ? err.message : 'Failed to load graph.';
        setState({ status: 'error', message });
      });
  }, [id]);

  return (
    <View style={styles.container}>
      <Stack.Screen options={{ title: 'Workflow Graph' }} />
      {state.status === 'loading' && (
        <View style={styles.center} testID="graph-loading">
          <Text style={styles.statusText}>Loading graph…</Text>
        </View>
      )}
      {state.status === 'error' && (
        <View style={styles.center} testID="graph-error">
          <Text style={styles.errorText}>{state.message}</Text>
        </View>
      )}
      {state.status === 'empty' && (
        <View style={styles.center} testID="graph-empty">
          <Text style={styles.statusText}>This workflow has no graph nodes yet.</Text>
        </View>
      )}
      {state.status === 'ready' && (
        <>
          <View style={styles.headerRow} testID="graph-conformance-header">
            <Chip>{state.conformance.profile}</Chip>
            {(state.conformance.details ?? []).map((d) => (
              <Chip key={d.code}>{d.code}</Chip>
            ))}
          </View>
          <View style={styles.canvas}>
            <FlowCanvas layout={state.layout} />
          </View>
        </>
      )}
    </View>
  );
}
