import React, { useState } from 'react';
import { StyleSheet, Text, TextInput, TouchableOpacity, View } from 'react-native';
import { CRMDetailSection, useCRMColors } from './CoreCRMReadOnly';
import { useCreateActivity, useCreateAttachment, useCreateNote } from '../../hooks/useCRM';
import { useAuthStore } from '../../stores/authStore';

type EntityChildFormsProps = {
  entityType: string;
  entityId: string;
};

type ActivityValues = {
  activityType: string;
  subject: string;
  body: string;
  status: string;
};

type AttachmentValues = {
  filename: string;
  storagePath: string;
  contentType: string;
  sizeBytes: string;
};

const emptyActivity: ActivityValues = {
  activityType: 'task',
  subject: '',
  body: '',
  status: 'open',
};

const emptyAttachment: AttachmentValues = {
  filename: '',
  storagePath: '',
  contentType: '',
  sizeBytes: '',
};

const SIGNED_IN_USER_REQUIRED = 'Signed-in user is required';
const VALIDATION_SOURCE = 'mobile-crm-validation';

function trimOrUndefined(value: string): string | undefined {
  const trimmed = value.trim();
  return trimmed === '' ? undefined : trimmed;
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'CRM request failed';
}

function Field({
  label,
  value,
  onChangeText,
  testID,
  multiline,
  keyboardType,
}: {
  label: string;
  value: string;
  onChangeText: (value: string) => void;
  testID: string;
  multiline?: boolean;
  keyboardType?: 'default' | 'numeric';
}) {
  const colors = useCRMColors();
  return (
    <View style={styles.field}>
      <Text style={[styles.label, { color: colors.onSurfaceVariant }]}>{label}</Text>
      <TextInput
        testID={testID}
        value={value}
        onChangeText={onChangeText}
        multiline={multiline}
        keyboardType={keyboardType}
        style={[
          styles.input,
          multiline ? styles.multiline : null,
          { borderColor: colors.outline, color: colors.onSurface, backgroundColor: colors.surface },
        ]}
      />
    </View>
  );
}

function SubmitButton({ label, testID, disabled, onPress }: { label: string; testID: string; disabled: boolean; onPress: () => void }) {
  const colors = useCRMColors();
  return (
    <TouchableOpacity
      testID={testID}
      style={[styles.submit, { backgroundColor: colors.primary }, disabled ? styles.disabled : null]}
      onPress={onPress}
      disabled={disabled}
    >
      <Text style={[styles.submitText, { color: colors.onPrimary }]}>{label}</Text>
    </TouchableOpacity>
  );
}

function ActivityForm({ entityType, entityId, userId }: EntityChildFormsProps & { userId: string | null }) {
  const createActivity = useCreateActivity();
  const [values, setValues] = useState<ActivityValues>(emptyActivity);
  const [error, setError] = useState<string | null>(null);

  const setField = (field: keyof ActivityValues, value: string) => {
    setError(null);
    setValues((current) => ({ ...current, [field]: value }));
  };

  const onSubmit = async () => {
    if (!userId) return setError(SIGNED_IN_USER_REQUIRED);
    if (!values.subject.trim()) return setError('Activity subject is required');
    try {
      await createActivity.mutateAsync({
        entityType,
        entityId,
        ownerId: userId,
        activityType: values.activityType.trim() || 'task',
        subject: values.subject.trim(),
        body: trimOrUndefined(values.body),
        status: values.status.trim() || 'open',
        metadata: { source: VALIDATION_SOURCE },
      });
      setValues(emptyActivity);
    } catch (submitError) {
      setError(errorMessage(submitError));
    }
    return undefined;
  };

  return (
    <View style={styles.card}>
      <Text style={styles.cardTitle}>Activity</Text>
      <Field label="Type" value={values.activityType} onChangeText={(value) => setField('activityType', value)} testID="crm-entity-child-activity-type" />
      <Field label="Subject" value={values.subject} onChangeText={(value) => setField('subject', value)} testID="crm-entity-child-activity-subject" />
      <Field label="Body" value={values.body} onChangeText={(value) => setField('body', value)} testID="crm-entity-child-activity-body" multiline />
      <Field label="Status" value={values.status} onChangeText={(value) => setField('status', value)} testID="crm-entity-child-activity-status" />
      {error ? <Text style={styles.error}>{error}</Text> : null}
      <SubmitButton label="Add Activity" testID="crm-entity-child-activity-submit" disabled={createActivity.isPending} onPress={onSubmit} />
    </View>
  );
}

function NoteForm({ entityType, entityId, userId }: EntityChildFormsProps & { userId: string | null }) {
  const createNote = useCreateNote();
  const [content, setContent] = useState('');
  const [isInternal, setIsInternal] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const colors = useCRMColors();

  const onSubmit = async () => {
    if (!userId) return setError(SIGNED_IN_USER_REQUIRED);
    if (!content.trim()) return setError('Note content is required');
    try {
      await createNote.mutateAsync({
        entityType,
        entityId,
        authorId: userId,
        content: content.trim(),
        isInternal,
        metadata: { source: VALIDATION_SOURCE },
      });
      setContent('');
      setIsInternal(false);
    } catch (submitError) {
      setError(errorMessage(submitError));
    }
    return undefined;
  };

  return (
    <View style={styles.card}>
      <Text style={styles.cardTitle}>Note</Text>
      <Field label="Content" value={content} onChangeText={(value) => { setError(null); setContent(value); }} testID="crm-entity-child-note-content" multiline />
      <TouchableOpacity
        testID="crm-entity-child-note-internal"
        style={[styles.toggle, { borderColor: isInternal ? colors.primary : colors.outline }]}
        onPress={() => setIsInternal((current) => !current)}
      >
        <Text style={[styles.toggleText, { color: colors.onSurface }]}>{isInternal ? 'Internal' : 'Public'}</Text>
      </TouchableOpacity>
      {error ? <Text style={styles.error}>{error}</Text> : null}
      <SubmitButton label="Add Note" testID="crm-entity-child-note-submit" disabled={createNote.isPending} onPress={onSubmit} />
    </View>
  );
}

function AttachmentForm({ entityType, entityId, userId }: EntityChildFormsProps & { userId: string | null }) {
  const createAttachment = useCreateAttachment();
  const [values, setValues] = useState<AttachmentValues>(emptyAttachment);
  const [error, setError] = useState<string | null>(null);

  const setField = (field: keyof AttachmentValues, value: string) => {
    setError(null);
    setValues((current) => ({ ...current, [field]: value }));
  };

  const sizeBytes = values.sizeBytes.trim() === '' ? undefined : Number(values.sizeBytes);
  const onSubmit = async () => {
    if (!userId) return setError(SIGNED_IN_USER_REQUIRED);
    if (!values.filename.trim()) return setError('Attachment filename is required');
    if (!values.storagePath.trim()) return setError('Attachment storage path is required');
    if (sizeBytes !== undefined && (!Number.isFinite(sizeBytes) || sizeBytes < 0)) {
      return setError('Attachment size must be a positive number');
    }
    try {
      await createAttachment.mutateAsync({
        entityType,
        entityId,
        uploaderId: userId,
        filename: values.filename.trim(),
        storagePath: values.storagePath.trim(),
        contentType: trimOrUndefined(values.contentType),
        sizeBytes,
        metadata: { source: VALIDATION_SOURCE },
      });
      setValues(emptyAttachment);
    } catch (submitError) {
      setError(errorMessage(submitError));
    }
    return undefined;
  };

  return (
    <View style={styles.card}>
      <Text style={styles.cardTitle}>Attachment Metadata</Text>
      <Field label="Filename" value={values.filename} onChangeText={(value) => setField('filename', value)} testID="crm-entity-child-attachment-filename" />
      <Field label="Storage Path" value={values.storagePath} onChangeText={(value) => setField('storagePath', value)} testID="crm-entity-child-attachment-storage-path" />
      <Field label="Content Type" value={values.contentType} onChangeText={(value) => setField('contentType', value)} testID="crm-entity-child-attachment-content-type" />
      <Field label="Size Bytes" value={values.sizeBytes} onChangeText={(value) => setField('sizeBytes', value)} testID="crm-entity-child-attachment-size-bytes" keyboardType="numeric" />
      {error ? <Text style={styles.error}>{error}</Text> : null}
      <SubmitButton label="Add Attachment" testID="crm-entity-child-attachment-submit" disabled={createAttachment.isPending} onPress={onSubmit} />
    </View>
  );
}

export function CRMEntityChildForms({ entityType, entityId }: EntityChildFormsProps) {
  const userId = useAuthStore((state) => state.userId);
  return (
    <CRMDetailSection title="Timeline Updates">
      <View testID="crm-entity-child-forms" style={styles.container}>
        <ActivityForm entityType={entityType} entityId={entityId} userId={userId} />
        <NoteForm entityType={entityType} entityId={entityId} userId={userId} />
        <AttachmentForm entityType={entityType} entityId={entityId} userId={userId} />
      </View>
    </CRMDetailSection>
  );
}

const styles = StyleSheet.create({
  container: { gap: 12 },
  card: { padding: 14, borderRadius: 8, backgroundColor: 'transparent', borderWidth: 1, borderColor: '#D0D5DD' },
  cardTitle: { fontSize: 16, fontWeight: '700', marginBottom: 12 },
  field: { marginBottom: 12 },
  label: { fontSize: 13, fontWeight: '600', marginBottom: 6 },
  input: { borderWidth: 1, borderRadius: 8, minHeight: 44, paddingHorizontal: 12, fontSize: 16 },
  multiline: { minHeight: 84, paddingTop: 10, textAlignVertical: 'top' },
  toggle: { borderWidth: 1, borderRadius: 8, padding: 12, marginBottom: 12 },
  toggleText: { fontSize: 15, fontWeight: '600' },
  error: { color: '#BA1A1A', fontSize: 14, marginBottom: 10 },
  submit: { minHeight: 44, borderRadius: 8, alignItems: 'center', justifyContent: 'center' },
  disabled: { opacity: 0.7 },
  submitText: { fontSize: 15, fontWeight: '700' },
});
