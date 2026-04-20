import React from 'react';
import { CRMAccountForm } from '../../../../../src/components/crm/CRMAccountForm';
import { useRouteId } from '../../../../../src/hooks/useRouteId';

export default function CRMAccountEditScreen() {
  return <CRMAccountForm mode="edit" accountId={useRouteId()} />;
}
