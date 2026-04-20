import React from 'react';
import { useLocalSearchParams, useRouter } from 'expo-router';
import {
  CRMDetailSection,
  CRMDetailShell,
  CRMReadOnlyRow,
  asText,
  unwrapDataArray,
} from './CoreCRMReadOnly';
import type { CRMAccount, CRMCase, CRMContact, CRMDeal, CRMLead } from '../../services/api';
import {
  normalizeCRMAccount,
  normalizeCRMCase,
  normalizeCRMContact,
  normalizeCRMDeal,
  normalizeCRMLead,
} from '../../services/api';
import { useAccount, useCase, useContact, useDeal, useLead } from '../../hooks/useCRM';
import { CRMEntityChildForms } from './CRMEntityChildForms';

type DetailPayload = Record<string, unknown> | null;

const NOT_AVAILABLE = 'Not available';
const NOT_SPECIFIED = 'Not specified';

function useRouteId(): string {
  const params = useLocalSearchParams<{ id: string | string[] }>();
  return Array.isArray(params.id) ? params.id[0] : params.id;
}

function record(value: unknown): DetailPayload {
  return value !== null && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

function relatedList<T>(payload: DetailPayload, key: string, normalize: (raw: unknown) => T): T[] {
  return unwrapDataArray<unknown>(payload?.[key]).map(normalize);
}

function accountMeta(account: CRMAccount) {
  return [
    { label: 'Industry', value: account.industry || NOT_SPECIFIED },
    { label: 'Email', value: account.email || NOT_AVAILABLE },
    { label: 'Phone', value: account.phone || NOT_AVAILABLE },
    { label: 'Website', value: account.website || NOT_AVAILABLE },
  ];
}

function contactMeta(contact: CRMContact) {
  return [
    { label: 'Email', value: contact.email || NOT_AVAILABLE },
    { label: 'Phone', value: contact.phone || NOT_AVAILABLE },
    { label: 'Title', value: contact.title || NOT_SPECIFIED },
    { label: 'Account', value: contact.accountId || 'Not linked' },
  ];
}

function leadMeta(lead: CRMLead) {
  return [
    { label: 'Status', value: lead.status || NOT_SPECIFIED },
    { label: 'Source', value: lead.source || NOT_SPECIFIED },
    { label: 'Score', value: lead.score === undefined ? 'Not scored' : String(lead.score) },
  ];
}

function dealMeta(deal: CRMDeal) {
  return [
    { label: 'Status', value: deal.status || 'open' },
    { label: 'Amount', value: deal.amount === undefined ? NOT_SPECIFIED : `$${deal.amount.toLocaleString()}` },
    { label: 'Pipeline', value: deal.pipelineId || NOT_SPECIFIED },
    { label: 'Stage', value: deal.stageId || NOT_SPECIFIED },
  ];
}

function caseMeta(caseData: CRMCase) {
  return [
    { label: 'Status', value: caseData.status || 'open' },
    { label: 'Priority', value: caseData.priority || 'medium' },
    { label: 'Channel', value: caseData.channel || NOT_SPECIFIED },
    { label: 'SLA', value: caseData.slaDeadline || 'Not set' },
  ];
}

function RelatedContacts({ contacts }: { contacts: CRMContact[] }) {
  const router = useRouter();
  if (contacts.length === 0) return <CRMDetailSection title="Contacts" empty="No related contacts" />;
  return (
    <CRMDetailSection title="Contacts">
      {contacts.map((contact, index) => (
        <CRMReadOnlyRow
          key={contact.id}
          title={[contact.firstName, contact.lastName].filter(Boolean).join(' ') || contact.email || 'Unknown Contact'}
          subtitle={contact.title}
          meta={contact.email}
          testID={`crm-account-contact-${index}`}
          onPress={() => router.push(`/crm/contacts/${contact.id}`)}
        />
      ))}
    </CRMDetailSection>
  );
}

function RelatedDeals({ deals }: { deals: CRMDeal[] }) {
  const router = useRouter();
  if (deals.length === 0) return <CRMDetailSection title="Deals" empty="No related deals" />;
  return (
    <CRMDetailSection title="Deals">
      {deals.map((deal, index) => (
        <CRMReadOnlyRow
          key={deal.id}
          title={deal.title}
          subtitle={deal.status}
          meta={deal.amount === undefined ? undefined : `$${deal.amount.toLocaleString()}`}
          testID={`crm-account-deal-${index}`}
          onPress={() => router.push(`/crm/deals/${deal.id}`)}
        />
      ))}
    </CRMDetailSection>
  );
}

function accountName(payload: DetailPayload): string | undefined {
  return asText(record(payload?.account)?.name);
}

export function CoreCRMAccountDetail() {
  const id = useRouteId();
  const { data, isLoading, error } = useAccount(id);
  const payload = record(data);
  const account = normalizeCRMAccount(payload?.account ?? payload);
  const contacts = relatedList(payload, 'contacts', normalizeCRMContact);
  const deals = relatedList(payload, 'deals', normalizeCRMDeal);
  return (
    <CRMDetailShell
      title={account.name}
      subtitle={account.description}
      metadata={accountMeta(account)}
      loading={isLoading}
      error={error?.message}
      testIDPrefix="crm-account-detail"
    >
      <RelatedContacts contacts={contacts} />
      <RelatedDeals deals={deals} />
      <CRMEntityChildForms entityType="account" entityId={id} />
    </CRMDetailShell>
  );
}

export function CoreCRMContactDetail() {
  const id = useRouteId();
  const { data, isLoading, error } = useContact(id);
  const contact = normalizeCRMContact(record(data)?.contact ?? data);
  const title = [contact.firstName, contact.lastName].filter(Boolean).join(' ') || contact.email || 'Unknown Contact';
  return (
    <CRMDetailShell
      title={title}
      subtitle={contact.title}
      metadata={contactMeta(contact)}
      loading={isLoading}
      error={error?.message}
      testIDPrefix="crm-contact-detail"
    >
      <CRMEntityChildForms entityType="contact" entityId={id} />
    </CRMDetailShell>
  );
}

export function CoreCRMLeadDetail() {
  const id = useRouteId();
  const { data, isLoading, error } = useLead(id);
  const lead = normalizeCRMLead(record(data)?.lead ?? data);
  const title = asText(lead.metadata.name, `Lead ${lead.id || id}`);
  return (
    <CRMDetailShell
      title={title}
      subtitle={asText(lead.metadata.email)}
      metadata={leadMeta(lead)}
      loading={isLoading}
      error={error?.message}
      testIDPrefix="crm-lead-detail"
    >
      <CRMEntityChildForms entityType="lead" entityId={id} />
    </CRMDetailShell>
  );
}

export function CoreCRMDealDetail() {
  const id = useRouteId();
  const { data, isLoading, error } = useDeal(id);
  const payload = record(data);
  const deal = normalizeCRMDeal(payload?.deal ?? payload);
  return (
    <CRMDetailShell
      title={deal.title}
      subtitle={accountName(payload)}
      metadata={dealMeta(deal)}
      loading={isLoading}
      error={error?.message}
      testIDPrefix="crm-deal-detail"
    >
      <RelatedContacts contacts={relatedList(payload, 'contact', normalizeCRMContact)} />
      <CRMEntityChildForms entityType="deal" entityId={id} />
    </CRMDetailShell>
  );
}

export function CoreCRMCaseDetail() {
  const id = useRouteId();
  const { data, isLoading, error } = useCase(id);
  const payload = record(data);
  const caseData = normalizeCRMCase(payload?.case ?? payload);
  return (
    <CRMDetailShell
      title={caseData.subject}
      subtitle={caseData.description}
      metadata={caseMeta(caseData)}
      loading={isLoading}
      error={error?.message}
      testIDPrefix="crm-case-detail"
    >
      <RelatedContacts contacts={relatedList(payload, 'contact', normalizeCRMContact)} />
      <CRMEntityChildForms entityType="case" entityId={id} />
    </CRMDetailShell>
  );
}
