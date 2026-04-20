import React, { useCallback, useMemo, useState } from 'react';
import { Alert } from 'react-native';
import { useRouter } from 'expo-router';
import { CRMListScreen } from './CRMListScreen';
import { CRMReadOnlyRow, asText, unwrapDataArray } from './CoreCRMReadOnly';
import type { CRMAccount, CRMCase, CRMContact, CRMDeal, CRMLead } from '../../services/api';
import {
  normalizeCRMAccount,
  normalizeCRMCase,
  normalizeCRMContact,
  normalizeCRMDeal,
  normalizeCRMLead,
} from '../../services/api';
import {
  useAccounts, useCases, useContacts, useDeals, useLeads,
  useDeleteAccount, useDeleteCase, useDeleteContact, useDeleteDeal, useDeleteLead,
} from '../../hooks/useCRM';

type ListHookResult = ReturnType<typeof useAccounts>;
type EntityName = 'accounts' | 'contacts' | 'leads' | 'deals' | 'cases';
type EntityItem = CRMAccount | CRMContact | CRMLead | CRMDeal | CRMCase;
type ListFrameProps<T extends EntityItem> = {
  entity: EntityName;
  query: ListHookResult;
  items: T[];
  emptyTitle: string;
  primaryActionLabel?: string;
  onPrimaryAction?: () => void;
  onRowEdit?: (id: string) => void;
  deleteFn: (id: string) => Promise<unknown>;
};

function listItems<T extends EntityItem>(data: ListHookResult['data'], normalize: (raw: unknown) => T): T[] {
  return (data?.pages ?? []).flatMap((page) => unwrapDataArray<unknown>(page).map(normalize));
}

function leadTitle(lead: CRMLead): string {
  const name = typeof lead.metadata.name === 'string' ? lead.metadata.name : '';
  return name || `Lead ${lead.id}`;
}

function rowText(entity: EntityName, item: EntityItem) {
  if (entity === 'accounts') return { title: (item as CRMAccount).name, subtitle: (item as CRMAccount).industry };
  if (entity === 'contacts') {
    const contact = item as CRMContact;
    const title = [contact.firstName, contact.lastName].filter(Boolean).join(' ') || contact.email || 'Unknown Contact';
    return { title, subtitle: contact.title, meta: contact.email };
  }
  if (entity === 'leads') {
    const lead = item as CRMLead;
    return { title: leadTitle(lead), subtitle: lead.status, meta: lead.source };
  }
  if (entity === 'deals') {
    const deal = item as CRMDeal;
    return { title: deal.title, subtitle: deal.status, meta: deal.amount ? `$${deal.amount.toLocaleString()}` : undefined };
  }
  const caseItem = item as CRMCase;
  return { title: caseItem.subject, subtitle: caseItem.status, meta: caseItem.priority };
}

function matchesSearch(item: EntityItem, query: string): boolean {
  const values = Object.values(item).map((value) => String(value ?? '').toLowerCase());
  return values.some((value) => value.includes(query));
}

function useSelectionState(filtered: EntityItem[]) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const toggleSelected = useCallback((id: string) => {
    setSelectedIds((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);
  const selectAllVisible = useCallback(() => {
    setSelectedIds((current) => {
      const next = new Set(current);
      filtered.forEach((item) => next.add(item.id));
      return next;
    });
  }, [filtered]);
  const clearSelection = useCallback(() => setSelectedIds(new Set()), []);
  return { selectedIds, setSelectedIds, toggleSelected, selectAllVisible, clearSelection };
}

function useBulkDelete(
  selectedIds: Set<string>,
  setSelectedIds: (s: Set<string>) => void,
  deleteFn: (id: string) => Promise<unknown>,
) {
  const [bulkDeletePending, setBulkDeletePending] = useState(false);
  const handleBulkDelete = useCallback(() => {
    const ids = Array.from(selectedIds);
    Alert.alert(
      'Delete selected',
      `Delete ${ids.length} item${ids.length !== 1 ? 's' : ''}?`,
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            setBulkDeletePending(true);
            const results = await Promise.allSettled(ids.map((id) => deleteFn(id)));
            setBulkDeletePending(false);
            const failedIds = ids.filter((_, i) => results[i].status === 'rejected');
            if (failedIds.length === 0) {
              setSelectedIds(new Set());
            } else {
              setSelectedIds(new Set(failedIds));
              Alert.alert('Some deletes failed', `${failedIds.length} item${failedIds.length !== 1 ? 's' : ''} could not be deleted.`);
            }
          },
        },
      ],
    );
  }, [selectedIds, deleteFn, setSelectedIds]);
  return { bulkDeletePending, handleBulkDelete };
}

function EntityListFrame<T extends EntityItem>({
  entity, query, items, emptyTitle, primaryActionLabel, onPrimaryAction, onRowEdit, deleteFn,
}: ListFrameProps<T>) {
  const router = useRouter();
  const [search, setSearch] = useState('');
  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return q ? items.filter((item) => matchesSearch(item, q)) : items;
  }, [items, search]);
  const { selectedIds, setSelectedIds, toggleSelected, selectAllVisible, clearSelection } = useSelectionState(filtered);
  const { bulkDeletePending, handleBulkDelete } = useBulkDelete(selectedIds, setSelectedIds, deleteFn);
  const renderItem = useCallback(
    ({ item, index }: { item: EntityItem; index: number }) => {
      const text = rowText(entity, item);
      return (
        <CRMReadOnlyRow
          title={asText(text.title, 'Untitled')}
          subtitle={text.subtitle}
          meta={text.meta}
          testID={`crm-${entity}-item-${index}`}
          onPress={() => router.push(`/crm/${entity}/${item.id}`)}
        />
      );
    },
    [entity, router],
  );
  return (
    <CRMListScreen
      data={filtered} loading={query.isLoading}
      error={query.error ? query.error.message : null}
      onRefresh={() => query.refetch()} searchValue={search} onSearchChange={setSearch}
      renderItem={renderItem} emptyTitle={emptyTitle} emptySubtitle="No CRM records available"
      testIDPrefix={`crm-${entity}`} hasData={items.length > 0}
      loadingMore={query.isFetchingNextPage} hasMore={query.hasNextPage ?? false}
      onEndReached={() => { if (query.hasNextPage && !query.isFetchingNextPage) query.fetchNextPage(); }}
      isRefreshing={query.isRefetching} onRetry={() => query.refetch()}
      primaryActionLabel={primaryActionLabel} onPrimaryAction={onPrimaryAction}
      selectedIds={selectedIds} onToggleSelect={toggleSelected}
      onSelectAllVisible={selectAllVisible} onClearSelection={clearSelection}
      onRowEdit={onRowEdit} onBulkDelete={handleBulkDelete} bulkDeletePending={bulkDeletePending}
    />
  );
}

export function CoreCRMAccountsList() {
  const query = useAccounts();
  const router = useRouter();
  const { mutateAsync: deleteAccount } = useDeleteAccount();
  return (
    <EntityListFrame
      entity="accounts"
      query={query}
      items={listItems(query.data, normalizeCRMAccount)}
      emptyTitle="No accounts found"
      primaryActionLabel="New Account"
      onPrimaryAction={() => router.push('/crm/accounts/new')}
      onRowEdit={(id) => router.push(`/crm/accounts/edit/${id}`)}
      deleteFn={deleteAccount}
    />
  );
}

export function CoreCRMContactsList() {
  const query = useContacts();
  const router = useRouter();
  const { mutateAsync: deleteContact } = useDeleteContact();
  return (
    <EntityListFrame
      entity="contacts"
      query={query}
      items={listItems(query.data, normalizeCRMContact)}
      emptyTitle="No contacts found"
      primaryActionLabel="New Contact"
      onPrimaryAction={() => router.push('/crm/contacts/new')}
      onRowEdit={(id) => router.push(`/crm/contacts/edit/${id}`)}
      deleteFn={deleteContact}
    />
  );
}

export function CoreCRMLeadsList() {
  const query = useLeads();
  const router = useRouter();
  const { mutateAsync: deleteLead } = useDeleteLead();
  return (
    <EntityListFrame
      entity="leads"
      query={query}
      items={listItems(query.data, normalizeCRMLead)}
      emptyTitle="No leads found"
      primaryActionLabel="New Lead"
      onPrimaryAction={() => router.push('/crm/leads/new')}
      onRowEdit={(id) => router.push(`/crm/leads/edit/${id}`)}
      deleteFn={deleteLead}
    />
  );
}

export function CoreCRMDealsList() {
  const query = useDeals();
  const router = useRouter();
  const { mutateAsync: deleteDeal } = useDeleteDeal();
  return (
    <EntityListFrame
      entity="deals"
      query={query}
      items={listItems(query.data, normalizeCRMDeal)}
      emptyTitle="No deals found"
      primaryActionLabel="New Deal"
      onPrimaryAction={() => router.push('/crm/deals/new')}
      onRowEdit={(id) => router.push(`/crm/deals/edit/${id}`)}
      deleteFn={deleteDeal}
    />
  );
}

export function CoreCRMCasesList() {
  const query = useCases();
  const router = useRouter();
  const { mutateAsync: deleteCase } = useDeleteCase();
  return (
    <EntityListFrame
      entity="cases"
      query={query}
      items={listItems(query.data, normalizeCRMCase)}
      emptyTitle="No cases found"
      primaryActionLabel="New Case"
      onPrimaryAction={() => router.push('/crm/cases/new')}
      onRowEdit={(id) => router.push(`/crm/cases/edit/${id}`)}
      deleteFn={deleteCase}
    />
  );
}
