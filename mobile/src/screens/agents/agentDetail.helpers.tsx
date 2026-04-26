// Task 4.5 — FR-230: Agent Run Detail Screen render helpers + types

import React from 'react';
import { View, Text } from 'react-native';
import { semanticColors } from '../../theme/colors';
import { getAgentStatusColor } from '../../theme/semantic';
import { spacing } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import type { ThemeColors } from '../../theme/types';
import { styles } from './agentDetail.styles';

export interface EvidenceItem {
  source_id: string;
  score: number;
  snippet: string;
}

export interface ToolCall {
  tool_name: string;
  params: Record<string, unknown>;
  result: Record<string, unknown>;
  latency_ms: number;
}

export interface AuditEvent {
  actor_id: string;
  action: string;
  timestamp: string;
  outcome: 'success' | 'denied' | 'error';
}

export interface AgentRunData {
  id: string;
  agent_id: string;
  agent_name: string;
  status:
    | 'running'
    | 'success'
    | 'failed'
    | 'abstained'
    | 'partial'
    | 'escalated'
    | 'accepted'
    | 'rejected'
    | 'delegated';
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
  rejection_reason?: string;
}

export function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

export function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    running: 'Running',
    success: 'Success',
    failed: 'Failed',
    abstained: 'Abstained',
    partial: 'Partial',
    escalated: 'Escalated',
    accepted: 'Accepted',
    rejected: 'Rejected',
    delegated: 'Delegated',
  };
  return labels[status] || status;
}

export function getStatusColor(status: string): string {
  const agentStatusColors: Record<string, string> = {
    partial: semanticColors.warning,
    escalated: semanticColors.info,
    accepted: semanticColors.success,
    rejected: getAgentStatusColor('error'),
    delegated: semanticColors.info,
  };
  return agentStatusColors[status] ?? getAgentStatusColor(status);
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
      <Text style={[typography.mono, { color: colors.onSurfaceVariant }]}>{inputsJson}</Text>
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
        <View key={idx} style={[styles.evidenceCard, { backgroundColor: colors.surface }]}>
          <View style={styles.evidenceHeader}>
            <Text style={{ color: colors.onSurface, fontWeight: '500' }}>Source #{idx + 1}</Text>
            <View style={[styles.scoreBadge, { backgroundColor: getStatusColor('success') }]}>
              <Text style={styles.scoreBadgeText}>{item.score.toFixed(2)}</Text>
            </View>
          </View>
          <Text style={{ color: colors.onSurfaceVariant, fontSize: 12, marginTop: spacing.sm }}>
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
        <View key={idx} style={[styles.reasoningStep, { backgroundColor: colors.surface }]}>
          <Text style={{ color: colors.onSurfaceVariant, fontSize: 12 }}>{step}</Text>
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
        <View key={idx} style={[styles.toolCallCard, { backgroundColor: colors.surface }]}>
          <Text style={[styles.toolName, { color: colors.primary }]}>{call.tool_name}</Text>
          <View style={[styles.codeBlock, { marginTop: spacing.sm, backgroundColor: colors.background }]}>
            <Text style={[typography.mono, { color: colors.onSurfaceVariant }]}>
              {JSON.stringify(call.params, null, 2)}
            </Text>
          </View>
          {call.latency_ms > 0 && (
            <Text style={[typography.monoSM, { color: colors.onSurfaceVariant, marginTop: spacing.xs }]}>
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

function renderRejectionReason(reason: string | undefined, colors: ThemeColors) {
  if (!reason) {
    return <Text style={{ color: colors.onSurfaceVariant }}>No rejection reason provided</Text>;
  }
  return (
    <View style={[styles.outputBlock, { backgroundColor: colors.surface }]}>
      <Text style={{ color: colors.onSurface }}>{reason}</Text>
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
        <View key={idx} style={[styles.auditEvent, { backgroundColor: colors.surface }]}>
          <View style={styles.auditHeader}>
            <Text style={{ color: colors.onSurface }}>{event.action}</Text>
            <Text style={[typography.monoSM, { color: colors.onSurfaceVariant }]}>
              {new Date(event.timestamp).toLocaleString()}
            </Text>
          </View>
          <View style={styles.auditFooter}>
            <Text style={{ color: colors.onSurfaceVariant }}>Actor: {event.actor_id}</Text>
            <View
              style={[
                styles.outcomeBadge,
                {
                  backgroundColor: getAgentStatusColor(event.outcome),
                },
              ]}
            >
              <Text style={styles.outcomeBadgeText}>{event.outcome}</Text>
            </View>
          </View>
        </View>
      ))}
    </View>
  );
}

export function renderContent(run: AgentRunData, colors: ThemeColors) {
  return (
    <>
      <View style={[styles.summaryCard, { backgroundColor: colors.surface }]}>
        <View style={styles.summaryHeader}>
          <Text style={[styles.agentName, { color: colors.onSurface }]}>{run.agent_name}</Text>
          <View testID="run-status-chip" style={[styles.statusBadge, { backgroundColor: getStatusColor(run.status) }]}>
            <Text style={styles.statusBadgeText}>{getStatusLabel(run.status)}</Text>
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
        {run.status === 'delegated' ? (
          <Text testID="agent-run-delegated-note" style={[styles.summaryMetric, { color: colors.primary, marginTop: spacing.sm }]}>
            Delegated to another agent. This is not a human handoff.
          </Text>
        ) : null}
      </View>

      {renderSection('Input', renderInputSection(run.inputs, colors), colors, 'agent-run-inputs')}
      {run.status === 'rejected'
        ? renderSection('Rejection Reason', renderRejectionReason(run.rejection_reason, colors), colors, 'agent-run-rejection-reason')
        : null}
      {renderSection('Evidence Retrieved', renderEvidenceSection(run.evidence_retrieved, colors), colors, 'agent-run-evidence')}
      {renderSection('Reasoning Trace', renderReasoningSection(run.reasoning_trace, colors), colors, 'agent-run-reasoning')}
      {renderSection('Tool Calls', renderToolCallsSection(run.tool_calls, colors), colors, 'agent-run-tool-calls')}
      {renderSection('Output', renderOutputSection(run.output, colors), colors, 'agent-run-output')}
      {renderSection('Audit Events', renderAuditSection(run.audit_events, colors), colors, 'agent-run-audit')}
    </>
  );
}
