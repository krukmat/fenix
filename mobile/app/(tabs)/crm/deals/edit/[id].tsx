import React from 'react';
import { CRMDealEditForm } from '../../../../../src/components/crm/CRMDealCreateForm';
import { useRouteId } from '../../../../../src/hooks/useRouteId';

export default function CRMDealEditScreen() {
  return <CRMDealEditForm dealId={useRouteId()} />;
}
