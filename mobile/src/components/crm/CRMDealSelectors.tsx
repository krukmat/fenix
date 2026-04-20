import React, { useEffect, useMemo, useRef } from 'react';
import { StyleSheet, Text, TouchableOpacity, View } from 'react-native';
import { useTheme } from 'react-native-paper';
import type { CRMAccount, CRMContact, CRMPipeline, CRMPipelineStage } from '../../services/api';
import {
  normalizeCRMAccount,
  normalizeCRMContact,
  normalizeCRMPipeline,
  normalizeCRMPipelineStage,
} from '../../services/api';
import { useAccounts, useContacts, usePipelineStages, usePipelines } from '../../hooks/useCRM';
import type { ThemeColors } from '../../theme/types';

export type CRMDealSelectorValues = {
  accountId: string;
  contactId: string;
  pipelineId: string;
  stageId: string;
};

type SelectorField = keyof CRMDealSelectorValues;

export const emptyDealSelectorValues: CRMDealSelectorValues = {
  accountId: '',
  contactId: '',
  pipelineId: '',
  stageId: '',
};

function useCRMColors(): ThemeColors {
  const theme = useTheme();
  return theme.colors as ThemeColors;
}

function record(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

function unwrapDataArray<T>(value: unknown): T[] {
  if (Array.isArray(value)) return value as T[];
  const payload = record(value);
  return Array.isArray(payload?.data) ? (payload.data as T[]) : [];
}

function listItems<T>(data: { pages?: unknown[] } | undefined, normalize: (raw: unknown) => T): T[] {
  return (data?.pages ?? []).flatMap((page) => unwrapDataArray<unknown>(page).map(normalize));
}

function directItems<T>(data: unknown, normalize: (raw: unknown) => T): T[] {
  return unwrapDataArray<unknown>(data).map(normalize);
}

function idsSignature(items: { id: string }[]): string {
  return items.map((item) => item.id).join('|');
}

export function validateDealSelectors(
  values: CRMDealSelectorValues,
  contacts: CRMContact[],
  stages: CRMPipelineStage[],
): string | null {
  if (!values.accountId) return 'Account is required';
  if (!values.pipelineId) return 'Pipeline is required';
  if (!values.stageId) return 'Stage is required';
  const contact = contacts.find((item) => item.id === values.contactId);
  if (contact?.accountId && contact.accountId !== values.accountId) return 'Selected contact belongs to another account';
  const stage = stages.find((item) => item.id === values.stageId);
  if (stage?.pipelineId && stage.pipelineId !== values.pipelineId) return 'Selected stage belongs to another pipeline';
  return null;
}

function labelContact(contact: CRMContact): string {
  return [contact.firstName, contact.lastName].filter(Boolean).join(' ') || contact.email || contact.id;
}

function OptionList<T extends { id: string }>({
  title,
  items,
  selectedId,
  label,
  testIDPrefix,
  optional,
  onSelect,
}: {
  title: string;
  items: T[];
  selectedId: string;
  label: (item: T) => string;
  testIDPrefix: string;
  optional?: boolean;
  onSelect: (id: string) => void;
}) {
  const colors = useCRMColors();
  return (
    <View style={styles.group}>
      <Text style={[styles.label, { color: colors.onSurfaceVariant }]}>{title}</Text>
      {optional ? (
        <TouchableOpacity testID={`${testIDPrefix}-none`} style={[styles.option, { borderColor: selectedId ? colors.outline : colors.primary }]} onPress={() => onSelect('')}>
          <Text style={[styles.optionText, { color: colors.onSurface }]}>None</Text>
        </TouchableOpacity>
      ) : null}
      {items.map((item) => (
        <TouchableOpacity
          key={item.id}
          testID={`${testIDPrefix}-${item.id}`}
          style={[styles.option, { borderColor: item.id === selectedId ? colors.primary : colors.outline }]}
          onPress={() => onSelect(item.id)}
        >
          <Text style={[styles.optionText, { color: colors.onSurface }]}>{label(item)}</Text>
        </TouchableOpacity>
      ))}
    </View>
  );
}

export function CRMDealSelectors({
  values,
  onChange,
  onDataChange,
}: {
  values: CRMDealSelectorValues;
  onChange: (field: SelectorField, value: string) => void;
  onDataChange?: (data: CRMDealSelectorData) => void;
}) {
  const colors = useCRMColors();
  const accountsQuery = useAccounts();
  const contactsQuery = useContacts();
  const pipelinesQuery = usePipelines();
  const stagesQuery = usePipelineStages(values.pipelineId);
  const accounts = useMemo(() => listItems(accountsQuery.data, normalizeCRMAccount), [accountsQuery.data]);
  const contacts = useMemo(() => listItems(contactsQuery.data, normalizeCRMContact), [contactsQuery.data]);
  const pipelines = useMemo(() => listItems(pipelinesQuery.data, normalizeCRMPipeline), [pipelinesQuery.data]);
  const stages = useMemo(() => directItems(stagesQuery.data, normalizeCRMPipelineStage), [stagesQuery.data]);
  const loading = accountsQuery.isLoading || contactsQuery.isLoading || pipelinesQuery.isLoading || stagesQuery.isLoading;
  const dataSignatureRef = useRef('');

  useEffect(() => {
    const signature = [
      idsSignature(accounts),
      idsSignature(contacts),
      idsSignature(pipelines),
      idsSignature(stages),
    ].join('::');
    if (signature === dataSignatureRef.current) return;
    dataSignatureRef.current = signature;
    onDataChange?.({ accounts, contacts, pipelines, stages });
  }, [accounts, contacts, onDataChange, pipelines, stages]);

  if (loading) {
    return <Text style={[styles.help, { color: colors.onSurfaceVariant }]}>Loading deal selectors...</Text>;
  }

  return (
    <View testID="crm-deal-selectors">
      <OptionList title="Account" items={accounts} selectedId={values.accountId} label={(account) => account.name} testIDPrefix="crm-deal-selector-account" onSelect={(id) => onChange('accountId', id)} />
      <OptionList title="Contact" items={contacts} selectedId={values.contactId} label={labelContact} testIDPrefix="crm-deal-selector-contact" optional onSelect={(id) => onChange('contactId', id)} />
      <OptionList title="Pipeline" items={pipelines} selectedId={values.pipelineId} label={(pipeline) => pipeline.name} testIDPrefix="crm-deal-selector-pipeline" onSelect={(id) => onChange('pipelineId', id)} />
      <OptionList title="Stage" items={stages} selectedId={values.stageId} label={(stage) => stage.name} testIDPrefix="crm-deal-selector-stage" onSelect={(id) => onChange('stageId', id)} />
    </View>
  );
}

export type CRMDealSelectorData = {
  accounts: CRMAccount[];
  contacts: CRMContact[];
  pipelines: CRMPipeline[];
  stages: CRMPipelineStage[];
};

const styles = StyleSheet.create({
  group: { marginBottom: 14, gap: 8 },
  label: { fontSize: 13, fontWeight: '600', marginBottom: 2 },
  option: { borderWidth: 1, borderRadius: 8, padding: 12 },
  optionText: { fontSize: 15, fontWeight: '600' },
  help: { fontSize: 13, marginBottom: 14 },
});
