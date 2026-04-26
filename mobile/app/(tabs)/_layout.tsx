// W2-T1 (mobile_wedge_harmonization_plan): Replace drawer with wedge bottom-tab navigation
// 5 primary tabs: Inbox, Support, Sales, Activity Log, Governance
// Legacy routes kept as hidden screens (href: null) for backward compatibility (W2-T2/W2-T3)

import React from 'react';
import { Tabs, Redirect } from 'expo-router';
import { MaterialCommunityIcons } from '@expo/vector-icons';
import type { ComponentProps } from 'react';
import { useAuthStore } from '../../src/stores/authStore';
import { useInbox } from '../../src/hooks/useWedge';
import { brandColors } from '../../src/theme/colors';
import { elevation, radius, spacing } from '../../src/theme/spacing';
import { typography } from '../../src/theme/typography';

// ─── Inbox badge ──────────────────────────────────────────────────────────────

function useInboxBadge(): number | undefined {
  const { data } = useInbox();
  if (!data) return undefined;
  const count = (data.approvals?.length ?? 0) + (data.handoffs?.length ?? 0);
  return count > 0 ? count : undefined;
}

type TabIconName = ComponentProps<typeof MaterialCommunityIcons>['name'];

const WEDGE_TAB_SCREENS: { name: string; title: string; icon: TabIconName; testID: string }[] = [
  { name: 'inbox/index', title: 'Inbox', icon: 'inbox-outline', testID: 'tab-inbox' },
  { name: 'support', title: 'Support', icon: 'lifebuoy', testID: 'tab-support' },
  { name: 'sales', title: 'Sales', icon: 'chart-line', testID: 'tab-sales' },
  { name: 'activity', title: 'Activity', icon: 'timeline-text-outline', testID: 'tab-activity' },
  { name: 'governance', title: 'Governance', icon: 'shield-account-outline', testID: 'tab-governance' },
];

const TAB_SCREEN_OPTIONS = {
  headerShown: true,
  headerShadowVisible: false,
  headerStyle: { backgroundColor: brandColors.surface },
  headerTintColor: brandColors.onBackground,
  headerTitleAlign: 'left' as const,
  headerTitleStyle: {
    color: brandColors.onBackground,
    ...typography.headingLG,
    fontSize: typography.headingMD.fontSize,
  },
  tabBarActiveTintColor: brandColors.primary,
  tabBarInactiveTintColor: brandColors.onSurfaceVariant,
  tabBarShowLabel: true,
  tabBarHideOnKeyboard: true,
  tabBarStyle: {
    backgroundColor: brandColors.surface,
    borderTopColor: brandColors.outlineVariant,
    borderTopWidth: 1,
    height: 68,
    paddingTop: 6,
    paddingBottom: spacing.sm,
    ...elevation.tabBar,
  },
  tabBarLabelStyle: {
    ...typography.labelMD,
    marginTop: 2,
    letterSpacing: 0.2,
  },
  tabBarItemStyle: { paddingVertical: 2 },
  tabBarBadgeStyle: {
    backgroundColor: brandColors.error,
    color: brandColors.onError,
    fontSize: 10,
    fontWeight: '700' as const,
    minWidth: 18,
    height: 18,
    borderRadius: radius.full,
  },
};

// ─── Root layout ──────────────────────────────────────────────────────────────

export default function TabsLayout() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const inboxBadge = useInboxBadge();

  if (!isAuthenticated) {
    return <Redirect href="/login" />;
  }

  return (
    <Tabs screenOptions={TAB_SCREEN_OPTIONS}>
      {/* ── Wedge tabs (visible) ── */}
      {WEDGE_TAB_SCREENS.map((screen) => (
        <Tabs.Screen
          key={screen.name}
          name={screen.name}
          options={{
            title: screen.title,
            tabBarIcon: ({ color, size }) => (
              <MaterialCommunityIcons name={screen.icon} color={color} size={size} />
            ),
            tabBarButtonTestID: screen.testID,
            ...(screen.name === 'inbox/index' ? { tabBarBadge: inboxBadge } : {}),
          }}
        />
      ))}

      {/* ── Legacy redirect shims — hidden from tab bar (W2-T2/W6-T2) ── */}
      <Tabs.Screen name="home" options={{ href: null, title: 'Home' }} />
      <Tabs.Screen name="accounts" options={{ href: null }} />
      <Tabs.Screen name="deals" options={{ href: null }} />
      <Tabs.Screen name="cases" options={{ href: null }} />
      <Tabs.Screen name="copilot/index" options={{ href: null }} />
      {/* Task Mobile P1.4 — T5: workflows hidden from tab bar (not a wedge tab) */}
      <Tabs.Screen name="workflows" options={{ href: null }} />
      {/* Task Mobile P1.4 — T1: contacts hidden from tab bar */}
      <Tabs.Screen name="contacts" options={{ href: null }} />
      {/* crm-dentro-governance: legacy CRM shim stays routeable but hidden from tab bar */}
      <Tabs.Screen name="crm" options={{ href: null }} />
    </Tabs>
  );
}
