// Task Mobile P1.4 — FR-300/FR-071, UC-A5/A7: Home feed (signals + approvals)

import React, { useCallback } from 'react';
import { Stack, useRouter } from 'expo-router';
import { HomeFeed } from '../../../src/components/home/HomeFeed';
import { useSignals, useDismissSignal, usePendingApprovals, useDecideApproval } from '../../../src/hooks/useAgentSpec';
import type { Signal } from '../../../src/services/api';

export default function HomeScreen() {
  const router = useRouter();

  const {
    data: signalsData,
    isLoading: loadingSignals,
    refetch: refetchSignals,
  } = useSignals({ status: 'active' });

  const {
    data: approvalsData,
    isLoading: loadingApprovals,
    refetch: refetchApprovals,
  } = usePendingApprovals();

  const dismissMutation = useDismissSignal();
  const decideMutation = useDecideApproval();

  const signals = signalsData?.pages.flat() ?? [];
  const approvals = approvalsData ?? [];
  const pendingCount = approvals.length;

  const handleRefresh = useCallback(() => {
    refetchSignals();
    refetchApprovals();
  }, [refetchSignals, refetchApprovals]);

  const handleDismissSignal = useCallback(
    (id: string) => {
      dismissMutation.mutate(id);
    },
    [dismissMutation]
  );

  const handleApprove = useCallback(
    (id: string) => {
      decideMutation.mutate({ id, decision: { decision: 'approve' } });
    },
    [decideMutation]
  );

  const handleDeny = useCallback(
    (id: string, reason: string) => {
      // W1-T1: 'deny' replaced by 'reject' per normalized approval contract
      decideMutation.mutate({ id, decision: { decision: 'reject', reason } });
    },
    [decideMutation]
  );

  const handleSignalPress = useCallback(
    (signal: Signal) => {
      router.push(`/home/signal/${signal.id}`);
    },
    [router]
  );

  return (
    <>
      <Stack.Screen options={{ title: 'Home' }} />
      <HomeFeed
        signals={signals}
        approvals={approvals}
        loadingSignals={loadingSignals}
        loadingApprovals={loadingApprovals}
        onRefresh={handleRefresh}
        onDismissSignal={handleDismissSignal}
        onApprove={handleApprove}
        onDeny={handleDeny}
        onSignalPress={handleSignalPress}
        pendingApprovalCount={pendingCount}
        testIDPrefix="home-feed"
      />
    </>
  );
}
