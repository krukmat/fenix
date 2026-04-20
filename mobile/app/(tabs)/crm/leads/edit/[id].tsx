import React from 'react';
import { useLocalSearchParams } from 'expo-router';
import { CRMLeadForm } from '../../../../../src/components/crm/CRMLeadForm';

function useRouteId(): string {
  const params = useLocalSearchParams<{ id: string | string[] }>();
  return Array.isArray(params.id) ? params.id[0] : params.id;
}

export default function CRMLeadEditScreen() {
  return <CRMLeadForm mode="edit" leadId={useRouteId()} />;
}
