// Deal edit removed from wedge — redirects to sales hub
import { Redirect } from 'expo-router';
export default function DealEditRedirect() {
  return <Redirect href="/sales" />;
}
