export interface SupportCaseDetailData {
  id: string;
  subject?: string;
  status: string;
  priority: 'low' | 'medium' | 'high';
  description?: string;
  accountId?: string;
  accountName?: string;
  contactId?: string;
  contactName?: string;
  slaDeadline?: string;
  handoffStatus?: string;
  assignee?: string;
  activeSignalCount?: number;
}
