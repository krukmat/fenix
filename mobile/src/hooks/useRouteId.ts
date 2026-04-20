import { useLocalSearchParams } from 'expo-router';

export function useRouteId(): string {
  const params = useLocalSearchParams<{ id: string | string[] }>();
  return Array.isArray(params.id) ? params.id[0] : params.id;
}
