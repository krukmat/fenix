import React from 'react';
import { useLocalSearchParams } from 'expo-router';
import { CRMCaseForm } from '../../../../../src/components/crm/CRMCaseForm';

function useRouteId(): string {
  const params = useLocalSearchParams<{ id: string | string[] }>();
  return Array.isArray(params.id) ? params.id[0] : params.id;
}

export default function CRMCaseEditScreen() {
  return <CRMCaseForm mode="edit" caseId={useRouteId()} />;
}
