export interface EvidenceSource {
  id: string;
  snippet: string;
  score: number;
  timestamp: string;
  title?: string;
  knowledge_item_id?: string;
  retrieval_method?: string;
  pii_redacted?: boolean;
}

export type CopilotConfidence = 'high' | 'medium' | 'low';
export type CopilotAnswerType = 'grounded_answer' | 'abstention';
export type CopilotAbstentionReason = 'insufficient_evidence' | 'irrelevant_evidence';

export interface CopilotEvidenceMeta {
  schemaVersion?: string;
  sourceCount?: number;
  dedupCount?: number;
  filteredCount?: number;
  confidence?: CopilotConfidence;
  warnings?: string[];
  retrievalMethodsUsed?: string[];
  builtAt?: string;
}

export type SSEMessage =
  | { type: 'token'; delta: string }
  | { type: 'evidence'; sources: EvidenceSource[]; meta?: CopilotEvidenceMeta }
  | { type: 'done'; answerType?: CopilotAnswerType; abstentionReason?: CopilotAbstentionReason }
  | { type: 'error'; message: string };

export interface SSEClient {
  close: () => void;
}
