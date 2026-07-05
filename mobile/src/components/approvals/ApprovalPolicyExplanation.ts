import type { ApprovalRequest } from '../../services/api';
import type { PolicyExplanation } from './ApprovalCardBlocks';
import { readString, readStringArray } from '../../utils/recordReaders';

interface PolicyFields {
  policyId?: string;
  policyType?: string;
  decision?: string;
  reason?: string;
  reasonCodes: string[];
}

function readPolicyFields(record: Record<string, unknown>): PolicyFields {
  return {
    policyId: readString(record, 'policy_id', 'policyId'),
    policyType: readString(record, 'policy_type', 'policyType'),
    decision: readString(record, 'decision', 'policy_decision', 'policyDecision'),
    reason: readString(record, 'reason', 'policy_reason', 'policyReason', 'policy_explanation', 'policyExplanation'),
    reasonCodes: readStringArray(record, 'reason_codes', 'reasonCodes', 'rule_ids', 'ruleIds') ?? [],
  };
}

function buildPolicyLines(fields: PolicyFields): string[] {
  const lines: string[] = [];
  if (fields.policyType) lines.push(`Type: ${fields.policyType}`);
  if (fields.policyId) lines.push(`Policy: ${fields.policyId}`);
  if (fields.decision) lines.push(`Decision: ${fields.decision}`);
  if (fields.reason) lines.push(fields.reason);
  if (fields.reasonCodes.length > 0) lines.push(`Rules: ${fields.reasonCodes.join(', ')}`);
  return lines;
}

export function getPolicyExplanation(payload: ApprovalRequest['payload']): PolicyExplanation | null {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
    return null;
  }

  const lines = buildPolicyLines(readPolicyFields(payload as Record<string, unknown>));

  return lines.length > 0
    ? { title: 'Policy explanation', lines }
    : null;
}
