import React from 'react';
import { useLocalSearchParams } from 'expo-router';
import { CRMAccountForm } from '../../../../../src/components/crm/CRMAccountForm';

function useRouteId(): string {
  const params = useLocalSearchParams<{ id: string | string[] }>();
  return Array.isArray(params.id) ? params.id[0] : params.id;
}

export default function CRMAccountEditScreen() {
  return <CRMAccountForm mode="edit" accountId={useRouteId()} />;
}
