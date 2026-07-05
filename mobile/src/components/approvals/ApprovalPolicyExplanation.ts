import type { ApprovalRequest } from '../../services/api';
import type { PolicyExplanation } from './ApprovalCardBlocks';

function readString(record: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === 'string' && value.trim()) {
      return value.trim();
    }
  }
  return undefined;
}

function readStringList(record: Record<string, unknown>, ...keys: string[]): string[] {
  for (const key of keys) {
    const value = record[key];
    if (Array.isArray(value)) {
      const items = value.filter((item): item is string => typeof item === 'string' && item.trim().length > 0);
      if (items.length > 0) {
        return items.map((item) => item.trim());
      }
    }
  }
  return [];
}

export function getPolicyExplanation(payload: ApprovalRequest['payload']): PolicyExplanation | null {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return null;
  }

  const record = payload as Record<string, unknown>;
  const policyId = readString(record, 'policy_id', 'policyId');
  const policyType = readString(record, 'policy_type', 'policyType');
  const decision = readString(record, 'decision', 'policy_decision', 'policyDecision');
  const reason = readString(record, 'reason', 'policy_reason', 'policyReason', 'policy_explanation', 'policyExplanation');
  const reasonCodes = readStringList(record, 'reason_codes', 'reasonCodes', 'rule_ids', 'ruleIds');

  const lines: string[] = [];
  if (policyType) lines.push(`Type: ${policyType}`);
  if (policyId) lines.push(`Policy: ${policyId}`);
  if (decision) lines.push(`Decision: ${decision}`);
  if (reason) lines.push(reason);
  if (reasonCodes.length > 0) lines.push(`Rules: ${reasonCodes.join(', ')}`);

  return lines.length > 0
    ? { title: 'Policy explanation', lines }
    : null;
}
