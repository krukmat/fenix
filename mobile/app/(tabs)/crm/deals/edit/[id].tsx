import React from 'react';
import { useLocalSearchParams } from 'expo-router';
import { CRMDealEditForm } from '../../../../../src/components/crm/CRMDealCreateForm';

function useRouteId(): string {
  const params = useLocalSearchParams<{ id: string | string[] }>();
  return Array.isArray(params.id) ? params.id[0] : params.id;
}

export default function CRMDealEditScreen() {
  return <CRMDealEditForm dealId={useRouteId()} />;
}
