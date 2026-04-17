// Task Mobile P1.4 — Home screen with unified feed (signals + approvals)
// FR-300 (Home), FR-071 (approvals badge), UC-A5/A6
import React, { useCallback } from 'react';
import { useRouter } from 'expo-router';
import { useSignals, usePendingApprovals, useDismissSignal, useDecideApproval } from '../../../src/hooks/useAgentSpec';
import { HomeFeed } from '../../../src/components/home/HomeFeed';
import type { Signal } from '../../../src/services/api';

export default function HomeScreen() {
  const router = useRouter();

  const { data: signalData, isLoading: loadingSignals, refetch: refetchSignals } = useSignals();
  const { data: approvalsData, isLoading: loadingApprovals, refetch: refetchApprovals } = usePendingApprovals();
  const dismissSignal = useDismissSignal();
  const decideApproval = useDecideApproval();

  const signals = (signalData?.pages ?? []).flatMap((p) =>
    Array.isArray(p) ? p : ((p as { data?: Signal[] }).data ?? [])
  );
  const approvals = approvalsData ?? [];
  const pendingApprovalCount = Array.isArray(approvalsData) ? approvalsData.length : 0;

  const handleRefresh = useCallback(() => {
    refetchSignals();
    refetchApprovals();
  }, [refetchSignals, refetchApprovals]);

  const handleDismissSignal = useCallback(
    (id: string) => dismissSignal.mutate(id),
    [dismissSignal],
  );

  const handleApprove = useCallback(
    (id: string) => decideApproval.mutate({ id, decision: { decision: 'approve' } }),
    [decideApproval],
  );

  const handleReject = useCallback(
    (id: string, reason: string) => decideApproval.mutate({ id, decision: { decision: 'reject', reason } }),
    [decideApproval],
  );

  const handleSignalPress = useCallback(
    (signal: Signal) => router.push(`/home/signal/${signal.id}`),
    [router],
  );

  return (
    <HomeFeed
      signals={signals}
      approvals={approvals}
      loadingSignals={loadingSignals}
      loadingApprovals={loadingApprovals}
      onRefresh={handleRefresh}
      onDismissSignal={handleDismissSignal}
      onApprove={handleApprove}
      onReject={handleReject}
      onSignalPress={handleSignalPress}
      pendingApprovalCount={pendingApprovalCount}
    />
  );
}
