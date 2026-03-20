// Task Mobile P1.7 — FR-200/UC-A5: signal-aware context banner
import React, { useEffect, useMemo, useRef, useState } from 'react';
import { FlatList, StyleSheet, View } from 'react-native';
import { IconButton, Text, TextInput, Banner } from 'react-native-paper';
import { useSSE, type CopilotMessage, type SendContext } from '../../hooks/useSSE';
import { toolApi } from '../../services/api';
import { ActionButton, type SuggestedAction } from './ActionButton';
import { EvidenceCard } from './EvidenceCard';

function MessageBubble({ item }: { item: CopilotMessage }) {
  const isUser = item.role === 'user';
  return (
    <View style={[styles.messageRow, isUser ? styles.userRow : styles.assistantRow]}>
      <View style={[styles.bubble, isUser ? styles.userBubble : styles.assistantBubble]}>
        <Text>{item.content || (item.isStreaming ? '…' : '')}</Text>
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
  listContent: { padding: 12, gap: 8 },
  messageRow: { flexDirection: 'row' },
  userRow: { justifyContent: 'flex-end' },
  assistantRow: { justifyContent: 'flex-start' },
  bubble: { maxWidth: '85%', borderRadius: 12, paddingHorizontal: 10, paddingVertical: 8 },
  userBubble: { backgroundColor: '#D8E8FF' },
  assistantBubble: { backgroundColor: '#F0F0F0' },
  footer: { marginTop: 8, gap: 8 },
  inputBar: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: 8, paddingBottom: 8 },
  input: { flex: 1 },
});
