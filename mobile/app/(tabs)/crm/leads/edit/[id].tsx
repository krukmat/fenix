import React from 'react';
import { CRMLeadForm } from '../../../../../src/components/crm/CRMLeadForm';
import { useRouteId } from '../../../../../src/hooks/useRouteId';

export default function CRMLeadEditScreen() {
  return <CRMLeadForm mode="edit" leadId={useRouteId()} />;
}
