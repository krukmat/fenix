import React from 'react';
import { CRMContactForm } from '../../../../../src/components/crm/CRMContactForm';
import { useRouteId } from '../../../../../src/hooks/useRouteId';

export default function CRMContactEditScreen() {
  return <CRMContactForm mode="edit" contactId={useRouteId()} />;
}
