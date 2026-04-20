import React from 'react';
import { useLocalSearchParams } from 'expo-router';
import { CRMContactForm } from '../../../../../src/components/crm/CRMContactForm';

function useRouteId(): string {
  const params = useLocalSearchParams<{ id: string | string[] }>();
  return Array.isArray(params.id) ? params.id[0] : params.id;
}

export default function CRMContactEditScreen() {
  return <CRMContactForm mode="edit" contactId={useRouteId()} />;
}
