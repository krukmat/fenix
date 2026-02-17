// Task 4.2 — FR-300: Drawer Layout

import React from 'react';
import { Drawer } from 'expo-router/drawer';
import { Redirect, useRouter } from 'expo-router';
import { DrawerContentScrollView, DrawerItemList, DrawerContentComponentProps } from '@react-navigation/drawer';
import { View, Text, StyleSheet, TouchableOpacity } from 'react-native';
import { useTheme } from 'react-native-paper';
import { useAuthStore } from '../../src/stores/authStore';

function CustomDrawerContent(props: DrawerContentComponentProps) {
  const theme = useTheme();
  const router = useRouter();
  const { userId, logout } = useAuthStore();

  const handleLogout = async () => {
    await logout();
    router.replace('/(auth)/login');
  };

  return (
    <DrawerContentScrollView {...props}>
      <View style={[styles.header, { backgroundColor: theme.colors.primary }]}>
        <Text style={styles.headerTitle}>FenixCRM</Text>
        <Text style={styles.headerSubtitle}>
          {userId ? 'Logged in' : 'Guest'}
        </Text>
      </View>
      <DrawerItemList {...props} />
      <View style={styles.footer}>
        <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
          <Text style={[styles.logoutText, { color: theme.colors.error }]}>
            Logout
          </Text>
        </TouchableOpacity>
      </View>
    </DrawerContentScrollView>
  );
}

export default function TabsLayout() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  // Redirect to login if not authenticated
  if (!isAuthenticated) {
    return <Redirect href="/(auth)/login" />;
  }

  return (
    <Drawer drawerContent={(props) => <CustomDrawerContent {...props} />}>
      <Drawer.Screen 
        name="accounts/index" 
        options={{ 
          title: 'Accounts',
          drawerLabel: 'Accounts',
        }} 
      />
      <Drawer.Screen 
        name="contacts/index" 
        options={{ 
          title: 'Contacts',
          drawerLabel: 'Contacts',
        }} 
      />
      <Drawer.Screen 
        name="deals/index" 
        options={{ 
          title: 'Deals',
          drawerLabel: 'Deals',
        }} 
      />
      <Drawer.Screen 
        name="cases/index" 
        options={{ 
          title: 'Cases',
          drawerLabel: 'Cases',
        }} 
      />
      <Drawer.Screen 
        name="copilot/index" 
        options={{ 
          title: 'Copilot',
          drawerLabel: 'Copilot',
        }} 
      />
      <Drawer.Screen 
        name="agents/index" 
        options={{ 
          title: 'Agent Runs',
          drawerLabel: 'Agent Runs',
        }} 
      />
    </Drawer>
  );
}

const styles = StyleSheet.create({
  header: {
    padding: 20,
    marginBottom: 10,
  },
  headerTitle: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#FFFFFF',
  },
  headerSubtitle: {
    fontSize: 14,
    color: '#FFFFFF',
    opacity: 0.8,
  },
  footer: {
    padding: 20,
    borderTopWidth: 1,
    borderTopColor: '#E0E0E0',
    marginTop: 'auto',
  },
  logoutButton: {
    paddingVertical: 10,
  },
  logoutText: {
    fontSize: 16,
    fontWeight: '500',
  },
});
