import React from 'react';
import { CRMCaseForm } from '../../../../../src/components/crm/CRMCaseForm';
import { useRouteId } from '../../../../../src/hooks/useRouteId';

export default function CRMCaseEditScreen() {
  return <CRMCaseForm mode="edit" caseId={useRouteId()} />;
}
