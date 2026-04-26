// Task Mobile P1.7 — FR-200/UC-A5: signal-aware context banner
// Task T8.2 - Copilot panel migrated to dark operational surface tokens
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { FlatList, StyleSheet, View } from 'react-native';
import { IconButton, Text, TextInput, Banner } from 'react-native-paper';
import { useSSE, type CopilotMessage, type SendContext } from '../../hooks/useSSE';
import { brandColors } from '../../theme/colors';
import { radius, spacing } from '../../theme/spacing';
import { toolApi } from '../../services/api';
import { ActionButton, type SuggestedAction } from './ActionButton';
import { EvidenceCard } from './EvidenceCard';

interface MessageBubbleProps {
  item: CopilotMessage;
}

function MessageBubble({ item }: MessageBubbleProps) {
  const isUser = item.role === 'user';
  // Use dark operational surfaces: user messages on surface, assistant on surfaceVariant
  // This aligns with Command Center design while keeping bubbles distinguishable
  const bubbleBg = isUser ? brandColors.surfaceVariant : brandColors.surface;
  const bubbleColor = isUser ? brandColors.onSurface : brandColors.onSurface;

  return (
    <View style={[styles.messageRow, isUser ? styles.userRow : styles.assistantRow]}>
      <View style={[styles.bubble, { backgroundColor: bubbleBg }]}>
        <Text style={{ color: bubbleColor }}>{item.content || (item.isStreaming ? '…' : '')}</Text>
      </View>
    </View>
  );
}

interface FooterProps {
  lastAssistant?: CopilotMessage;
}

function Footer({ lastAssistant }: FooterProps) {
  if (!lastAssistant) return null;

  return (
    <View style={styles.footer}>
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

export function CopilotPanel({ initialContext }: CopilotPanelProps = {}) {
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

  const buildContext = (): SendContext | undefined => {
    if (!initialContext) return undefined;
    return {
      entityType: initialContext.entityType,
      entityId: initialContext.entityId,
      signalId: initialContext.signalId,
      signalType: initialContext.signalType,
    };
  };

  const onSend = () => {
    const trimmed = inputText.trim();
    if (!trimmed || isStreaming) return;
    sendQuery(trimmed, buildContext());
    setInputText('');
  };

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
  inputBar: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: spacing.sm, paddingBottom: spacing.sm },
  input: { flex: 1 },
});
