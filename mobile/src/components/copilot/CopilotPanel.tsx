// Task Mobile P1.7 — FR-200/UC-A5: signal-aware context banner
// Task T8.2 - Copilot panel migrated to dark operational surface tokens
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { FlatList, StyleSheet, View } from 'react-native';
import { IconButton, Text, TextInput, Banner } from 'react-native-paper';
import { useSSE, type CopilotMessage, type SendContext } from '../../hooks/useSSE';
import { brandColors, semanticColors } from '../../theme/colors';
import { chipShape, radius, spacing, elevation } from '../../theme/spacing';
import { typography } from '../../theme/typography';
import { toolApi } from '../../services/api';
import { ActionButton, type SuggestedAction } from './ActionButton';
import { EvidenceCard } from './EvidenceCard';

interface MessageBubbleProps {
  item: CopilotMessage;
}

const CONFIDENCE_LABELS: Record<NonNullable<CopilotMessage['confidence']>, string> = {
  high: 'High',
  medium: 'Medium',
  low: 'Low',
};

const CONFIDENCE_COLORS: Record<NonNullable<CopilotMessage['confidence']>, string> = {
  high: semanticColors.confidenceHigh,
  medium: semanticColors.confidenceMed,
  low: semanticColors.confidenceLow,
};

function formatAbstentionReason(reason?: CopilotMessage['abstentionReason']): string {
  switch (reason) {
    case 'insufficient_evidence':
      return 'Copilot abstained because the evidence was not strong enough to support a grounded answer.';
    case 'irrelevant_evidence':
      return 'Copilot abstained because the retrieved evidence did not match the request closely enough.';
    default:
      return 'Copilot abstained because it could not verify a grounded answer from the available evidence.';
  }
}

function MessageBubble({ item }: MessageBubbleProps) {
  const isUser = item.role === 'user';
  const isEmptyAbstention = !isUser && item.answerType === 'abstention' && !item.isStreaming && item.content.trim() === '';

  if (isEmptyAbstention) {
    // The footer trust unit takes over rendering for empty abstentions.
    return null;
  }

  // Use dark operational surfaces: user messages on surface, assistant on surfaceVariant
  // This aligns with Command Center design while keeping bubbles distinguishable
  const bubbleBg = isUser ? brandColors.surfaceVariant : brandColors.surface;
  const bubbleColor = isUser ? brandColors.onSurface : brandColors.onSurface;

  return (
    <View
      style={[styles.messageRow, isUser ? styles.userRow : styles.assistantRow]}
      testID={`copilot-message-${item.id}`}
    >
      <View style={[styles.bubble, { backgroundColor: bubbleBg }]}>
        <Text style={{ color: bubbleColor }}>{item.content || (item.isStreaming ? '…' : '')}</Text>
      </View>
    </View>
  );
}

interface FooterProps {
  lastAssistant?: CopilotMessage;
}

function AbstentionPanel({ reason }: { reason?: CopilotMessage['abstentionReason'] }) {
  return (
    <View style={styles.abstentionPanel} testID="copilot-abstention-panel">
      <Text style={styles.trustEyebrow}>Copilot abstained</Text>
      <Text style={styles.abstentionText}>{formatAbstentionReason(reason)}</Text>
      <View style={styles.manualLane} testID="copilot-abstention-manual-lane">
        <Text style={styles.manualLaneLabel}>Recommended next step</Text>
        <Text style={styles.manualLaneText}>Escalate or handle manually with the evidence below.</Text>
      </View>
    </View>
  );
}

function ConfidenceBadge({ confidence }: { confidence?: CopilotMessage['confidence'] }) {
  if (!confidence) {
    return null;
  }

  const confidenceLabel = CONFIDENCE_LABELS[confidence];
  const confidenceColor = CONFIDENCE_COLORS[confidence];

  return (
    <View style={[styles.confidenceBadge, { backgroundColor: confidenceColor }]} testID="copilot-confidence-badge">
      <Text style={styles.confidenceText}>{`${confidenceLabel} confidence`}</Text>
    </View>
  );
}

function WarningsRow({ warnings }: { warnings?: string[] }) {
  if (!warnings || warnings.length === 0) {
    return null;
  }

  return (
    <View style={styles.warningBlock} testID="copilot-warnings-row">
      <Text style={styles.trustEyebrow}>Warnings</Text>
      {warnings.map((warning, idx) => (
        <View key={`${warning}-${idx}`} style={styles.warningPill} testID={`copilot-warning-${idx}`}>
          <Text style={styles.warningText}>{warning}</Text>
        </View>
      ))}
    </View>
  );
}

function Footer({ lastAssistant }: FooterProps) {
  if (!lastAssistant) return null;

  const isAbstained = lastAssistant.answerType === 'abstention';
  return (
    <View style={styles.footer}>
      {isAbstained ? <AbstentionPanel reason={lastAssistant.abstentionReason} /> : null}
      {isAbstained ? null : <ConfidenceBadge confidence={lastAssistant.confidence} />}
      <WarningsRow warnings={lastAssistant.warnings} />
      {(lastAssistant.evidenceSources ?? []).map((source, idx) => (
        <EvidenceCard key={source.id} source={source} index={idx + 1} testIDPrefix={`evidence-card-${idx}`} />
      ))}

      {(lastAssistant.actions ?? []).map((action, idx) => (
        <ActionButton
          key={`${action.tool}-${action.label}-${idx}`}
          action={action}
          onExecute={async (selected: SuggestedAction) => {
            await toolApi.execute(selected.tool, selected.params);
          }}
          testIDPrefix={`action-${idx + 1}`}
        />
      ))}
    </View>
  );
}

export interface CopilotInitialContext {
  signalId?: string;
  signalType?: string;
  entityType?: string;
  entityId?: string;
}

interface CopilotPanelProps {
  initialContext?: CopilotInitialContext;
  // F9.A5: optional callback fires with the submitted query so callers can trigger a support run
  onSupportTrigger?: (customerQuery: string) => void;
}

function ContextBanner({ context }: { context: CopilotInitialContext }) {
  const parts: string[] = [];
  if (context.signalType) parts.push(`signal: ${context.signalType}`);
  if (context.entityType && context.entityId) parts.push(`${context.entityType} ${context.entityId}`);
  if (parts.length === 0) return null;

  return (
    <Banner
      visible
      testID="copilot-context-banner"
      actions={[]}
      icon="information-outline"
    >
      {`Analyzing ${parts.join(' · ')}`}
    </Banner>
  );
}

function buildSendContext(initialContext?: CopilotInitialContext): SendContext | undefined {
  if (!initialContext) {
    return undefined;
  }

  return {
    entityType: initialContext.entityType,
    entityId: initialContext.entityId,
    signalId: initialContext.signalId,
    signalType: initialContext.signalType,
  };
}

function useCopilotPanelModel(initialContext?: CopilotInitialContext, onSupportTrigger?: (customerQuery: string) => void) {
  const [inputText, setInputText] = useState('');
  const flatListRef = useRef<FlatList<CopilotMessage>>(null);
  const { messages, isStreaming, error, sendQuery } = useSSE();

  const lastAssistant = useMemo(
    () => [...messages].reverse().find((m) => m.role === 'assistant'),
    [messages],
  );

  useEffect(() => {
    if (messages.length > 0) {
      flatListRef.current?.scrollToEnd({ animated: true });
    }
  }, [messages.length]);

  const onSend = () => {
    const trimmed = inputText.trim();
    if (!trimmed || isStreaming) return;
    sendQuery(trimmed, buildSendContext(initialContext));
    onSupportTrigger?.(trimmed);
    setInputText('');
  };

  return {
    inputText,
    setInputText,
    flatListRef,
    messages,
    isStreaming,
    error,
    lastAssistant,
    onSend,
  };
}

export function CopilotPanel({ initialContext, onSupportTrigger }: CopilotPanelProps = {}) {
  const {
    inputText,
    setInputText,
    flatListRef,
    messages,
    isStreaming,
    error,
    lastAssistant,
    onSend,
  } = useCopilotPanelModel(initialContext, onSupportTrigger);

  return (
    <View style={styles.container} testID="copilot-panel">
      {initialContext && <ContextBanner context={initialContext} />}
      <FlatList
        ref={flatListRef}
        data={messages}
        keyExtractor={(item) => item.id}
        renderItem={({ item }) => <MessageBubble item={item} />}
        ListFooterComponent={<Footer lastAssistant={lastAssistant} />}
        contentContainerStyle={styles.listContent}
        testID="copilot-messages"
      />

      <Text testID="copilot-response-text">{lastAssistant?.content || ''}</Text>

      {isStreaming && <Text testID="copilot-streaming">Streaming…</Text>}
      {error && <Text testID="copilot-error">{error}</Text>}

      <View style={styles.inputBar}>
        <TextInput
          mode="outlined"
          value={inputText}
          onChangeText={setInputText}
          placeholder="Ask Copilot..."
          style={styles.input}
          testID="copilot-input"
        />
        <IconButton
          icon="send"
          onPress={onSend}
          disabled={!inputText.trim() || isStreaming}
          testID="copilot-send"
        />
        <IconButton
          icon="send-circle"
          onPress={onSend}
          disabled={!inputText.trim() || isStreaming}
          testID="copilot-send-button"
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1 },
  listContent: { padding: spacing.base, gap: spacing.sm },
  messageRow: { flexDirection: 'row' },
  userRow: { justifyContent: 'flex-end' },
  assistantRow: { justifyContent: 'flex-start' },
  bubble: { maxWidth: '85%', borderRadius: radius.sm, paddingHorizontal: spacing.base, paddingVertical: spacing.sm },
  footer: { marginTop: spacing.sm, gap: spacing.sm },
  confidenceBadge: {
    alignSelf: 'flex-start',
    ...chipShape,
  },
  confidenceText: {
    color: brandColors.onSurface,
    ...typography.labelMD,
  },
  trustEyebrow: {
    color: brandColors.onSurfaceVariant,
    ...typography.eyebrow,
  },
  warningBlock: {
    gap: spacing.xs,
  },
  warningPill: {
    borderRadius: radius.md,
    paddingHorizontal: spacing.sm,
    paddingVertical: spacing.sm,
    backgroundColor: semanticColors.warningContainer,
  },
  warningText: {
    color: semanticColors.onWarningContainer,
  },
  abstentionPanel: {
    borderRadius: radius.md,
    padding: spacing.base,
    gap: spacing.sm,
    backgroundColor: brandColors.surfaceVariant,
    ...elevation.card,
  },
  abstentionText: {
    color: brandColors.onSurface,
  },
  manualLane: {
    borderRadius: radius.sm,
    padding: spacing.sm,
    gap: spacing.xs,
    borderWidth: 1,
    borderColor: brandColors.outline,
    backgroundColor: brandColors.surface,
  },
  manualLaneLabel: {
    color: brandColors.onSurfaceVariant,
    ...typography.labelMD,
  },
  manualLaneText: {
    color: brandColors.onSurface,
  },
  inputBar: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: spacing.sm, paddingBottom: spacing.sm },
  input: { flex: 1 },
});
