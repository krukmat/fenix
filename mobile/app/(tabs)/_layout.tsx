// Task 4.2 — FR-300: Drawer Layout
// Task Mobile P1.5 — FR-300/FR-071: 5-item drawer (Home, CRM, Copilot, Workflows, Activity Log)

import React, { useState } from 'react';
import { Drawer } from 'expo-router/drawer';
import { Redirect, useRouter } from 'expo-router';
import { DrawerContentScrollView, DrawerContentComponentProps } from '@react-navigation/drawer';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useAuthStore } from '../../src/stores/authStore';
import { usePendingApprovals } from '../../src/hooks/useAgentSpec';

// ─── Primitives ───────────────────────────────────────────────────────────────

function DrawerNavItem({
  label,
  testID,
  onPress,
  badge,
}: {
  label: string;
  testID: string;
  onPress: () => void;
  badge?: number;
}) {
  return (
    <TouchableOpacity testID={testID} style={styles.drawerItem} onPress={onPress}>
      <View style={styles.drawerItemRow}>
        <Text style={styles.drawerItemText}>{label}</Text>
        {badge != null && badge > 0 && (
          <View style={styles.badge} testID={`${testID}-badge`}>
            <Text style={styles.badgeText}>{badge > 99 ? '99+' : badge}</Text>
          </View>
        )}
      </View>
    </TouchableOpacity>
  );
}

function DrawerSubItem({
  label,
  testID,
  onPress,
}: {
  label: string;
  testID: string;
  onPress: () => void;
}) {
  return (
    <TouchableOpacity testID={testID} style={styles.drawerSubItem} onPress={onPress}>
      <Text style={styles.drawerSubItemText}>{label}</Text>
    </TouchableOpacity>
  );
}

// ─── CRM Section (colapsable) ─────────────────────────────────────────────────

function CRMSection({
  navigation,
}: {
  navigation: DrawerContentComponentProps['navigation'];
}) {
  const [expanded, setExpanded] = useState(false);
  return (
    <>
      <TouchableOpacity
        testID="drawer-crm-tab"
        style={styles.drawerItem}
        onPress={() => setExpanded((v) => !v)}
      >
        <View style={styles.drawerItemRow}>
          <Text style={styles.drawerItemText}>CRM</Text>
          <Text style={styles.chevron}>{expanded ? '▲' : '▼'}</Text>
        </View>
      </TouchableOpacity>
      {expanded && (
        <View testID="drawer-crm-submenu">
          <DrawerSubItem
            label="Accounts"
            testID="drawer-crm-accounts"
            onPress={() => navigation.navigate('crm/accounts/index')}
          />
          <DrawerSubItem
            label="Contacts"
            testID="drawer-crm-contacts"
            onPress={() => navigation.navigate('crm/contacts/index')}
          />
          <DrawerSubItem
            label="Deals"
            testID="drawer-crm-deals"
            onPress={() => navigation.navigate('crm/deals/index')}
          />
          <DrawerSubItem
            label="Cases"
            testID="drawer-crm-cases"
            onPress={() => navigation.navigate('crm/cases/index')}
          />
        </View>
      )}
    </>
  );
}

// ─── Custom Drawer Content ────────────────────────────────────────────────────

function CustomDrawerContent(props: DrawerContentComponentProps) {
  const theme = useTheme();
  const router = useRouter();
  const { userId, logout } = useAuthStore();
  const { data: pendingApprovals } = usePendingApprovals();
  const pendingCount = pendingApprovals?.length ?? 0;

  const handleLogout = async () => {
    await logout();
    router.replace('/(auth)/login');
  };

  return (
    <DrawerContentScrollView {...props} testID="drawer-content">
      <View style={[styles.header, { backgroundColor: theme.colors.primary }]}>
        <Text style={styles.headerTitle}>FenixCRM</Text>
        <Text style={styles.headerSubtitle}>{userId ? 'Logged in' : 'Guest'}</Text>
      </View>

      {/* 1 — Home (with approvals badge) */}
      <DrawerNavItem
        label="Home"
        testID="drawer-home-tab"
        badge={pendingCount}
        onPress={() => props.navigation.navigate('home/index')}
      />

      {/* 2 — CRM (collapsible submenu) */}
      <CRMSection navigation={props.navigation} />

      {/* 3 — Copilot */}
      <DrawerNavItem
        label="Copilot"
        testID="drawer-copilot-tab"
        onPress={() => props.navigation.navigate('copilot/index')}
      />

      {/* 4 — Workflows */}
      <DrawerNavItem
        label="Workflows"
        testID="drawer-workflows-tab"
        onPress={() => props.navigation.navigate('workflows/index')}
      />

      {/* 5 — Activity Log (renamed from Agent Runs) */}
      <DrawerNavItem
        label="Activity Log"
        testID="drawer-activity-tab"
        onPress={() => props.navigation.navigate('activity')}
      />

      <View style={styles.footer}>
        <TouchableOpacity testID="drawer-logout-button" style={styles.logoutButton} onPress={handleLogout}>
          <Text style={[styles.logoutText, { color: theme.colors.error }]}>Logout</Text>
        </TouchableOpacity>
      </View>
    </DrawerContentScrollView>
  );
}

// ─── Root layout ──────────────────────────────────────────────────────────────

export default function TabsLayout() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  if (!isAuthenticated) {
    return <Redirect href="/(auth)/login" />;
  }

  return (
    <Drawer
      drawerContent={(props) => <CustomDrawerContent {...props} />}
      screenOptions={({ navigation }) => ({
        headerLeft: () => (
          <TouchableOpacity
            testID="drawer-open-button"
            style={styles.drawerOpenButton}
            onPress={() => navigation.openDrawer()}
          >
            <Text style={styles.drawerOpenButtonText}>☰</Text>
          </TouchableOpacity>
        ),
      })}
    >
      {/* Home */}
      <Drawer.Screen name="home" options={{ title: 'Home', drawerItemStyle: { display: 'none' } }} />

      {/* CRM hub + sub-screens */}
      <Drawer.Screen name="crm" options={{ title: 'CRM', drawerItemStyle: { display: 'none' } }} />

      {/* Copilot */}
      <Drawer.Screen name="copilot/index" options={{ title: 'Copilot', drawerItemStyle: { display: 'none' } }} />

      {/* Workflows */}
      <Drawer.Screen name="workflows" options={{ title: 'Workflows', drawerItemStyle: { display: 'none' } }} />

      {/* Activity Log */}
      <Drawer.Screen name="activity" options={{ title: 'Activity Log', drawerItemStyle: { display: 'none' } }} />

      {/* Legacy routes — kept for backward compat, hidden from drawer */}
      <Drawer.Screen name="accounts/index" options={{ drawerItemStyle: { display: 'none' } }} />
      <Drawer.Screen name="contacts/index" options={{ drawerItemStyle: { display: 'none' } }} />
      <Drawer.Screen name="deals/index" options={{ drawerItemStyle: { display: 'none' } }} />
      <Drawer.Screen name="cases/index" options={{ drawerItemStyle: { display: 'none' } }} />
      <Drawer.Screen name="agents" options={{ drawerItemStyle: { display: 'none' } }} />
    </Drawer>
  );
}

// ─── Styles ───────────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  header: { padding: 20, marginBottom: 10 },
  headerTitle: { fontSize: 24, fontWeight: 'bold', color: '#FFFFFF' },
  headerSubtitle: { fontSize: 14, color: '#FFFFFF', opacity: 0.8 },
  footer: {
    padding: 20,
    borderTopWidth: 1,
    borderTopColor: '#E0E0E0',
    marginTop: 'auto',
  },
  logoutButton: { paddingVertical: 10 },
  logoutText: { fontSize: 16, fontWeight: '500' },
  drawerItem: { paddingHorizontal: 20, paddingVertical: 12 },
  drawerItemRow: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between' },
  drawerItemText: { fontSize: 16 },
  chevron: { fontSize: 12, color: '#757575' },
  drawerSubItem: { paddingHorizontal: 36, paddingVertical: 10 },
  drawerSubItemText: { fontSize: 15, color: '#555555' },
  badge: {
    backgroundColor: '#E53935',
    borderRadius: 10,
    minWidth: 20,
    height: 20,
    paddingHorizontal: 5,
    justifyContent: 'center',
    alignItems: 'center',
  },
  badgeText: { color: '#FFFFFF', fontSize: 11, fontWeight: '700' },
  drawerOpenButton: { paddingHorizontal: 14, paddingVertical: 8 },
  drawerOpenButtonText: { fontSize: 20, fontWeight: '700' },
});
