// Deal creation removed from wedge — redirects to sales hub
import { Redirect } from 'expo-router';
export default function DealNewRedirect() {
  return <Redirect href="/sales" />;
}
