// W2-T1 (mobile_wedge_harmonization_plan): Replace drawer with wedge bottom-tab navigation
// 5 primary tabs: Inbox, Support, Sales, Activity Log, Governance
// Legacy routes kept as hidden screens (href: null) for backward compatibility (W2-T2/W2-T3)

import React from 'react';
import { Tabs, Redirect } from 'expo-router';
import { useAuthStore } from '../../src/stores/authStore';
import { useInbox } from '../../src/hooks/useWedge';

// ─── Inbox badge ──────────────────────────────────────────────────────────────

function useInboxBadge(): number | undefined {
  const { data } = useInbox();
  if (!data) return undefined;
  const count = (data.approvals?.length ?? 0) + (data.handoffs?.length ?? 0);
  return count > 0 ? count : undefined;
}

// ─── Root layout ──────────────────────────────────────────────────────────────

export default function TabsLayout() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const inboxBadge = useInboxBadge();

  if (!isAuthenticated) {
    return <Redirect href="/login" />;
  }

  return (
    <Tabs
      screenOptions={{
        headerShown: true,
        tabBarActiveTintColor: '#E53935',
        tabBarInactiveTintColor: '#757575',
        tabBarStyle: { backgroundColor: '#FFFFFF' },
      }}
    >
      {/* ── Wedge tabs (visible) ── */}
      <Tabs.Screen
        name="inbox/index"
        options={{
          title: 'Inbox',
          tabBarBadge: inboxBadge,
          tabBarIcon: () => null,
        }}
      />
      <Tabs.Screen
        name="support/index"
        options={{
          title: 'Support',
          tabBarIcon: () => null,
        }}
      />
      <Tabs.Screen
        name="sales/index"
        options={{
          title: 'Sales',
          tabBarIcon: () => null,
        }}
      />
      <Tabs.Screen
        name="activity/index"
        options={{
          title: 'Activity',
          tabBarIcon: () => null,
        }}
      />
      <Tabs.Screen
        name="governance/index"
        options={{
          title: 'Governance',
          tabBarIcon: () => null,
        }}
      />

      {/* ── Legacy routes — hidden from tab bar, still navigable (W2-T2/W2-T3) ── */}
      <Tabs.Screen name="home/index" options={{ href: null, title: 'Home' }} />
      <Tabs.Screen name="agents/index" options={{ href: null, title: 'Agents' }} />
      <Tabs.Screen name="accounts/index" options={{ href: null }} />
      <Tabs.Screen name="contacts/index" options={{ href: null }} />
      <Tabs.Screen name="deals/index" options={{ href: null }} />
      <Tabs.Screen name="cases/index" options={{ href: null }} />
      <Tabs.Screen name="copilot/index" options={{ href: null }} />
      <Tabs.Screen name="workflows/index" options={{ href: null }} />
      <Tabs.Screen name="crm/index" options={{ href: null }} />
    </Tabs>
  );
}
