// Task 4.5 — Agent Run Detail Screen

import React from 'react';
import { View, Text, StyleSheet, ScrollView, ActivityIndicator, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useLocalSearchParams, Stack, useRouter } from 'expo-router';
import { useAgentRun } from '../../../src/hooks/useCRM';
import type { ThemeColors } from '../../../src/theme/types';

function useColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

interface EvidenceItem {
  source_id: string;
  score: number;
  snippet: string;
}

interface ToolCall {
  tool_name: string;
  params: Record<string, unknown>;
  result: Record<string, unknown>;
  latency_ms: number;
}

interface AuditEvent {
  actor_id: string;
  action: string;
  timestamp: string;
  outcome: 'success' | 'denied' | 'error';
}

interface AgentRunData {
  id: string;
  agent_id: string;
  agent_name: string;
  status: 'running' | 'success' | 'failed' | 'abstained' | 'partial' | 'escalated';
  triggered_by: string;
  trigger_type: 'manual' | 'event' | 'schedule';
  inputs: Record<string, unknown>;
  evidence_retrieved: EvidenceItem[];
  reasoning_trace: string[];
  tool_calls: ToolCall[];
  output?: string;
  audit_events: AuditEvent[];
  created_at: string;
  started_at: string;
  completed_at?: string;
  latency_ms: number;
  cost_euros: number;
  handoff_status?: string;
}

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    running: 'Running',
    success: 'Success',
    failed: 'Failed',
    abstained: 'Abstained',
    escalated: 'Escalated',
  };
  return labels[status] || status;
}

function getStatusColor(status: string): string {
  const colors: Record<string, string> = {
    running: '#3B82F6',
    success: '#10B981',
    failed: '#EF4444',
    abstained: '#F59E0B',
    partial: '#F97316',
    escalated: '#8B5CF6',
  };
  return colors[status] || '#999';
}

function renderSection(title: string, children: React.ReactNode, colors: ThemeColors, testID?: string) {
  return (
    <View style={styles.section} testID={testID}>
      <Text style={[styles.sectionTitle, { color: colors.onSurface }]}>{title}</Text>
      {children}
    </View>
  );
}

function renderInputSection(inputs: Record<string, unknown>, colors: ThemeColors) {
  const inputsJson = JSON.stringify(inputs, null, 2);
  return (
    <View style={[styles.codeBlock, { backgroundColor: colors.surface }]}>
      <Text style={{ color: colors.onSurfaceVariant, fontSize: 12 }}>{inputsJson}</Text>
    </View>
  );
}

function renderEvidenceSection(evidence: EvidenceItem[], colors: ThemeColors) {
  if (evidence.length === 0) {
    return <Text style={{ color: colors.onSurfaceVariant }}>No evidence retrieved</Text>;
  }

  return (
    <View>
      {evidence.map((item, idx) => (
        <View
          key={idx}
          style={[styles.evidenceCard, { backgroundColor: colors.surface }]}
        >
          <View style={styles.evidenceHeader}>
            <Text style={{ color: colors.onSurface, fontWeight: '500' }}>
              Source #{idx + 1}
            </Text>
            <View style={[styles.scoreBadge, { backgroundColor: getStatusColor('success') }]}>
              <Text style={styles.scoreBadgeText}>{item.score.toFixed(2)}</Text>
            </View>
          </View>
          <Text style={{ color: colors.onSurfaceVariant, fontSize: 12, marginTop: 8 }}>
            {item.snippet}
          </Text>
        </View>
      ))}
    </View>
  );
}

function renderReasoningSection(trace: string[], colors: ThemeColors) {
  if (trace.length === 0) {
    return <Text style={{ color: colors.onSurfaceVariant }}>No reasoning trace</Text>;
  }

  return (
    <View>
      {trace.map((step, idx) => (
        <View
          key={idx}
          style={[styles.reasoningStep, { backgroundColor: colors.surface }]}
        >
          <Text style={{ color: colors.onSurfaceVariant, fontSize: 12 }}>
            {step}
          </Text>
        </View>
      ))}
    </View>
  );
}

function renderToolCallsSection(toolCalls: ToolCall[], colors: ThemeColors) {
  if (toolCalls.length === 0) {
    return <Text style={{ color: colors.onSurfaceVariant }}>No tool calls</Text>;
  }

  return (
    <View>
      {toolCalls.map((call, idx) => (
        <View
          key={idx}
          style={[styles.toolCallCard, { backgroundColor: colors.surface }]}
        >
          <Text style={[styles.toolName, { color: colors.primary }]}>
            {call.tool_name}
          </Text>
          <View style={[styles.codeBlock, { marginTop: 8, backgroundColor: colors.background }]}>
            <Text style={{ color: colors.onSurfaceVariant, fontSize: 12 }}>
              {JSON.stringify(call.params, null, 2)}
            </Text>
          </View>
          {call.latency_ms && (
            <Text style={{ color: colors.onSurfaceVariant, fontSize: 10, marginTop: 4 }}>
              Latency: {formatLatency(call.latency_ms)}
            </Text>
          )}
        </View>
      ))}
    </View>
  );
}

function renderOutputSection(output: string | undefined, colors: ThemeColors) {
  if (!output) {
    return <Text style={{ color: colors.onSurfaceVariant }}>No output generated</Text>;
  }

  return (
    <View style={[styles.outputBlock, { backgroundColor: colors.surface }]}>
      <Text style={{ color: colors.onSurface }}>{output}</Text>
    </View>
  );
}

function renderAuditSection(auditEvents: AuditEvent[], colors: ThemeColors) {
  if (auditEvents.length === 0) {
    return <Text style={{ color: colors.onSurfaceVariant }}>No audit events</Text>;
  }

  return (
    <View>
      {auditEvents.map((event, idx) => (
        <View
          key={idx}
          style={[styles.auditEvent, { backgroundColor: colors.surface }]}
        >
          <View style={styles.auditHeader}>
            <Text style={{ color: colors.onSurface }}>{event.action}</Text>
            <Text style={{ color: colors.onSurfaceVariant, fontSize: 10 }}>
              {new Date(event.timestamp).toLocaleString()}
            </Text>
          </View>
          <View style={styles.auditFooter}>
            <Text style={{ color: colors.onSurfaceVariant }}>
              Actor: {event.actor_id}
            </Text>
            <View
              style={[
                styles.outcomeBadge,
                {
                  backgroundColor:
                    event.outcome === 'success'
                      ? '#4CAF50'
                      : event.outcome === 'denied'
                      ? '#FF9800'
                      : '#F44336',
                },
              ]}
            >
              <Text style={styles.outcomeBadgeText}>
                {event.outcome}
              </Text>
            </View>
          </View>
        </View>
      ))}
    </View>
  );
}

function renderContent(run: AgentRunData, colors: ThemeColors) {
  return (
    <>
      {/* Summary Section */}
      <View style={[styles.summaryCard, { backgroundColor: colors.surface }]}>
        <View style={styles.summaryHeader}>
          <Text style={[styles.agentName, { color: colors.onSurface }]}>
            {run.agent_name}
          </Text>
          <View
            style={[
              styles.statusBadge,
              { backgroundColor: getStatusColor(run.status) },
            ]}
          >
            <Text style={styles.statusBadgeText}>
              {getStatusLabel(run.status)}
            </Text>
          </View>
        </View>
        <View style={styles.summaryMetrics}>
          <Text style={[styles.summaryMetric, { color: colors.onSurfaceVariant }]}>
            {formatLatency(run.latency_ms)}
          </Text>
          <Text style={[styles.summaryMetric, { color: colors.onSurfaceVariant }]}>
            {run.cost_euros.toFixed(4)} €
          </Text>
          <Text style={[styles.summaryMetric, { color: colors.onSurfaceVariant }]}>
            Triggered by: {run.triggered_by}
          </Text>
        </View>
      </View>

      {renderSection('Input', renderInputSection(run.inputs, colors), colors, 'agent-run-inputs')}
      {renderSection('Evidence Retrieved', renderEvidenceSection(run.evidence_retrieved, colors), colors, 'agent-run-evidence')}
      {renderSection('Reasoning Trace', renderReasoningSection(run.reasoning_trace, colors), colors, 'agent-run-reasoning')}
      {renderSection('Tool Calls', renderToolCallsSection(run.tool_calls, colors), colors, 'agent-run-tool-calls')}
      {renderSection('Output', renderOutputSection(run.output, colors), colors, 'agent-run-output')}
      {renderSection('Audit Events', renderAuditSection(run.audit_events, colors), colors, 'agent-run-audit')}
    </>
  );
}

// eslint-disable-next-line complexity
export default function AgentRunDetailScreen() {
  const colors = useColors();
  const router = useRouter();
  const params = useLocalSearchParams<{ id: string | string[] }>();
  const id = Array.isArray(params.id) ? params.id[0] : params.id;
  const { data, isLoading, error } = useAgentRun(id);
  const run: AgentRunData | undefined = data?.data;

  const content = run ? renderContent(run, colors) : null;
  const title = run ? `Run: ${run.agent_name}` : 'Agent Run';

  return (
    <>
      <Stack.Screen
        options={{
          title,
          headerBackTitle: 'Back',
        }}
      />
      <ScrollView style={[styles.container, { backgroundColor: colors.background }]}>
        {isLoading ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <ActivityIndicator size="large" color={colors.primary} />
            <Text style={{ color: colors.onSurfaceVariant, marginTop: 12 }}>
              Loading agent run...
            </Text>
          </View>
        ) : error || !run ? (
          <View style={[styles.centered, { backgroundColor: colors.background }]}>
            <Text style={{ color: colors.error, fontSize: 16 }}>
              {error?.message || 'Agent run not found'}
            </Text>
            <TouchableOpacity
              style={[styles.backButton, { marginTop: 16, backgroundColor: colors.primary }]}
              onPress={() => router.push('/agents')}
            >
              <Text style={styles.backButtonText}>Back to List</Text>
            </TouchableOpacity>
          </View>
        ) : (
          content
        )}
      </ScrollView>
    </>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  centered: { justifyContent: 'center', alignItems: 'center', flex: 1 },
  summaryCard: {
    margin: 16,
    padding: 16,
    borderRadius: 8,
    elevation: 2,
  },
  summaryHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  agentName: {
    fontSize: 20,
    fontWeight: '600',
  },
  statusBadge: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 4,
  },
  statusBadgeText: {
    color: '#FFF',
    fontSize: 11,
    fontWeight: '600',
  },
  summaryMetrics: {
    flexDirection: 'row',
    justifyContent: 'space-around',
  },
  summaryMetric: {
    fontSize: 12,
  },
  section: { padding: 16 },
  sectionTitle: {
    fontSize: 16,
    fontWeight: '600',
    marginBottom: 12,
    textTransform: 'uppercase',
    letterSpacing: 0.5,
  },
  codeBlock: {
    padding: 12,
    borderRadius: 6,
    fontFamily: 'monospace',
  },
  evidenceCard: {
    padding: 12,
    borderRadius: 6,
    marginBottom: 8,
  },
  evidenceHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  scoreBadge: {
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  scoreBadgeText: {
    color: '#FFF',
    fontSize: 10,
    fontWeight: '600',
  },
  reasoningStep: {
    padding: 12,
    borderRadius: 6,
    marginBottom: 8,
    borderLeftWidth: 3,
    borderLeftColor: '#2196F3',
  },
  toolCallCard: {
    padding: 12,
    borderRadius: 6,
    marginBottom: 8,
  },
  toolName: {
    fontSize: 14,
    fontWeight: '600',
  },
  outputBlock: {
    padding: 16,
    borderRadius: 6,
  },
  auditEvent: {
    padding: 12,
    borderRadius: 6,
    marginBottom: 8,
  },
  auditHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  },
  auditFooter: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  outcomeBadge: {
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
  },
  outcomeBadgeText: {
    color: '#FFF',
    fontSize: 10,
    fontWeight: '600',
  },
  backButton: {
    paddingVertical: 8,
    paddingHorizontal: 24,
    borderRadius: 8,
  },
  backButtonText: {
    color: '#FFF',
    fontSize: 14,
    fontWeight: '600',
  },
});

